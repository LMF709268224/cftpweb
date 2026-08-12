package handler

import (
	"context"
	"errors"
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
	candidateProfileRefreshWorkers  = 8
	candidateProfileBackfillLimit   = 4
)

var (
	listCandidateProfileUsers = casdoorsdk.GetUsers
	getCandidateProfileUser   = casdoorsdk.GetUserByUserId
)

type CandidateProfileCache struct {
	gmid gmidpb.MidServiceClient

	mu            sync.RWMutex
	refreshMu     sync.Mutex
	users         []*casdoorsdk.User
	names         map[string]string
	ulidsByUUID   map[string]string
	inFlight      map[string]struct{}
	backfillSlots chan struct{}
	ready         bool
}

func NewCandidateProfileCache(gmid gmidpb.MidServiceClient) *CandidateProfileCache {
	return &CandidateProfileCache{
		gmid:          gmid,
		users:         []*casdoorsdk.User{},
		names:         map[string]string{},
		ulidsByUUID:   map[string]string{},
		inFlight:      map[string]struct{}{},
		backfillSlots: make(chan struct{}, candidateProfileBackfillLimit),
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
	if c == nil || c.gmid == nil {
		return errors.New("candidate profile cache is not configured")
	}
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()
	return c.refresh(ctx)
}

func (c *CandidateProfileCache) refresh(ctx context.Context) error {
	users, err := listCandidateProfileUsers()
	if err != nil {
		return err
	}

	type mapping struct {
		uuid string
		ulid string
		name string
	}
	jobs := make(chan *casdoorsdk.User)
	results := make(chan mapping, candidateProfileRefreshWorkers)
	var workers sync.WaitGroup
	for range candidateProfileRefreshWorkers {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for user := range jobs {
				userUUID := strings.TrimSpace(user.Id)
				lookupCtx, cancel := context.WithTimeout(ctx, candidateProfileGmidTimeout)
				resp, lookupErr := c.gmid.GetUlidByUUID(lookupCtx, &gmidpb.GetUlidByUUIDRequest{UserUuid: userUUID})
				cancel()
				if lookupErr != nil {
					slog.Warn("candidate profile cache failed to map casdoor uuid", "user_uuid", userUUID, "error", lookupErr)
					continue
				}
				candidateULID := strings.TrimSpace(resp.GetUserUlid())
				if candidateULID != "" {
					results <- mapping{uuid: userUUID, ulid: candidateULID, name: candidateDisplayName(user)}
				}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, user := range users {
			if user == nil || strings.TrimSpace(user.Id) == "" {
				continue
			}
			select {
			case jobs <- user:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		workers.Wait()
		close(results)
	}()

	names := make(map[string]string, len(users))
	ulidsByUUID := make(map[string]string, len(users))
	for result := range results {
		ulidsByUUID[result.uuid] = result.ulid
		if result.name != "" {
			names[result.ulid] = result.name
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	c.mu.Lock()
	c.users = append([]*casdoorsdk.User(nil), users...)
	c.names = names
	c.ulidsByUUID = ulidsByUUID
	c.inFlight = map[string]struct{}{}
	c.ready = true
	c.mu.Unlock()

	slog.Info("candidate profile cache refreshed", "count", len(names))
	return nil
}

func (c *CandidateProfileCache) enqueue(candidateULID string) {
	if c == nil || c.gmid == nil {
		return
	}

	select {
	case c.backfillSlots <- struct{}{}:
	default:
		return
	}

	c.mu.Lock()
	if _, ok := c.inFlight[candidateULID]; ok {
		c.mu.Unlock()
		<-c.backfillSlots
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
		<-c.backfillSlots
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

	user, err := getCandidateProfileUser(userUUID)
	if err != nil {
		slog.Warn("candidate profile cache backfill failed to load casdoor user", "candidate_ulid", candidateULID, "user_uuid", userUUID, "error", err)
		return
	}
	if user == nil || strings.TrimSpace(user.Id) != userUUID {
		slog.Warn("candidate profile cache backfill returned a different casdoor user", "candidate_ulid", candidateULID, "user_uuid", userUUID)
		return
	}

	c.mu.Lock()
	c.ulidsByUUID[userUUID] = candidateULID
	if name := candidateDisplayName(user); name != "" {
		c.names[candidateULID] = name
	}
	c.mu.Unlock()
}

func (c *CandidateProfileCache) ULIDForUUID(userUUID string) (string, bool) {
	if c == nil {
		return "", false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.ready {
		return "", false
	}
	ulid, ok := c.ulidsByUUID[strings.TrimSpace(userUUID)]
	return ulid, ok
}

func (c *CandidateProfileCache) Users() ([]*casdoorsdk.User, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.ready {
		return nil, false
	}
	return append([]*casdoorsdk.User(nil), c.users...), true
}

func (c *CandidateProfileCache) UsersOrRefresh(ctx context.Context) ([]*casdoorsdk.User, error) {
	if users, ok := c.Users(); ok {
		return users, nil
	}
	if c == nil || c.gmid == nil {
		return nil, errors.New("candidate profile cache is not configured")
	}
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()
	if users, ok := c.Users(); ok {
		return users, nil
	}
	if err := c.refresh(ctx); err != nil {
		return nil, err
	}
	users, ok := c.Users()
	if !ok {
		return nil, errors.New("candidate profile cache refresh produced no snapshot")
	}
	return users, nil
}

func (c *CandidateProfileCache) Ready() bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ready
}

func candidateDisplayName(user *casdoorsdk.User) string {
	if user == nil {
		return ""
	}
	for _, value := range []string{user.DisplayName, user.RealName, strings.TrimSpace(user.FirstName + " " + user.LastName), user.Name} {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
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
