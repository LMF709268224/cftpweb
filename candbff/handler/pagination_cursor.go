package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

const (
	defaultCursorPageSize           = 20
	maxCursorPageSize               = 100
	maxCandidateListPageSize uint32 = 20
	exactCountLimit                 = 100000
)

const maxCursorScanPages = 100

type cursorScanGuard struct {
	seen  map[string]struct{}
	pages int
}

func newCursorScanGuard() *cursorScanGuard {
	return &cursorScanGuard{seen: make(map[string]struct{})}
}

func (g *cursorScanGuard) next(current string, hasMore bool, rawNext string) (string, bool, error) {
	g.pages++
	next := strings.TrimSpace(rawNext)
	if !hasMore || next == "" {
		return "", true, nil
	}
	if g.pages >= maxCursorScanPages {
		return "", false, fmt.Errorf("pagination exceeded %d pages", maxCursorScanPages)
	}
	if next == current {
		return "", false, fmt.Errorf("pagination cursor did not advance")
	}
	if _, ok := g.seen[next]; ok {
		return "", false, fmt.Errorf("pagination cursor loop detected")
	}
	g.seen[next] = struct{}{}
	return next, false, nil
}

type cursorPage struct {
	Cursor   string
	PageSize uint32
	Sort     int32
}

type countResult struct {
	Total uint32
	Exact bool
}

func (r countResult) Label() string {
	if r.Exact {
		return fmt.Sprintf("%d", r.Total)
	}
	return fmt.Sprintf("%d+", r.Total)
}

func countCursorAll(ctx context.Context, call func(context.Context, string, uint32) (uint32, string, error)) (countResult, error) {
	const batchLimit uint32 = 1000
	cursor := ""
	total := uint32(0)
	seen := map[string]struct{}{}

	for i := 0; i < 100; i++ {
		if cursor != "" {
			if _, ok := seen[cursor]; ok {
				return countResult{Total: total, Exact: false}, fmt.Errorf("count cursor loop detected")
			}
			seen[cursor] = struct{}{}
		}

		count, nextCursor, err := call(ctx, cursor, batchLimit)
		if err != nil {
			return countResult{}, err
		}
		total += count
		if nextCursor == "" {
			return countResult{Total: total, Exact: true}, nil
		}
		if nextCursor == cursor {
			return countResult{Total: total, Exact: false}, fmt.Errorf("count cursor did not advance")
		}
		if count == 0 {
			return countResult{Total: total, Exact: false}, fmt.Errorf("count returned zero with next cursor")
		}
		if total >= exactCountLimit {
			return countResult{Total: exactCountLimit, Exact: false}, nil
		}
		cursor = nextCursor
	}

	return countResult{Total: total, Exact: false}, fmt.Errorf("count exceeded max iterations")
}

func parseCursorPage(r *http.Request, fallback int) cursorPage {
	pageSize := parsePositiveIntQuery(r, "page_size", parsePositiveIntQuery(r, "limit", fallback))
	if pageSize <= 0 {
		pageSize = defaultCursorPageSize
	}
	if pageSize > maxCursorPageSize {
		pageSize = maxCursorPageSize
	}
	sortStr := strings.TrimSpace(r.URL.Query().Get("sort"))
	var sortOrder int32
	if sortStr == "1" {
		sortOrder = 1
	}
	return cursorPage{
		Cursor:   strings.TrimSpace(r.URL.Query().Get("cursor")),
		PageSize: uint32(pageSize),
		Sort:     sortOrder,
	}
}
