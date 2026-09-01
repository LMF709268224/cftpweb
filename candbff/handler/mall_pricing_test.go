package handler

import (
	"net/http"
	"net/url"
	"testing"
)

func TestBundlePricingDetailRequestIncludesPaymentMode(t *testing.T) {
	const (
		bundleID    = "01KYN000000000000000000101"
		candidateID = "01KYN000000000000000000102"
		selections  = `{"pipeline":{"stages":[]}}`
	)

	target := "/api/mall/bundles/" + bundleID + "/pricing-detail?payment_mode=BY_STAGE&selected_exemptions_json=" + url.QueryEscape(selections)
	httpRequest := newCandidateHandlerRequest(http.MethodGet, target, "", candidateID, map[string]string{"bundleId": bundleID})
	request := bundlePricingDetailRequest(httpRequest, bundleID, candidateID)

	if request.GetBundleUlid() != bundleID {
		t.Fatalf("bundle_ulid = %q, want %q", request.GetBundleUlid(), bundleID)
	}
	if request.GetCandidateUlid() != candidateID {
		t.Fatalf("candidate_ulid = %q, want %q", request.GetCandidateUlid(), candidateID)
	}
	if request.GetPaymentMode() != "BY_STAGE" {
		t.Fatalf("payment_mode = %q, want BY_STAGE", request.GetPaymentMode())
	}
	if request.GetSelectedExemptionsJson() != selections {
		t.Fatalf("selected_exemptions_json = %q, want %q", request.GetSelectedExemptionsJson(), selections)
	}
}
