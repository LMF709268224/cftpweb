package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"candbff/config"
)

const maxPaymentReturnURLLength = 2048

func validatePaymentReturnURLs(r *http.Request, successURL, cancelURL string) error {
	allowedOrigins := paymentReturnAllowedOrigins(r)
	if err := validatePaymentReturnURL(successURL, allowedOrigins); err != nil {
		return fmt.Errorf("success_url %w", err)
	}
	if err := validatePaymentReturnURL(cancelURL, allowedOrigins); err != nil {
		return fmt.Errorf("cancel_url %w", err)
	}
	return nil
}

func validatePaymentReturnURL(raw string, allowedOrigins map[string]struct{}) error {
	if raw == "" {
		return fmt.Errorf("is required")
	}
	if len(raw) > maxPaymentReturnURLLength {
		return fmt.Errorf("is too long")
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.Opaque != "" {
		return fmt.Errorf("must be an absolute HTTP(S) URL")
	}
	if parsed.User != nil {
		return fmt.Errorf("must not contain user information")
	}

	origin, ok := paymentURLOrigin(parsed)
	if !ok {
		return fmt.Errorf("must be an absolute HTTP(S) URL")
	}
	if _, allowed := allowedOrigins[origin]; !allowed {
		return fmt.Errorf("origin is not allowed")
	}
	return nil
}

func paymentReturnAllowedOrigins(r *http.Request) map[string]struct{} {
	allowed := make(map[string]struct{})
	configured := strings.TrimSpace(os.Getenv(config.EnvPaymentReturnAllowedOrigins))
	if configured != "" {
		for _, value := range strings.Split(configured, ",") {
			if origin, ok := configuredPaymentOrigin(value); ok {
				allowed[origin] = struct{}{}
			}
		}
		return allowed
	}

	if origin, ok := requestPaymentOrigin(r); ok {
		allowed[origin] = struct{}{}
	}
	return allowed
}

func configuredPaymentOrigin(raw string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.User != nil || parsed.Path != "" && parsed.Path != "/" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false
	}
	return paymentURLOrigin(parsed)
}

func requestPaymentOrigin(r *http.Request) (string, bool) {
	if r == nil || strings.TrimSpace(r.Host) == "" {
		return "", false
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]); forwarded == "http" || forwarded == "https" {
		scheme = forwarded
	}

	parsed, err := url.Parse(scheme + "://" + strings.TrimSpace(r.Host))
	if err != nil || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false
	}
	return paymentURLOrigin(parsed)
}

func paymentURLOrigin(parsed *url.URL) (string, bool) {
	if parsed == nil || parsed.User != nil {
		return "", false
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", false
	}
	host := strings.ToLower(parsed.Host)
	if host == "" || parsed.Hostname() == "" {
		return "", false
	}
	return scheme + "://" + host, true
}
