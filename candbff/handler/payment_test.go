package handler

import "testing"

func TestCandidateOrderRawStatus(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "numeric pending payment", raw: "2", want: "PENDING_PAYMENT"},
		{name: "prefixed pending payment", raw: "ORDER_STATUS_PENDING_PAYMENT", want: "PENDING_PAYMENT"},
		{name: "plain pending", raw: "PENDING", want: "PENDING_PAYMENT"},
		{name: "success alias", raw: "SUCCESS", want: "COMPLETED"},
		{name: "cancel alias", raw: "CANCEL", want: "CANCELLED"},
		{name: "hyphen normalization", raw: "pending-payment", want: "PENDING_PAYMENT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := candidateOrderRawStatus(tt.raw); got != tt.want {
				t.Fatalf("candidateOrderRawStatus(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

