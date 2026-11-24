package handler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
)

func TestProfileCharterholdersUseCurrentValidCredentialsAcrossPages(t *testing.T) {
	client := &userStatusCredentialClient{
		pages: map[string]*gcredspb.ListCandidateCredentialsResponse{
			"":       {Credentials: []*gcredspb.CredentialSummary{currentIdentityCredential("cfta-old"), currentIdentityCredential("work")}, HasMore: true, NextCursor: "second"},
			"second": {Credentials: []*gcredspb.CredentialSummary{currentIdentityCredential("cftp"), currentIdentityCredential("cfta-new"), currentIdentityCredential("cftp")}},
		},
		definitions: map[string]*gcredspb.CredentialDefinition{
			"cfta-old": {Respath: "/gcreds/core/cfta", Name: "Localized title"},
			"cfta-new": {Respath: "/gcc/credential/system/cfta-certification"},
			"cftp":     {Respath: "/gcreds/core/cftp"},
			"work":     {Respath: "/gcreds/work-experience", Name: "CFtP Work Experience Proof"},
		},
	}
	result := (&Handler{Creds: client}).loadUserQualificationStatus(context.Background(), "candidate-1")
	if !result.Available || !reflect.DeepEqual(result.CharterholderPrograms, []string{"CFtA", "CFtP"}) || result.CredentialCount != 5 {
		t.Fatalf("qualification = %#v", result)
	}
	if len(client.requests) != 2 || client.requests[1].GetCursor() != "second" || client.requests[1].GetCandidateUlid() != "candidate-1" {
		t.Fatalf("requests = %#v", client.requests)
	}
	if client.definitionCalls["cftp"] != 1 {
		t.Fatalf("definition fetched %d times", client.definitionCalls["cftp"])
	}
}

func TestProfileCharterholdersExcludeExpiredRevokedAndHistoricalCredentials(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		change func(*gcredspb.CredentialSummary)
	}{
		{"revoked", func(c *gcredspb.CredentialSummary) { c.Status = gcredspb.CredentialStatus_CREDENTIAL_STATUS_REVOKED }},
		{"expired status", func(c *gcredspb.CredentialSummary) { c.Status = gcredspb.CredentialStatus_CREDENTIAL_STATUS_EXPIRED }},
		{"expired date", func(c *gcredspb.CredentialSummary) { c.ValidUntil = time.Now().Add(-time.Minute).Format(time.RFC3339) }},
		{"invalid date", func(c *gcredspb.CredentialSummary) { c.ValidUntil = "invalid" }},
		{"historical", func(c *gcredspb.CredentialSummary) { c.IsCurrent = false }},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			credential := currentIdentityCredential("cftp")
			scenario.change(credential)
			client := &userStatusCredentialClient{pages: map[string]*gcredspb.ListCandidateCredentialsResponse{"": {Credentials: []*gcredspb.CredentialSummary{credential}}}}
			result := (&Handler{Creds: client}).loadUserQualificationStatus(context.Background(), "candidate-1")
			if !result.Available || result.HasQualification || len(result.CharterholderPrograms) != 0 || len(client.definitionCalls) != 0 {
				t.Fatalf("qualification = %#v", result)
			}
		})
	}
}

func TestProfileCharterholdersDoNotReturnPartialResultsAfterFailure(t *testing.T) {
	for _, scenario := range []string{"page error", "cursor loop", "definition error", "empty response"} {
		t.Run(scenario, func(t *testing.T) {
			client := &userStatusCredentialClient{
				pages: map[string]*gcredspb.ListCandidateCredentialsResponse{
					"":     {Credentials: []*gcredspb.CredentialSummary{currentIdentityCredential("cfta")}, HasMore: true, NextCursor: "next"},
					"next": {Credentials: []*gcredspb.CredentialSummary{currentIdentityCredential("unknown")}},
				},
				definitions: map[string]*gcredspb.CredentialDefinition{"cfta": {Respath: "/gcreds/core/cfta"}},
			}
			switch scenario {
			case "page error":
				client.failedCursor = "next"
				client.err = errors.New("unavailable")
			case "cursor loop":
				client.pages["next"] = &gcredspb.ListCandidateCredentialsResponse{HasMore: true, NextCursor: "next"}
			case "empty response":
				client.pages["next"] = nil
			}
			result := (&Handler{Creds: client}).loadUserQualificationStatus(context.Background(), "candidate-1")
			if result.Available || result.HasQualification || len(result.CharterholderPrograms) != 0 {
				t.Fatalf("partial result = %#v", result)
			}
		})
	}
}

func TestProfileIdentityMappingsDoNotGuessFromSimilarNames(t *testing.T) {
	if certificationCodeForPipeline("/gcc/pipeline/core/cfta") != "CFtA" || certificationCodeForPipeline("/gcc/pipeline/core/cftp") != "CFtP" {
		t.Fatal("configured certification paths were not mapped")
	}
	for _, value := range []string{"", "/courses/cftp-intro", "/gcc/pipeline/core/cftx"} {
		if certificationCodeForPipeline(value) != "" {
			t.Fatalf("unexpected certification: %q", value)
		}
	}
	for _, value := range []string{"/gcreds/cftp-exemption", "/gcreds/cftp-work-experience", "CFtP Certification"} {
		if certificationCodeForCredential(value) != "" {
			t.Fatalf("unexpected charterholder: %q", value)
		}
	}
	if membershipDisplayCode("Affiliate Membership") != "affiliate_member" || membershipDisplayCode("Charterholder Membership") != "charterholder_member" || membershipDisplayCode("CFtP Fellow") != "" {
		t.Fatal("membership labels must use the configured plan names")
	}
}
