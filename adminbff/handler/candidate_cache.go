package handler

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	gmidpb "github.com/afnandelfin620-star/cftptest/cftp/gmid"
	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

const (
	candidateProfileRefreshInterval = 30 * time.Minute
	candidateProfileRefreshTimeout  = 2 * time.Minute
	candidateProfileLookupTimeout   = 30 * time.Second
	candidateProfileGmidTimeout     = 3 * time.Second
)

type CandidateProfileCache struct {
	gmid gmidpb.MidServiceClient

	mu       sync.RWMutex
	names    map[string]string
	inFlight map[string]struct{}
}

func NewCandidateProfileCache(gmid gmidpb.MidServiceClient) *CandidateProfileCache {
	return &CandidateProfileCache{
		gmid:     gmid,
		names:    map[string]string{},
		inFlight: map[string]struct{}{},
	}
}

func (c *CandidateProfileCache) Start(ctx context.Context) {
	if c == nil || c.gmid == nil {
		return
	}

	go func() {
		c.refreshWithTimeout(ctx)

		ticker := time.NewTicker(candidateProfileRefreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.refreshWithTimeout(ctx)
			}
		}
	}()
}

func (c *CandidateProfileCache) NameOrQueue(candidateULID string) string {
	candidateULID = strings.TrimSpace(candidateULID)
	if candidateULID == "" || c == nil {
		return ""
	}

	c.mu.RLock()
	name := c.names[candidateULID]
	c.mu.RUnlock()
	if name != "" {
		return name
	}

	c.enqueue(candidateULID)
	return ""
}

func (c *CandidateProfileCache) refreshWithTimeout(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, candidateProfileRefreshTimeout)
	defer cancel()

	if err := c.Refresh(ctx); err != nil {
		slog.Warn("candidate profile cache refresh failed; keeping previous cache", "error", err)
	}
}

func (c *CandidateProfileCache) Refresh(ctx context.Context) error {
	users, err := casdoorsdk.GetUsers()
	if err != nil {
		return err
	}

	names := make(map[string]string, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		name := strings.TrimSpace(user.Name)
		userUUID := strings.TrimSpace(user.Id)
		if name == "" || userUUID == "" {
			continue
		}

		lookupCtx, cancel := context.WithTimeout(ctx, candidateProfileGmidTimeout)
		resp, err := c.gmid.GetUlidByUUID(lookupCtx, &gmidpb.GetUlidByUUIDRequest{
			UserUuid: userUUID,
		})
		cancel()
		if err != nil {
			slog.Warn("candidate profile cache failed to map casdoor uuid", "user_uuid", userUUID, "error", err)
			continue
		}

		candidateULID := strings.TrimSpace(resp.GetUserUlid())
		if candidateULID != "" {
			names[candidateULID] = name
		}
	}

	c.mu.Lock()
	c.names = names
	c.inFlight = map[string]struct{}{}
	c.mu.Unlock()

	slog.Info("candidate profile cache refreshed", "count", len(names))
	return nil
}

func (c *CandidateProfileCache) enqueue(candidateULID string) {
	if c == nil || c.gmid == nil {
		return
	}

	c.mu.Lock()
	if _, ok := c.inFlight[candidateULID]; ok {
		c.mu.Unlock()
		return
	}
	c.inFlight[candidateULID] = struct{}{}
	c.mu.Unlock()

	go c.fetchOne(candidateULID)
}

func (c *CandidateProfileCache) fetchOne(candidateULID string) {
	defer func() {
		c.mu.Lock()
		delete(c.inFlight, candidateULID)
		c.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), candidateProfileLookupTimeout)
	defer cancel()

	uuidResp, err := c.gmid.GetUUIDByUlid(ctx, &gmidpb.GetUUIDByUlidRequest{UserUlid: candidateULID})
	if err != nil {
		slog.Warn("candidate profile cache backfill failed to map candidate ulid", "candidate_ulid", candidateULID, "error", err)
		return
	}

	userUUID := strings.TrimSpace(uuidResp.GetUserUuid())
	if userUUID == "" {
		slog.Warn("candidate profile cache backfill got empty casdoor uuid", "candidate_ulid", candidateULID)
		return
	}

	users, err := casdoorsdk.GetUsers()
	if err != nil {
		slog.Warn("candidate profile cache backfill failed to load casdoor users", "candidate_ulid", candidateULID, "error", err)
		return
	}

	for _, user := range users {
		if user == nil || strings.TrimSpace(user.Id) != userUUID {
			continue
		}
		name := strings.TrimSpace(user.Name)
		if name == "" {
			return
		}

		c.mu.Lock()
		c.names[candidateULID] = name
		c.mu.Unlock()
		return
	}

	slog.Warn("candidate profile cache backfill did not find casdoor user", "candidate_ulid", candidateULID, "user_uuid", userUUID)
}

func (h *Handler) StartCandidateProfileCache(ctx context.Context) {
	if h == nil || h.CandidateProfiles == nil {
		return
	}
	h.CandidateProfiles.Start(ctx)
}

func (h *Handler) candidateName(candidateULID string) string {
	if h == nil || h.CandidateProfiles == nil {
		return ""
	}
	return h.CandidateProfiles.NameOrQueue(candidateULID)
}

func (h *Handler) attachCandidateName(payload map[string]interface{}, candidateULID string) {
	if candidateULID != "" {
		payload["candidate_ulid"] = candidateULID
	}
	name := h.candidateName(candidateULID)
	if name != "" {
		payload["candidate_name"] = name
	}
}
