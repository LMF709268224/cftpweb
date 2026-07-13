package handler

import "testing"

func TestCandidateOrderRawStatus(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "numeric pending payment", raw: "2", want: "2"},
		{name: "prefixed pending payment", raw: "ORDER_STATUS_PENDING_PAYMENT", want: "ORDER_STATUS_PENDING_PAYMENT"},
		{name: "plain pending", raw: "PENDING", want: "PENDING"},
		{name: "success alias", raw: "SUCCESS", want: "SUCCESS"},
		{name: "cancel alias", raw: "CANCEL", want: "CANCEL"},
		{name: "trim space", raw: " PENDING ", want: "PENDING"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := candidateOrderRawStatus(tt.raw); got != tt.want {
				t.Fatalf("candidateOrderRawStatus(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestCanCancelCommonOrderStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{name: "wait payment", status: "WAIT_PAYMENT", want: true},
		{name: "pending", status: "PENDING", want: true},
		{name: "completed", status: "COMPLETED", want: false},
		{name: "cancelled", status: "CANCELLED", want: false},
		{name: "closed", status: "CLOSED", want: false},
		{name: "business wait state is not common state", status: "WAIT_STAGE_PAYMENT", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canCancelCommonOrderStatus(tt.status); got != tt.want {
				t.Fatalf("canCancelCommonOrderStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
