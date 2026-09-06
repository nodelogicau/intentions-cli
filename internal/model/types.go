// Package model is the format layer of the Intentions Format: object types,
// identifiers, the YAML codec with canonical field order, the field rules
// shared by writes and validation, and the write policy. It has no
// filesystem access.
package model

import (
	"time"

	"gopkg.in/yaml.v3"

	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// Format is the format version this implementation writes and reads.
const Format = "intentions/0.1"

// Type is an object or record type.
type Type string

// The four file-backed types.
const (
	TypeIntention    Type = "intention"
	TypeAvailability Type = "availability"
	TypeCommitment   Type = "commitment"
	TypeResolution   Type = "resolution"
)

// Types lists the file-backed types in directory order.
var Types = []Type{TypeIntention, TypeAvailability, TypeCommitment, TypeResolution}

// Prefix returns the id prefix for the type.
func (t Type) Prefix() string {
	switch t {
	case TypeIntention:
		return "int"
	case TypeAvailability:
		return "avl"
	case TypeCommitment:
		return "cmt"
	case TypeResolution:
		return "res"
	}
	return ""
}

// Dir returns the workspace directory for the type.
func (t Type) Dir() string {
	switch t {
	case TypeIntention:
		return "intentions"
	case TypeAvailability:
		return "availability"
	case TypeCommitment:
		return "commitments"
	case TypeResolution:
		return "resolutions"
	}
	return ""
}

// Source says who an object or record is on behalf of and which hand wrote it.
// Canonical order: author, harness, model.
type Source struct {
	Author  string `json:"author,omitempty"`
	Harness string `json:"harness,omitempty"`
	Model   string `json:"model,omitempty"`
}

// IsZero reports whether no field is set.
func (s Source) IsZero() bool { return s == Source{} }

// Ref is an outbound serves reference. Canonical order: id, role.
type Ref struct {
	ID   string `json:"id"`
	Role string `json:"role"`
}

// Serves roles.
const (
	RoleInOrderTo    = "in-order-to"
	RoleForTheSakeOf = "for-the-sake-of"
	RoleInstanceOf   = "instance-of"
)

// Roles lists the admitted serves roles.
var Roles = []string{RoleInOrderTo, RoleForTheSakeOf, RoleInstanceOf}

// Party is a commitment party. Canonical order: uri, status.
type Party struct {
	URI    string `json:"uri"`
	Status string `json:"status"`
}

// Party statuses.
var PartyStatuses = []string{"tentative", "accepted", "declined"}

// Retired is the single retirement record. Canonical order: kind, reason,
// superseded_by, source, timestamp.
type Retired struct {
	Kind         string
	Reason       string
	SupersededBy string
	Source       Source
	Timestamp    time.Time
}

// Retirement kinds per type.
var (
	IntentionRetirementKinds    = []string{"fulfilled", "abandoned", "superseded"}
	AvailabilityRetirementKinds = []string{"retracted", "superseded"}
	CommitmentRetirementKinds   = []string{"cancelled"}
)

// Acknowledgement is an entry in an acknowledgements list. Canonical order:
// kind, counterpart, counterpart_version, reason, source, timestamp.
type Acknowledgement struct {
	Kind               string
	Counterpart        string
	CounterpartVersion string
	Reason             string
	Source             Source
	Timestamp          time.Time
}

// Policy is an auto_select or auto_firm condition. Canonical order:
// max_duration, stability.
type Policy struct {
	MaxDuration *temporal.Duration
	Stability   string
}

// Stability values.
var Stabilities = []string{"tentative", "firm"}

// Preference values.
var Preferences = []string{"earliest", "latest", "adjacent", "spread"}

// Scope values, in widening order.
var Scopes = []string{"personal", "organisation", "public"}

// Origin is a commitment's origin: a resolution id, or import.
type Origin struct {
	Resolution string
	Import     bool
}

// External is the single link to an external calendar object.
type External struct {
	System string
	UID    string
}

// External systems.
var ExternalSystems = []string{"icalendar", "jscalendar"}

// Extra is a field this implementation does not know, preserved verbatim.
type Extra struct {
	Key  string
	Node *yaml.Node
}

// Intention is what a person means to do. Canonical order: id, subject,
// title, description, duration, window, stability, activity, location,
// parties, serves, cadence, occurrence, placement, preference, auto_select,
// auto_firm, reference, source, timestamp, acknowledgements, retired.
type Intention struct {
	ID               string
	Subject          string
	Title            string
	Description      string
	Duration         *temporal.DurationSpec
	Window           *temporal.Window
	Stability        string
	Activity         string
	Location         []string
	Parties          []string
	Serves           []Ref
	Cadence          *temporal.Cadence
	Occurrence       string
	Placement        *temporal.Placement
	Preference       string
	AutoSelect       *Policy
	AutoFirm         *Policy
	Reference        string
	Source           Source
	Timestamp        time.Time
	Acknowledgements []Acknowledgement
	Retired          *Retired
	Version          string // cached; the computed value is authoritative
	Extras           []Extra
}

// Availability is a standing statement of capacity. Canonical order: id,
// subject, title, description, duration, window, conditional, location,
// cadence, valid_until, scope, source, timestamp, retired.
type Availability struct {
	ID          string
	Subject     string
	Title       string
	Description string
	Duration    *temporal.DurationSpec
	Window      *temporal.Window
	Conditional []string
	Location    []string
	Cadence     *temporal.Cadence
	ValidUntil  temporal.ValidUntil
	Scope       string
	Source      Source
	Timestamp   time.Time
	Retired     *Retired
	Version     string
	Extras      []Extra
}

// Commitment is the interpersonal object. Canonical order: id, parties,
// placement, intention, origin, transparent, external, title, description,
// source, timestamp, acknowledgements, retired.
type Commitment struct {
	ID               string
	Parties          []Party
	Placement        *temporal.Placement
	Intention        string
	Origin           Origin
	Transparent      *bool
	External         *External
	Title            string
	Description      string
	Source           Source
	Timestamp        time.Time
	Acknowledgements []Acknowledgement
	Retired          *Retired
	Version          string
	Extras           []Extra
}

// Resolution is the record of a selection. Canonical order: id, intention,
// placement, selector, candidates_considered, displaced, source, timestamp.
type Resolution struct {
	ID                   string
	Intention            string
	Placement            *temporal.Placement
	Selector             string
	CandidatesConsidered *int
	Displaced            []string
	Source               Source
	Timestamp            time.Time
	Version              string
	Extras               []Extra
}

// Object is any file-backed object or record.
type Object interface {
	GetID() string
	GetType() Type
	GetSource() Source
	GetRetired() *Retired
	// Refs returns every outbound reference id, sorted and unique.
	Refs() []string
	// SubjectURI is the subject, or empty for types that have none.
	SubjectURI() string
	// CachedVersion is the version field as read from the file.
	CachedVersion() string
	SetVersion(v string)
}

func (o *Intention) GetID() string         { return o.ID }
func (o *Intention) GetType() Type         { return TypeIntention }
func (o *Intention) GetSource() Source     { return o.Source }
func (o *Intention) GetRetired() *Retired  { return o.Retired }
func (o *Intention) SubjectURI() string    { return o.Subject }
func (o *Intention) CachedVersion() string { return o.Version }
func (o *Intention) SetVersion(v string)   { o.Version = v }
func (o *Intention) Refs() []string {
	var ids []string
	for _, r := range o.Serves {
		ids = append(ids, r.ID)
	}
	if o.Window != nil && o.Window.Relative != nil {
		ids = append(ids, o.Window.Relative.Target)
	}
	if o.Retired != nil && o.Retired.SupersededBy != "" {
		ids = append(ids, o.Retired.SupersededBy)
	}
	return uniqueSorted(ids)
}

// IsStanding reports whether the intention carries a cadence.
func (o *Intention) IsStanding() bool { return o.Cadence != nil }

// InstanceOf returns the standing intention id when this is an instance.
func (o *Intention) InstanceOf() string {
	for _, r := range o.Serves {
		if r.Role == RoleInstanceOf {
			return r.ID
		}
	}
	return ""
}

func (o *Availability) GetID() string         { return o.ID }
func (o *Availability) GetType() Type         { return TypeAvailability }
func (o *Availability) GetSource() Source     { return o.Source }
func (o *Availability) GetRetired() *Retired  { return o.Retired }
func (o *Availability) SubjectURI() string    { return o.Subject }
func (o *Availability) CachedVersion() string { return o.Version }
func (o *Availability) SetVersion(v string)   { o.Version = v }
func (o *Availability) Refs() []string {
	var ids []string
	if o.Window != nil && o.Window.Relative != nil {
		ids = append(ids, o.Window.Relative.Target)
	}
	if o.Retired != nil && o.Retired.SupersededBy != "" {
		ids = append(ids, o.Retired.SupersededBy)
	}
	return uniqueSorted(ids)
}

func (o *Commitment) GetID() string         { return o.ID }
func (o *Commitment) GetType() Type         { return TypeCommitment }
func (o *Commitment) GetSource() Source     { return o.Source }
func (o *Commitment) GetRetired() *Retired  { return o.Retired }
func (o *Commitment) SubjectURI() string    { return "" }
func (o *Commitment) CachedVersion() string { return o.Version }
func (o *Commitment) SetVersion(v string)   { o.Version = v }
func (o *Commitment) Refs() []string {
	var ids []string
	if o.Intention != "" {
		ids = append(ids, o.Intention)
	}
	if o.Origin.Resolution != "" {
		ids = append(ids, o.Origin.Resolution)
	}
	if o.Retired != nil && o.Retired.SupersededBy != "" {
		ids = append(ids, o.Retired.SupersededBy)
	}
	return uniqueSorted(ids)
}

// IsTransparent reports the effective transparency; absent means false.
func (o *Commitment) IsTransparent() bool { return o.Transparent != nil && *o.Transparent }

func (o *Resolution) GetID() string         { return o.ID }
func (o *Resolution) GetType() Type         { return TypeResolution }
func (o *Resolution) GetSource() Source     { return o.Source }
func (o *Resolution) GetRetired() *Retired  { return nil }
func (o *Resolution) SubjectURI() string    { return "" }
func (o *Resolution) CachedVersion() string { return o.Version }
func (o *Resolution) SetVersion(v string)   { o.Version = v }
func (o *Resolution) Refs() []string {
	ids := []string{}
	if o.Intention != "" {
		ids = append(ids, o.Intention)
	}
	ids = append(ids, o.Displaced...)
	return uniqueSorted(ids)
}

// NewObject returns an empty object of the type.
func NewObject(t Type) Object {
	switch t {
	case TypeIntention:
		return &Intention{}
	case TypeAvailability:
		return &Availability{}
	case TypeCommitment:
		return &Commitment{}
	case TypeResolution:
		return &Resolution{}
	}
	return nil
}
