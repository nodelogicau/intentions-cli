package mcp

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nodelogicau/intentions-cli/internal/apperr"
	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/projection"
	"github.com/nodelogicau/intentions-cli/internal/render"
	"github.com/nodelogicau/intentions-cli/internal/resolve"
	"github.com/nodelogicau/intentions-cli/internal/store"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

func (s *Server) registerCommitmentTools() {
	sdk.AddTool(s.srv, &sdk.Tool{Name: "commitment_accept", Annotations: additive,
		Description: "Record that a party has accepted this commitment. A party's status is a fact about their will: no policy authorises this and nothing infers it, so call it only when the person has said so in their own words. The party defaults to the workspace's subject; answering for anyone else is an iTIP reply, which arrives by import."},
		s.commitmentAccept)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "commitment_decline", Annotations: additive,
		Description: "Record that a party has declined this commitment. Only on the person's word, exactly as with accept. Declining does not cancel the commitment or free the time; cancel does that."},
		s.commitmentDecline)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "commitment_cancel", Annotations: additive,
		Description: "Cancel a commitment: appends the terminal retirement kind `cancelled` and, when it names an intention that is still placed, clears that placement in the same act. The intention keeps its window and stability, is not retired, and appears in `unresolved` again. Nothing is deleted and the resolution record stays as history."},
		s.commitmentCancel)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "commitment_show", Annotations: readOnly,
		Description: "One commitment with its computed version, the intention it fulfils, and its current consistency flags."},
		s.commitmentShow)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "commitment_list", Annotations: readOnly,
		Description: "Commitments (active by default), filtered by party, party status, the intention fulfilled, or cancelled. Use it to see what the person has been asked to answer."},
		s.commitmentList)
}

type answerIn struct {
	ID     string    `json:"id" jsonschema:"the commitment being answered"`
	Party  string    `json:"party,omitempty" jsonschema:"the party answering (default: the workspace's defaults.subject)"`
	Source *sourceIn `json:"source,omitempty"`
}

func (s *Server) commitmentAccept(ctx context.Context, req *sdk.CallToolRequest, in answerIn) (*sdk.CallToolResult, any, error) {
	return s.answer(req, in, "accepted")
}

func (s *Server) commitmentDecline(ctx context.Context, req *sdk.CallToolRequest, in answerIn) (*sdk.CallToolResult, any, error) {
	return s.answer(req, in, "declined")
}

func (s *Server) answer(req *sdk.CallToolRequest, in answerIn, status string) (*sdk.CallToolResult, any, error) {
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
	src := s.source(req, in.Source)
	if src.Author == "" {
		return errResult(apperr.Refused("an author is required: pass source.author, or set defaults.source.author in intentions.yaml")), nil, nil
	}
	i, err := model.CheckAnswer(c, Clean(in.Party), status, s.ws.Config.Defaults.Subject)
	if err != nil {
		return errResult(err), nil, nil
	}
	prev, _ := projection.Version(c)
	o := *c
	o.Parties = append([]model.Party(nil), c.Parties...)
	o.Parties[i].Status = status
	if err := s.write(&o); err != nil {
		return errResult(err), nil, nil
	}
	g.Add(&o)
	out, err := render.Object(&o)
	if err != nil {
		return errResult(err), nil, nil
	}
	at, _ := now("")
	out["previous_version"], out["party"], out["status"] = prev, o.Parties[i].URI, status
	out["source"] = src
	out["flags"] = s.flagsOn(resolve.NewEnv(s.ws, g, at), o.ID)
	return okResult(fmt.Sprintf("%s is %s on %s", o.Parties[i].URI, status, o.ID)), out, nil
}

type cancelIn struct {
	ID        string    `json:"id"`
	Reason    string    `json:"reason,omitempty" jsonschema:"why it was cancelled"`
	Timestamp string    `json:"timestamp,omitempty"`
	Source    *sourceIn `json:"source,omitempty"`
}

