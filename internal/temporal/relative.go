package temporal

import (
	"fmt"
	"strings"
)

// Relation is an RFC 9253 temporal relation.
type Relation string

// The four admitted relations.
const (
	FinishToStart  Relation = "FINISHTOSTART"
	FinishToFinish Relation = "FINISHTOFINISH"
	StartToFinish  Relation = "STARTTOFINISH"
	StartToStart   Relation = "STARTTOSTART"
)

// Relations lists the admitted vocabulary in a stable order.
var Relations = []Relation{FinishToStart, FinishToFinish, StartToFinish, StartToStart}

// ValidRelation reports whether r is admitted.
func ValidRelation(r Relation) bool {
	for _, x := range Relations {
		if x == r {
			return true
		}
	}
	return false
}

// Gap is the admitted range between the related ends.
type Gap struct {
	Min *Duration
	Max *Duration
}

// Relative is a WINDOW's relational anchor. The dependent window holds the
// reference; the target is never modified.
type Relative struct {
	Target   string
	Relation Relation
	Gap      *Gap
}

// Validate checks the relation vocabulary and gap ordering.
func (r Relative) Validate() error {
	if strings.TrimSpace(r.Target) == "" {
		return fmt.Errorf("relative anchor has no target")
	}
	if !ValidRelation(r.Relation) {
		return fmt.Errorf("invalid relation %q: must be one of FINISHTOSTART, FINISHTOFINISH, STARTTOFINISH, STARTTOSTART", r.Relation)
	}
	if r.Gap != nil && r.Gap.Min != nil && r.Gap.Max != nil && r.Gap.Min.ApproxSeconds() > r.Gap.Max.ApproxSeconds() {
		return fmt.Errorf("relative anchor gap min %s is longer than max %s", *r.Gap.Min, *r.Gap.Max)
	}
	return nil
}

// ParseRelative parses the command-line form target:RELATION[:min[:max]].
func ParseRelative(s string) (Relative, error) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) < 2 || len(parts) > 4 {
		return Relative{}, fmt.Errorf("invalid --relative %q: use target:RELATION[:min[:max]] such as int_A:FINISHTOSTART:P0D:P3D", s)
	}
	r := Relative{Target: parts[0], Relation: Relation(strings.ToUpper(parts[1]))}
	if len(parts) >= 3 {
		mn, err := ParseDuration(parts[2])
		if err != nil {
			return Relative{}, err
		}
		r.Gap = &Gap{Min: &mn}
	}
	if len(parts) == 4 {
		mx, err := ParseDuration(parts[3])
		if err != nil {
			return Relative{}, err
		}
		r.Gap.Max = &mx
	}
	return r, r.Validate()
}
