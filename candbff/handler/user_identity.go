package handler

import (
	"context"
	"log/slog"
	"strings"
	"time"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
)

// These identifiers are the CFtA/CFtP configurations in dev and prod.
// Display names and numeric membership tiers are not certification identifiers.
func certificationCodeForPipeline(gpath string) string {
	switch gpath {
	case "/gcc/pipeline/core/cfta":
		return "CFtA"
	case "/gcc/pipeline/core/cftp":
		return "CFtP"
	default:
		return ""
	}
}

func certificationCodeForCredential(respath string) string {
	switch respath {
	// Both CFtA definitions exist in the service; either can establish the identity.
	case "/gcreds/core/cfta", "/gcc/credential/system/cfta-certification":
		return "CFtA"
	case "/gcreds/core/cftp":
		return "CFtP"
	default:
		return ""
	}
}

// Map the configured base plan names to the customer's display terminology
// before localization. This does not change membership tiers or permissions.
func membershipDisplayCode(name string) string {
	switch strings.TrimSpace(name) {
	case "Affiliate Membership":
		return "affiliate_member"
	case "Charterholder Membership":
		return "charterholder_member"
	default:
		return ""
	}
}

func (h *Handler) loadUserQualificationStatus(ctx context.Context, candidateID string) UserQualificationStatusRsp {
	unavailable := UserQualificationStatusRsp{CharterholderPrograms: []string{}}
	if h.Creds == nil || strings.TrimSpace(candidateID) == "" {
		return unavailable
	}
	out := UserQualificationStatusRsp{CharterholderPrograms: []string{}}
	definitions := make(map[string]string)
	held := make(map[string]bool)
	guard := newCursorScanGuard()
	cursor := ""
	now := time.Now()
	for {
		resp, err := h.Creds.ListCandidateCredentials(ctx, &gcredspb.ListCandidateCredentialsRequest{
			CandidateUlid: candidateID,
			Cursor:        cursor,
			PageSize:      100,
		})
		if err != nil || resp == nil {
			slog.Warn("failed to load current user qualifications", "candidate_id", candidateID, "error", err)
			return unavailable
		}
		for _, credential := range resp.GetCredentials() {
			if credential == nil || !credential.GetIsCurrent() || credential.GetStatus() != gcredspb.CredentialStatus_CREDENTIAL_STATUS_ACTIVE {
				continue
			}
			if until := strings.TrimSpace(credential.GetValidUntil()); until != "" {
				expires, err := time.Parse(time.RFC3339, until)
				if err != nil || !expires.After(now) {
					continue
				}
			}
			out.CredentialCount++
			definitionID := credential.GetCredDefUlid()
			code, loaded := definitions[definitionID]
			if !loaded {
				definition, err := h.Creds.GetCredentialDefinitionDetail(ctx, &gcredspb.GetCredentialDefinitionDetailRequest{CredDefUlid: definitionID})
				if err != nil || definition == nil {
					slog.Warn("failed to identify current user qualification", "candidate_id", candidateID, "cred_def_id", definitionID, "error", err)
					return unavailable
				}
				code = certificationCodeForCredential(definition.GetRespath())
				definitions[definitionID] = code
			}
			if code != "" {
				held[code] = true
			}
		}
		next, done, err := guard.next(cursor, resp.GetHasMore(), resp.GetNextCursor())
		if err != nil {
			slog.Warn("failed to page current user qualifications", "candidate_id", candidateID, "error", err)
			return unavailable
		}
		if done {
			break
		}
		cursor = next
	}
	for _, code := range []string{"CFtA", "CFtP"} {
		if held[code] {
			out.CharterholderPrograms = append(out.CharterholderPrograms, code)
		}
	}
	out.Available = true
	out.HasQualification = out.CredentialCount > 0
	return out
}
