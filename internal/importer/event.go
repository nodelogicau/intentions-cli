// Package importer brings one calendar occurrence into the workspace as a
// commitment. The harness parses iCalendar or JSCalendar and passes the
// occurrence as an Event; the mapping from participants to parties, hold
// recognition, and the choice between creating and updating are made here,
// so every harness writes the same commitment from the same event.
package importer

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/nodelogicau/intentions-cli/internal/apperr"
	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// Entry is one organizer or attendee: its calendar address, its
// participation status as iCalendar PARTSTAT or JSCalendar
// participationStatus, and whether it only receives information.
type Entry struct {
	Address        string `json:"address"`
	Status         string `json:"status,omitempty"`
	NonParticipant bool   `json:"non_participant,omitempty"`
}

// Event is one parsed occurrence.
type Event struct {
	System   string `json:"system"`
	UID      string `json:"uid"`
	Start    string `json:"start"`
	Duration string `json:"duration"`
	// RecurrenceID is an occurrence's original start (iCalendar
	// RECURRENCE-ID, JSCalendar recurrenceId), present for an occurrence of
	// a recurring event: it tells a new occurrence from a moved one.
	RecurrenceID string  `json:"recurrence_id,omitempty"`
	Title        string  `json:"title,omitempty"`
	Location     string  `json:"location,omitempty"`
	Transparent  bool    `json:"transparent,omitempty"`
	Organizer    *Entry  `json:"organizer,omitempty"`
	Attendees    []Entry `json:"attendees,omitempty"`
	// TimeZone is the zone the event's start names (iCalendar
	// DTSTART;TZID=, JSCalendar timeZone). A known IANA name is kept as the
	// start's RFC 9557 suffix.
	TimeZone string `json:"time_zone,omitempty"`

	placement  temporal.Placement
	recurrence *temporal.PlacementStart
	dropped    *ZoneDropped
}

// ZoneDropped reports a time_zone the import could not keep.
type ZoneDropped struct {
	TimeZone string `json:"time_zone"`
	Reason   string `json:"reason"`
}

// Decode reads an event strictly: unknown keys, a missing required field,
// an unparseable start or duration, or a location that is not a URI are
// usage errors.
func Decode(data []byte) (Event, error) {
	var ev Event
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&ev); err != nil {
		return ev, apperr.Usage("event: %v", err)
	}
	return ev, ev.check()
}

func (ev *Event) check() error {
	if ev.System != model.ExternalSystems[0] && ev.System != model.ExternalSystems[1] {
		return apperr.Usage("event: system must be icalendar or jscalendar, not %q", ev.System)
	}
	if strings.TrimSpace(ev.UID) == "" {
		return apperr.Usage("event: uid is required")
	}
	if ev.Start == "" || ev.Duration == "" {
		return apperr.Usage("event: start and duration are required")
	}
	start, err := temporal.ParsePlacementStart(ev.Start)
	if err != nil {
		return apperr.Usage("event: %v", err)
	}
	if start, err = ev.zone(start); err != nil {
		return err
	}
	d, err := temporal.ParseDuration(ev.Duration)
	if err != nil || d.IsZero() {
		return apperr.Usage("event: duration %q must be a positive ISO 8601 duration", ev.Duration)
	}
	if ev.Location != "" && !model.ValidURI(ev.Location) {
		return apperr.Usage("event: location %q is not a URI; a free-text location does not cross", ev.Location)
	}
	if ev.RecurrenceID != "" {
		r, err := temporal.ParsePlacementStart(ev.RecurrenceID)
		if err != nil {
			return apperr.Usage("event: recurrence_id: %v", err)
		}
		ev.recurrence = &r
	}
	ev.placement = temporal.Placement{Start: start, Duration: temporal.DurationSpec{Nominal: d}, Location: ev.Location}
	if err := ev.placement.Validate(); err != nil {
		return apperr.Usage("event: %v", err)
	}
	entries := ev.Attendees
	if ev.Organizer != nil {
		entries = append([]Entry{*ev.Organizer}, entries...)
	}
	for _, e := range entries {
		if _, err := mapStatus(e.Status); err != nil {
			return err
		}
	}
	return nil
}

// zone applies time_zone to the start: a known IANA name is appended as the
// suffix with the offset kept as given; UTC gives none; an unknown name is
// dropped and reported. A suffix already on start must agree.
func (ev *Event) zone(start temporal.PlacementStart) (temporal.PlacementStart, error) {
	tz := strings.TrimSpace(ev.TimeZone)
	switch {
	case start.Zone != "":
		if tz != "" && tz != start.Zone {
			return start, apperr.Usage("event: start carries [%s] but time_zone is %q; they must name the same zone", start.Zone, tz)
		}
		return start, nil
	case tz == "" || start.AllDay || temporal.IsUTCZone(tz) || start.UTCStart():
		return start, nil
	}
	if _, err := temporal.LoadZone(tz); err != nil {
		ev.dropped = &ZoneDropped{TimeZone: tz, Reason: "not an IANA time-zone name this binary's tz database knows"}
		return start, nil
	}
	zoned, err := temporal.ParsePlacementStart(start.Raw + "[" + tz + "]")
	if err != nil {
		ev.dropped = &ZoneDropped{TimeZone: tz, Reason: "not an IANA time-zone name"}
		return start, nil
	}
	return zoned, nil
}

// Dropped is the time_zone the import could not keep, if any.
func (ev Event) Dropped() *ZoneDropped { return ev.dropped }

// Placement is the occurrence's placement, valid after Decode.
func (ev Event) Placement() temporal.Placement { return ev.placement }

// External is the occurrence's link.
func (ev Event) External() model.External { return model.External{System: ev.System, UID: ev.UID} }

func mapStatus(s string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "ACCEPTED":
		return "accepted", nil
	case "", "TENTATIVE", "NEEDS-ACTION":
		return "tentative", nil
	case "DECLINED", "DELEGATED":
		return "declined", nil
	}
	return "", apperr.Usage("event: participation status %q is not one of ACCEPTED, TENTATIVE, NEEDS-ACTION, DECLINED, DELEGATED", s)
}

// Resolver maps a calendar address to the subject calendar.addresses lists
// it under.
type Resolver func(address string) (string, bool)

// Parties maps the event's organizer and attendees to parties for subject,
// sorted by URI. The subject is always a party.
func Parties(ev Event, subject string, lookup Resolver, normalise func(string) string) []model.Party {
	named := ev.Organizer != nil || len(ev.Attendees) > 0
	status := map[string]string{}
	add := func(e Entry, attendee bool) {
		if e.NonParticipant || strings.TrimSpace(e.Address) == "" {
			return
		}
		uri := normalise(e.Address)
		if s, ok := lookup(e.Address); ok {
			uri = s
		}
		st := "accepted" // an organizer with no attendee entry: organising is their act
		if attendee {
			st, _ = mapStatus(e.Status)
		}
		status[uri] = st // the organizer goes first, so an attendee entry wins
	}
	if ev.Organizer != nil {
		add(*ev.Organizer, false)
	}
	for _, a := range ev.Attendees {
		add(a, true)
	}
	if _, ok := status[subject]; !ok {
		if named {
			status[subject] = "tentative"
		} else {
			status[subject] = "accepted"
		}
	}
	var out []model.Party
	for uri, st := range status {
		out = append(out, model.Party{URI: uri, Status: st})
	}
	return model.SortParties(out)
}