func (s *Server) commitmentCancel(ctx context.Context, req *sdk.CallToolRequest, in cancelIn) (*sdk.CallToolResult, any, error) {
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
	at, err := now("")
	if err != nil {
		return errResult(err), nil, nil
	}
	ts, err := timestamp(in.Timestamp, at)
	if err != nil {
		return errResult(err), nil, nil
	}
	r := model.Retired{Kind: "cancelled", Reason: in.Reason, Source: s.source(req, in.Source), Timestamp: ts}
	if err := model.CheckRetirement(g, c, r); err != nil {
		return errResult(err), nil, nil
	}
	prev, _ := projection.Version(c)
	o := *c
	o.Retired = &r
	if err := s.write(&o); err != nil {
		return errResult(err), nil, nil
	}
	g.Add(&o)
	out, err := render.Object(&o)
	if err != nil {
		return errResult(err), nil, nil
	}
	out["previous_version"] = prev
	out["cancelled"] = map[string]any{"reason": in.Reason, "timestamp": temporal.FormatTimestamp(ts)}
	summary := "Cancelled " + o.ID
	if o.Intention != "" {
		if obj, ok := g.Get(o.Intention); ok {
			if in, ok := obj.(*model.Intention); ok && in.Retired == nil && in.Placement != nil {
				freed := *in
				freed.Placement = nil
				if err := s.write(&freed); err != nil {
					return errResult(err), nil, nil
				}
				g.Add(&freed)
				fm, err := render.Object(&freed)
				if err != nil {
					return errResult(err), nil, nil
				}
				out["freed"] = fm
				summary += "; " + freed.ID + " is unplaced again"
			}
		}
	}
	return okResult(summary), out, nil
}

func (s *Server) commitmentShow(ctx context.Context, req *sdk.CallToolRequest, in showIn) (*sdk.CallToolResult, any, error) {
	g, err := s.ws.Load()
	if err != nil {
		return errResult(err), nil, nil
	}
	c, err := getCommitment(g, in.ID)
	if err != nil {
		return errResult(err), nil, nil
	}
	at, err := now(in.Now)
	if err != nil {
		return errResult(err), nil, nil
	}
	out, err := render.Object(c)
	if err != nil {
		return errResult(err), nil, nil
	}
	if c.Intention != "" {
		entry := map[string]any{"id": c.Intention}
		if obj, ok := g.Get(c.Intention); ok {
			if in, ok := obj.(*model.Intention); ok {
				entry["title"], entry["placed"] = in.Title, in.Placement != nil
				if in.Retired != nil {
					entry["retired"] = in.Retired.Kind
				}
			}
		} else {
			entry["missing"] = true
		}
		out["intention_resolved"] = entry
	}
	out["flags"] = s.flagsOn(resolve.NewEnv(s.ws, g, at), c.ID)
	return okResult(fmt.Sprintf("%s %s", c.ID, c.Title)), out, nil
}

type commitmentListIn struct {
	Party     string `json:"party,omitempty"`
	Status    string `json:"status,omitempty" jsonschema:"tentative | accepted | declined"`
	Intention string `json:"intention,omitempty" jsonschema:"the intention fulfilled"`
	Cancelled bool   `json:"cancelled,omitempty" jsonschema:"list cancelled commitments instead of active"`
}

func (s *Server) commitmentList(ctx context.Context, req *sdk.CallToolRequest, in commitmentListIn) (*sdk.CallToolResult, any, error) {
	g, err := s.ws.Load()
	if err != nil {
		return errResult(err), nil, nil
	}
	if in.Status != "" && !model.ValidPartyStatus(in.Status) {
		return errResult(apperr.Usage("status must be one of %s", strings.Join(model.PartyStatuses, ", "))), nil, nil
	}
	var items []*model.Commitment
	for _, o := range g.Commitments() {
		if (o.Retired != nil) != in.Cancelled || (in.Intention != "" && o.Intention != in.Intention) {
			continue
		}
		if in.Party != "" || in.Status != "" {
			match := false
			for _, p := range o.Parties {
				if (in.Party == "" || p.URI == in.Party) && (in.Status == "" || p.Status == in.Status) {
					match = true
				}
			}
			if !match {
				continue
			}
		}
		items = append(items, o)
	}
	list, err := render.Objects(items)
	if err != nil {
		return errResult(err), nil, nil
	}
	return okResult(fmt.Sprintf("%d commitments", len(list))), map[string]any{"commitments": list, "count": len(list)}, nil
}

func getCommitment(g *store.Graph, id string) (*model.Commitment, error) {
	obj, err := getObject(g, id)
	if err != nil {
		return nil, err
	}
	c, ok := obj.(*model.Commitment)
	if !ok {
		return nil, apperr.Usage("%s is a %s, not a commitment", id, obj.GetType())
	}
	return c, nil
}
