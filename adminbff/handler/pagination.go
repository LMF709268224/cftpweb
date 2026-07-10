package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

const (
	defaultCursorPageSize = 20
	maxCursorPageSize     = 200
	exactCountLimit       = 100000
)

type cursorPage struct {
	Cursor   string
	PageSize uint32
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
	return cursorPage{
		Cursor:   strings.TrimSpace(r.URL.Query().Get("cursor")),
		PageSize: uint32(pageSize),
	}
}

func cursorListPayload(items interface{}, page cursorPage, nextCursor string, hasMore bool) map[string]interface{} {
	return map[string]interface{}{
		"items":       items,
		"page_size":   page.PageSize,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	}
}
