package handler

import (
	"testing"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
)

func TestEligibilityFromBundlePreservesConflictBlockers(t *testing.T) {
	blockerTypes := []string{
		"FORBIDDEN_QUALIFICATION",
		"CONFLICT_PIPELINE_IN_PROGRESS",
		"CONFLICT_CHECK_UNAVAILABLE",
		"MEMBERSHIP_UPGRADE_REQUIRED",
		"MEMBERSHIP_ORDER_IN_PROGRESS",
	}
	blockers := make([]*mallpb.EligibilityBlocker, 0, len(blockerTypes))
	for _, blockerType := range blockerTypes {
		blockers = append(blockers, &mallpb.EligibilityBlocker{
			BlockerType: blockerType,
			Description: blockerType + " description",
			Details:     []string{blockerType + " detail"},
		})
	}

	result := eligibilityFromBundle(&mallpb.CheckBundleEligibilityResponse{Blockers: blockers})

	if result.CanPurchase || result.CanUnlock || result.Eligible {
		t.Fatalf("blocked eligibility was enabled: %+v", result)
	}
	if len(result.Blockers) != len(blockerTypes) {
		t.Fatalf("blocker count = %d, want %d", len(result.Blockers), len(blockerTypes))
	}
	for index, blockerType := range blockerTypes {
		if result.Blockers[index].BlockerType != blockerType || result.Blockers[index].Details[0] != blockerType+" detail" {
			t.Fatalf("blocker %d was not preserved: %+v", index, result.Blockers[index])
		}
	}
}

func TestEligibilityAllowsExemptionManagementWhileAnotherQualificationIsPending(t *testing.T) {
	for _, blockerType := range []string{
		"EXEMPTION_DOCUMENTS_PENDING_UPLOAD",
		"EXEMPTION_UNDER_REVIEW",
	} {
		eligibility := bundleEligibilitySummary{
			Blockers: []bundleEligibilityBlocker{{BlockerType: blockerType}},
		}
		if !eligibilityAllowsExemptionManagement(eligibility) {
			t.Fatalf("exemption management was blocked by %s", blockerType)
		}
	}
}

func TestEligibilityDoesNotBypassPurchaseBlockersForExemptionManagement(t *testing.T) {
	eligibility := bundleEligibilitySummary{
		Blockers: []bundleEligibilityBlocker{
			{BlockerType: "EXEMPTION_DOCUMENTS_PENDING_UPLOAD"},
			{BlockerType: "CONFLICT_PIPELINE_IN_PROGRESS"},
		},
	}
	if eligibilityAllowsExemptionManagement(eligibility) {
		t.Fatal("conflicting pipeline blocker was bypassed")
	}
}

func TestToPipelineConfigSeparatesFinalAuditRequirementsAndAwards(t *testing.T) {
	pipeline := &gccpb.PipelineConfig{
		PipelineUlid: "pipeline-1",
		PrerequisiteQuals: []*gccpb.QualificationRequirement{{
			QualUlid: "prerequisite-1",
			NameHint: "Prerequisite",
		}},
		FinalAuditQuals: []*gccpb.QualificationRequirement{{
			QualUlid: "audit-1",
			NameHint: "Work Experience",
		}},
		AwardCerts: []*gccpb.CertificateAward{{
			QualUlid:        "award-1",
			PdfTemplateUlid: "template-1",
			NameHint:        "CFTP Certificate",
		}},
	}

	result := toPipelineConfig(pipeline)

	if len(result.PrerequisiteQuals) != 1 || result.PrerequisiteQuals[0].QualUlid != "prerequisite-1" {
		t.Fatalf("prerequisite qualifications = %+v", result.PrerequisiteQuals)
	}
	if len(result.FinalAuditQuals) != 1 || result.FinalAuditQuals[0].QualUlid != "audit-1" {
		t.Fatalf("final audit qualifications = %+v", result.FinalAuditQuals)
	}
	if len(result.AwardCerts) != 1 || result.AwardCerts[0].PdfTemplateUlid != "template-1" || result.AwardCerts[0].NameHint != "CFTP Certificate" {
		t.Fatalf("award certificates = %+v", result.AwardCerts)
	}
	if !result.HasCertificate {
		t.Fatal("pipeline with an award certificate was not marked as issuing a certificate")
	}

	pipeline.AwardCerts = nil
	if result := toPipelineConfig(pipeline); result.HasCertificate {
		t.Fatal("final audit requirements alone marked the pipeline as issuing a certificate")
	}
}
