package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	sortAsc  = "asc"
	sortDesc = "desc"

	sortByName = "name"
)

var (
	projectSortFields = []string{"name", "accepted_at", "incubating_at", "graduated_at"}
	memberSortFields  = []string{"name", "joined_at"}
	sortOrders        = []string{sortAsc, sortDesc}
)

// sortSpec is a validated sort request.
type sortSpec struct {
	field string
	desc  bool
}

// parseSort validates sort_by/sort_order against the allowed fields. Empty
// values select the defaults: sort by name, ascending.
func parseSort(sortBy, sortOrder string, fields []string) (sortSpec, error) {
	field := strings.ToLower(strings.TrimSpace(sortBy))
	if field == "" {
		field = sortByName
	}
	if !containsString(fields, field) {
		return sortSpec{}, fmt.Errorf("invalid sort_by %q; allowed values: %s", sortBy, strings.Join(fields, ", "))
	}
	order := strings.ToLower(strings.TrimSpace(sortOrder))
	if order == "" {
		order = sortAsc
	}
	if !containsString(sortOrders, order) {
		return sortSpec{}, fmt.Errorf("invalid sort_order %q; allowed values: %s", sortOrder, strings.Join(sortOrders, ", "))
	}
	return sortSpec{field: field, desc: order == sortDesc}, nil
}

func containsString(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

// dateFor returns the item date selected by a date sort field.
func dateFor(item *LandscapeItem, field string) *time.Time {
	switch field {
	case "accepted_at":
		return item.AcceptedAt
	case "incubating_at":
		return item.IncubatingAt
	case "graduated_at":
		return item.GraduatedAt
	case "joined_at":
		return item.JoinedAt
	default:
		return nil
	}
}

// sortItems orders items in place. Ties fall back to ascending name; items
// without a date sort last regardless of direction.
func sortItems(items []*LandscapeItem, spec sortSpec) {
	byName := func(a, b *LandscapeItem) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if spec.field == sortByName {
			c := byName(a, b)
			if spec.desc {
				return c > 0
			}
			return c < 0
		}
		da, db := dateFor(a, spec.field), dateFor(b, spec.field)
		switch {
		case da == nil && db == nil:
			return byName(a, b) < 0
		case da == nil:
			return false
		case db == nil:
			return true
		case da.Equal(*db):
			return byName(a, b) < 0
		case spec.desc:
			return da.After(*db)
		default:
			return da.Before(*db)
		}
	})
}

// pageBounds returns the [start, end) window for offset/limit over n items.
// A non-positive limit yields an empty page.
func pageBounds(n, offset, limit int) (int, int) {
	start := offset
	if start < 0 {
		start = 0
	}
	if start > n {
		start = n
	}
	end := start
	if limit > 0 {
		end = start + limit
		if end > n || end < start {
			end = n
		}
	}
	return start, end
}
