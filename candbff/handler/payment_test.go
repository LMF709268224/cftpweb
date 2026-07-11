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

func TestCanCancelBusinessOrder(t *testing.T) {
	tests := []struct {
		name    string
		bizType string
		status  string
		want    bool
	}{
		{name: "bundle wait payment", bizType: orderBizBundlePurchase, status: "WAIT_PAYMENT", want: true},
		{name: "bundle paid", bizType: orderBizBundlePurchase, status: "COMPLETED", want: false},
		{name: "stage exemption selection", bizType: orderBizStagePayment, status: "WAIT_EXEMPTION_SELECTION", want: true},
		{name: "stage wait payment", bizType: orderBizStagePayment, status: "WAIT_STAGE_PAYMENT", want: true},
		{name: "retake wait payment", bizType: orderBizCourseRetakePayment, status: "WAIT_PAYMENT", want: true},
		{name: "unlock wait payment", bizType: orderBizPipelineUnlock, status: "WAIT_PAYMENT", want: true},
		{name: "credential wait review fee", bizType: orderBizCredentialApply, status: "WAIT_REVIEW_FEE_PAYMENT", want: true},
		{name: "credential upload ready", bizType: orderBizCredentialApply, status: "UPLOAD_READY", want: false},
		{name: "pipeline payment unsupported", bizType: orderBizPipelinePayment, status: "WAIT_PAYMENT", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canCancelBusinessOrder(tt.bizType, tt.status); got != tt.want {
				t.Fatalf("canCancelBusinessOrder(%q, %q) = %v, want %v", tt.bizType, tt.status, got, tt.want)
			}
		})
	}
}
