package mcp

import (
	"context"
	"encoding/json"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nodelogicau/intentions-cli/internal/apperr"
	"github.com/nodelogicau/intentions-cli/internal/consistency"
	"github.com/nodelogicau/intentions-cli/internal/importer"
	"github.com/nodelogicau/intentions-cli/internal/query"
	"github.com/nodelogicau/intentions-cli/internal/render"
	"github.com/nodelogicau/intentions-cli/internal/resolve"
)

func (s *Server) registerImportTools() {
	sdk.AddTool(s.srv, &sdk.Tool{Name: "commitment_import", Annotations: additive,
		Description: "Import one calendar occurrence you have parsed as a commitment. Pass `event` as {system: icalendar|jscalendar, uid, start, duration, recurrence_id? (the occurrence's original start, for every occurrence of a recurring event), title?, location? (a URI only; drop free text), transparent?, organizer? {address, status?}, attendees? [{address, status?, non_participant?}]}: address is the calendar address (JSCalendar sendTo, imip first), status is PARTSTAT or participationStatus, non_participant marks ROLE=NON-PARTICIPANT or roles holding only informational. JSCalendar participants all go in attendees. Parties are mapped by rule here; never write a commitment file yourself. outcome is created, updated (a re-import: the commitment keeps the intention it was adopted for), unchanged, recognised (the event is an intention's own hold: nothing imported) or retired (the person cancelled it: it stays cancelled). Results equal `intentions commitment import --json`."},
		s.commitmentImport)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "commitment_adopt", Annotations: additive,
		Description: "Record that an imported event is the occasion of one of the person's intentions: a selection by the person (selector person) that gives the intention the event's placement and names it on the commitment. Only the person can say an event is their intention's occasion, so call it only on their word, never on your own judgement or under a policy. Offer it when a directed_at_no_one commitment looks like an intention's occasion, and call it again to fix adopted_moved after the event moves. Refused for an intention with parties or a hold link, one whose subject is not a party, or one another commitment already names; a placed intention needs replace. Results equal `intentions commitment adopt --json`."},
		s.commitmentAdopt)
}

type importIn struct {
	Event      map[string]any `json:"event" jsonschema:"the parsed occurrence"`
	Subject    string         `json:"subject,omitempty" jsonschema:"the particular the import is made for (default: defaults.subject)"`
	Commitment string         `json:"commitment,omitempty" jsonschema:"the commitment this occurrence updates, when the match is ambiguous"`
	Timestamp  string         `json:"timestamp,omitempty"`
	Source     *sourceIn      `json:"source,omitempty"`
}

func (s *Server) commitmentImport(ctx context.Context, req *sdk.CallToolRequest, in importIn) (*sdk.CallToolResult, any, error) {
	data, err := json.Marshal(in.Event)
	if err != nil || in.Event == nil {
		return errResult(apperr.Usage("event is required")), nil, nil
	}
	ev, err := importer.Decode(data)
	if err != nil {
		return errResult(err), nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	subject := in.Subject
	if subject == "" {
		subject = s.ws.Config.Defaults.Subject
	}
	if subject == "" {
		return errResult(apperr.Usage("an import is made for one particular: pass subject, or set defaults.subject")), nil, nil
	}
	at, err := now("")
	if err != nil {
		return errResult(err), nil, nil
	}
	ts, err := timestamp(in.Timestamp, at)
	if err != nil {
		return errResult(err), nil, nil
	}
	r, err := importer.Import(s.ws, g, ev, importer.Options{Subject: subject, Commitment: in.Commitment, Source: s.source(req, in.Source), Timestamp: ts})
	if err != nil {
		return errResult(err), nil, nil
	}
	out := render.Import(r, nil, nil)
	if r.Commitment != nil && r.Commitment.Retired == nil {
		e, err := s.env(g, at, "", "")
		if err != nil {
			return errResult(err), nil, nil
		}
		flags, err := consistency.Check(e, []string{r.Commitment.ID})
		if err != nil {
			return errResult(err), nil, nil
		}
		out = render.Import(r, flags, query.ImportFindings(g, r.Commitment))
	}
	var fs []query.Finding
	if f, ok := out["findings"].([]query.Finding); ok {
		fs = f
	}
	return okResult(render.ImportText(r) + findingsText(fs)), out, nil
}

type commitmentAdoptIn struct {
	ID        string    `json:"id" jsonschema:"the imported commitment"`
	Intention string    `json:"intention" jsonschema:"the intention whose occasion it is"`
	Replace   bool      `json:"replace,omitempty" jsonschema:"place an already placed intention at the event's time instead"`
	Now       string    `json:"now,omitempty"`
	Source    *sourceIn `json:"source,omitempty"`
}

func (s *Server) commitmentAdopt(ctx context.Context, req *sdk.CallToolRequest, in commitmentAdoptIn) (*sdk.CallToolResult, any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	c, err := getCommitment(g, in.ID)
	if err != nil {
		return errResult(err), nil, nil
	}
	target, err := getIntention(g, in.Intention)
	if err != nil {
		return errResult(err), nil, nil
	}
	at, err := now(in.Now)
	if err != nil {
		return errResult(err), nil, nil
	}
	e, err := s.env(g, at, "", "")
	if err != nil {
		return errResult(err), nil, nil
	}
	sel, err := resolve.Adopt(e, c, target, resolve.AdoptOptions{Source: s.source(req, in.Source), Timestamp: at, Replace: in.Replace})
	if err != nil {
		return errResult(err), nil, nil
	}
	flags, err := consistency.Check(e, []string{sel.Intention.ID, sel.Resolution.ID, sel.Adopted.ID})
	if err != nil {
		return errResult(err), nil, nil
	}
	return okResult(sel.Describe()), render.Selection(sel, resolve.Result{Considered: 1}, flags, ""), nil
}
