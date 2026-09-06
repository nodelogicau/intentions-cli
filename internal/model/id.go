package model

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// idPattern is the lenient on-read form: any prefix_opaque identifier.
var idPattern = regexp.MustCompile(`^(int|avl|cmt|res)_[A-Za-z0-9-]+$`)

// strictPattern is the minted form: a lowercase canonical UUIDv7.
var strictPattern = regexp.MustCompile(`^(int|avl|cmt|res)_[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// MintID mints a new identifier for the type. google/uuid's NewV7 carries a
// monotonic sub-millisecond sequence, so ids minted by one process sort in
// creation order.
func MintID(t Type) string {
	u, err := uuid.NewV7()
	if err != nil {
		panic(fmt.Sprintf("uuid: %v", err))
	}
	return t.Prefix() + "_" + strings.ToLower(u.String())
}

// ValidID reports whether s has the lenient identifier shape.
func ValidID(s string) bool { return idPattern.MatchString(s) }

// StrictID reports whether s is a minted-form identifier.
func StrictID(s string) bool { return strictPattern.MatchString(s) }

// TypeOfID returns the type an identifier's prefix denotes.
func TypeOfID(id string) (Type, bool) {
	i := strings.IndexByte(id, '_')
	if i < 0 {
		return "", false
	}
	for _, t := range Types {
		if t.Prefix() == id[:i] {
			return t, ValidID(id)
		}
	}
	return "", false
}

func uniqueSorted(ids []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// SortStrings returns a sorted, de-duplicated copy for set-valued lists.
func SortStrings(vs []string) []string {
	if vs == nil {
		return nil
	}
	out := uniqueSorted(vs)
	return out
}

// SortRefs sorts references by id then role.
func SortRefs(rs []Ref) []Ref {
	out := make([]Ref, len(rs))
	copy(out, rs)
	sort.Slice(out, func(i, j int) bool {
		if out[i].ID != out[j].ID {
			return out[i].ID < out[j].ID
		}
		return out[i].Role < out[j].Role
	})
	return out
}

// SortParties sorts parties by uri.
func SortParties(ps []Party) []Party {
	out := make([]Party, len(ps))
	copy(out, ps)
	sort.Slice(out, func(i, j int) bool { return out[i].URI < out[j].URI })
	return out
}
