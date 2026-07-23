package handler

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	accessTokenCookieName      = "access_token"
	refreshTokenCookieName     = "refresh_token"
	tokenCookieEncodingPrefix  = "z1"
	tokenCookieChunkSize       = 2800
	maxTokenCookieChunks       = 8
	maxDecodedTokenCookieBytes = 128 << 10
)

func setTokenCookies(w http.ResponseWriter, r *http.Request, accessToken, refreshToken string, expiresAt time.Time) error {
	if expiresAt.IsZero() || expiresAt.Before(time.Now()) {
		expiresAt = time.Now().Add(24 * time.Hour)
	}

	accessParts, err := encodeTokenCookie(accessToken)
	if err != nil {
		return fmt.Errorf("encode access token cookie: %w", err)
	}

	var refreshParts []string
	if strings.TrimSpace(refreshToken) != "" {
		refreshParts, err = encodeTokenCookie(refreshToken)
		if err != nil {
			return fmt.Errorf("encode refresh token cookie: %w", err)
		}
	}

	secure := strings.HasPrefix(requestOrigin(r), "https")
	writeTokenCookieParts(w, accessTokenCookieName, accessParts, expiresAt, secure)
	if len(refreshParts) == 0 {
		clearTokenCookie(w, refreshTokenCookieName, secure)
	} else {
		writeTokenCookieParts(w, refreshTokenCookieName, refreshParts, expiresAt.AddDate(0, 1, 0), secure)
	}
	return nil
}

func ReadAccessTokenCookie(r *http.Request) (string, error) {
	return readTokenCookie(r, accessTokenCookieName)
}

func readRefreshTokenCookie(r *http.Request) (string, error) {
	return readTokenCookie(r, refreshTokenCookieName)
}

func readTokenCookie(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err == http.ErrNoCookie {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	value := strings.TrimSpace(cookie.Value)
	if value == "" || !strings.HasPrefix(value, tokenCookieEncodingPrefix+".") {
		return value, nil
	}

	header := strings.SplitN(value, ".", 3)
	if len(header) != 3 {
		return "", fmt.Errorf("invalid %s cookie header", name)
	}
	partCount, err := strconv.Atoi(header[1])
	if err != nil || partCount < 1 || partCount > maxTokenCookieChunks {
		return "", fmt.Errorf("invalid %s cookie part count", name)
	}

	var encoded strings.Builder
	encoded.Grow(partCount * tokenCookieChunkSize)
	encoded.WriteString(header[2])
	for i := 1; i < partCount; i++ {
		part, err := r.Cookie(tokenCookiePartName(name, i))
		if err != nil || part.Value == "" {
			return "", fmt.Errorf("missing %s cookie part %d", name, i)
		}
		encoded.WriteString(part.Value)
	}

	compressed, err := base64.RawURLEncoding.DecodeString(encoded.String())
	if err != nil {
		return "", fmt.Errorf("decode %s cookie: %w", name, err)
	}

	reader := flate.NewReader(bytes.NewReader(compressed))
	defer reader.Close()

	decoded, err := io.ReadAll(io.LimitReader(reader, maxDecodedTokenCookieBytes+1))
	if err != nil {
		return "", fmt.Errorf("decompress %s cookie: %w", name, err)
	}
	if len(decoded) > maxDecodedTokenCookieBytes {
		return "", fmt.Errorf("%s cookie exceeds decoded size limit", name)
	}
	return string(decoded), nil
}

func encodeTokenCookie(token string) ([]string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("token is empty")
	}
	if len(token) > maxDecodedTokenCookieBytes {
		return nil, fmt.Errorf("token exceeds size limit")
	}

	var compressed bytes.Buffer
	writer, err := flate.NewWriter(&compressed, flate.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err = writer.Write([]byte(token)); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err = writer.Close(); err != nil {
		return nil, err
	}

	encoded := base64.RawURLEncoding.EncodeToString(compressed.Bytes())
	partCount := (len(encoded) + tokenCookieChunkSize - 1) / tokenCookieChunkSize
	if partCount < 1 || partCount > maxTokenCookieChunks {
		return nil, fmt.Errorf("encoded token requires %d cookie parts", partCount)
	}

	parts := make([]string, partCount)
	for i := range parts {
		start := i * tokenCookieChunkSize
		end := min(start+tokenCookieChunkSize, len(encoded))
		parts[i] = encoded[start:end]
	}
	parts[0] = tokenCookieEncodingPrefix + "." + strconv.Itoa(partCount) + "." + parts[0]
	return parts, nil
}

func writeTokenCookieParts(w http.ResponseWriter, name string, parts []string, expiresAt time.Time, secure bool) {
	for i, value := range parts {
		http.SetCookie(w, authCookie(tokenCookiePartName(name, i), value, expiresAt, secure, 0))
	}
	for i := len(parts); i < maxTokenCookieChunks; i++ {
		http.SetCookie(w, authCookie(tokenCookiePartName(name, i), "", time.Now().Add(-time.Hour), secure, -1))
	}
}

func clearTokenCookies(w http.ResponseWriter, r *http.Request) {
	secure := strings.HasPrefix(requestOrigin(r), "https")
	clearTokenCookie(w, accessTokenCookieName, secure)
	clearTokenCookie(w, refreshTokenCookieName, secure)
}

func clearTokenCookie(w http.ResponseWriter, name string, secure bool) {
	expiresAt := time.Now().Add(-time.Hour)
	for i := 0; i < maxTokenCookieChunks; i++ {
		http.SetCookie(w, authCookie(tokenCookiePartName(name, i), "", expiresAt, secure, -1))
	}
}

func tokenCookiePartName(name string, index int) string {
	if index == 0 {
		return name
	}
	return name + "_" + strconv.Itoa(index)
}

func authCookie(name, value string, expiresAt time.Time, secure bool, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Expires:  expiresAt,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	}
}
