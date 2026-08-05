package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"candbff/config"
)

func TestGetPublicConfigOnlyReturnsBrowserSafeStripeKey(t *testing.T) {
	t.Setenv(config.EnvStripePublishableKey, "pk_test_candidate")
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_must_not_be_exposed")
	handler := &Handler{}
	recorder := httptest.NewRecorder()

	handler.GetPublicConfig(
		recorder,
		httptest.NewRequest(http.MethodGet, "/api/public/config", nil),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if recorder.Body.String() == "" {
		t.Fatal("response body is empty")
	}
	if containsJSONValue(recorder.Body.Bytes(), "sk_test_must_not_be_exposed") {
		t.Fatal("public config exposed Stripe secret key")
	}

	var response struct {
		Data PublicConfigRsp `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Data.StripePublishableKey != "pk_test_candidate" {
		t.Fatalf("publishable key = %q", response.Data.StripePublishableKey)
	}
}

func containsJSONValue(body []byte, value string) bool {
	var decoded interface{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return false
	}
	return containsValue(decoded, value)
}

func containsValue(value interface{}, target string) bool {
	switch typed := value.(type) {
	case string:
		return typed == target
	case []interface{}:
		for _, item := range typed {
			if containsValue(item, target) {
				return true
			}
		}
	case map[string]interface{}:
		for _, item := range typed {
			if containsValue(item, target) {
				return true
			}
		}
	}
	return false
}
