package server

import (
	"testing"
	"time"
)

func TestHTTPWriteTimeoutAllowsRequestTimeoutResponse(t *testing.T) {
	if requestTimeout != 60*time.Second {
		t.Fatalf("expected request timeout to remain 60s, got %s", requestTimeout)
	}
	if httpWriteTimeout != 75*time.Second {
		t.Fatalf("expected HTTP write timeout to be 75s, got %s", httpWriteTimeout)
	}
	if httpWriteTimeout <= requestTimeout {
		t.Fatalf("HTTP write timeout %s must exceed request timeout %s", httpWriteTimeout, requestTimeout)
	}
}
