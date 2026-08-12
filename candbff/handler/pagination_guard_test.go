package handler

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseUint32QueryUsesCandidatePageLimit(t *testing.T) {
	tests := []struct {
		query string
		want  uint32
	}{
		{query: "", want: maxCandidateListPageSize},
		{query: "?page_size=0", want: maxCandidateListPageSize},
		{query: "?page_size=invalid", want: maxCandidateListPageSize},
		{query: "?page_size=7", want: 7},
		{query: "?page_size=21", want: maxCandidateListPageSize},
		{query: "?page_size=4294967295", want: maxCandidateListPageSize},
	}
	for _, test := range tests {
		req := httptest.NewRequest("GET", "/api/resource-packs"+test.query, nil)
		if got := parseUint32Query(req, "page_size"); got != test.want {
			t.Errorf("query %q page size = %d, want %d", test.query, got, test.want)
		}
	}
}

func TestCursorScanGuardStopsAndRejectsLoops(t *testing.T) {
	guard := newCursorScanGuard()
	next, done, err := guard.next("", true, "cursor-1")
	if err != nil || done || next != "cursor-1" {
		t.Fatalf("first next = (%q, %v, %v), want cursor-1, false, nil", next, done, err)
	}
	if _, _, err := guard.next("cursor-1", true, "cursor-1"); err == nil {
		t.Fatal("same cursor was accepted")
	}

	guard = newCursorScanGuard()
	if _, _, err := guard.next("", true, "cursor-1"); err != nil {
		t.Fatalf("first cursor: %v", err)
	}
	if _, _, err := guard.next("cursor-1", true, "cursor-2"); err != nil {
		t.Fatalf("second cursor: %v", err)
	}
	if _, _, err := guard.next("cursor-2", true, "cursor-1"); err == nil {
		t.Fatal("cursor cycle was accepted")
	}

	guard = newCursorScanGuard()
	if next, done, err := guard.next("cursor", false, "ignored"); err != nil || !done || next != "" {
		t.Fatalf("finished scan = (%q, %v, %v), want empty, true, nil", next, done, err)
	}
}

func TestCursorScanGuardLimitsPages(t *testing.T) {
	guard := newCursorScanGuard()
	current := ""
	for i := 1; i < maxCursorScanPages; i++ {
		next := strings.Repeat("x", i)
		var err error
		current, _, err = guard.next(current, true, next)
		if err != nil {
			t.Fatalf("page %d: %v", i, err)
		}
	}
	if _, _, err := guard.next(current, true, "limit"); err == nil {
		t.Fatal("page limit was not enforced")
	}
}
