// Package store owns the workspace on disk: intentions.yaml, discovery, the
// object files, atomic writes, loading everything into a graph, and the
// derived index.
package store

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/temporal"
)

// File names.
const (
	ConfigFile      = "intentions.yaml"
	IndexFile       = "index.yaml"
	ConventionsFile = "intentions.md"
	PointerFile     = ".intentions"
	EnvWorkspace    = "INTENTIONS_WORKSPACE"
)

// Config is intentions.yaml as this implementation reads it. Keys it does not
// know are preserved in the underlying document and survive a rewrite.
type Config struct {
	Format   string `json:"format"`
	Hash     string `json:"hash"`
	Resolver struct {
		Timezone   string `json:"timezone"`
		Hemisphere string `json:"hemisphere,omitempty"`
		Step       string `json:"step,omitempty"`    // candidate grid; absent means PT15M
		Horizon    string `json:"horizon,omitempty"` // how far ahead resolution looks; absent means P4W
		Scope      string `json:"scope,omitempty"`   // the resolver's own scope; absent means personal
	} `json:"resolver"`
	// UnknownResolverKeys are keys under resolver this implementation does
	// not know (a stale week_start, say): ignored, reported at info level.
	UnknownResolverKeys []string `json:"-"`
	Availability        struct {
		DefaultHorizon string `json:"default_horizon"`
	} `json:"availability"`
	// StaleGeneration records a generation.horizon left by an older writer.
	// resolver.horizon is the one planning horizon; this is ignored and
	// reported at info level, as a stale week_start is.
	StaleGeneration string `json:"-"`
	Defaults        struct {
		Subject string       `json:"subject,omitempty"`
		Source  model.Source `json:"source"`
	} `json:"defaults"`

	doc *yaml.Node
}

// NewConfig returns the recommended defaults for a fresh workspace.
func NewConfig() Config {
	var c Config
	c.Format = model.Format
	c.Hash = "sha256"
	c.Resolver.Timezone = "UTC"
	c.Resolver.Step = "PT15M"
	c.Resolver.Horizon = "P4W"
	c.Availability.DefaultHorizon = "P13W"
	return c
}

// ParseConfig reads intentions.yaml.
func ParseConfig(data []byte) (Config, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return Config{}, fmt.Errorf("%s: %v", ConfigFile, err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return Config{}, fmt.Errorf("%s: must be a YAML mapping", ConfigFile)
	}
	c := Config{doc: &doc}
	root := doc.Content[0]
	c.Format = getPath(root, "format")
	c.Hash = getPath(root, "hash")
	c.Resolver.Timezone = getPath(root, "resolver", "timezone")
	c.Resolver.Hemisphere = getPath(root, "resolver", "hemisphere")
	c.Resolver.Step = getPath(root, "resolver", "step")
	c.Resolver.Horizon = getPath(root, "resolver", "horizon")
	c.Resolver.Scope = getPath(root, "resolver", "scope")
	c.UnknownResolverKeys = unknownKeys(root, []string{"timezone", "hemisphere", "step", "horizon", "scope"}, "resolver")
	c.Availability.DefaultHorizon = getPath(root, "availability", "default_horizon")
	c.StaleGeneration = getPath(root, "generation", "horizon")
	c.Defaults.Subject = getPath(root, "defaults", "subject")
	c.Defaults.Source.Author = getPath(root, "defaults", "source", "author")
	c.Defaults.Source.Harness = getPath(root, "defaults", "source", "harness")
	c.Defaults.Source.Model = getPath(root, "defaults", "source", "model")
	return c, nil
}

// Validate checks the configuration this implementation depends on.
func (c Config) Validate() error {
	if c.Format != model.Format {
		return fmt.Errorf("%s: format %q is not supported; this implementation reads %s", ConfigFile, c.Format, model.Format)
	}
	if c.Hash != "sha256" {
		return fmt.Errorf("%s: hash %q is not admitted; this format version admits only sha256", ConfigFile, c.Hash)
	}
	if c.Resolver.Timezone == "" {
		return fmt.Errorf("%s: resolver.timezone is required", ConfigFile)
	}
	if _, err := time.LoadLocation(c.Resolver.Timezone); err != nil {
		return fmt.Errorf("%s: resolver.timezone %q is not a known IANA zone", ConfigFile, c.Resolver.Timezone)
	}
	if _, err := temporal.ParseHemisphere(c.Resolver.Hemisphere); err != nil {
		return fmt.Errorf("%s: resolver.hemisphere: %v", ConfigFile, err)
	}
	if c.Resolver.Step != "" {
		if d, err := temporal.ParseDuration(c.Resolver.Step); err != nil || d.IsZero() || !d.HasTimePart() || d.Years+d.Months+d.Weeks+d.Days != 0 {
			return fmt.Errorf("%s: resolver.step %q must be a positive time-of-day duration such as PT15M", ConfigFile, c.Resolver.Step)
		}
	}
	if c.Resolver.Horizon != "" {
		if _, err := temporal.ParseDuration(c.Resolver.Horizon); err != nil {
			return fmt.Errorf("%s: resolver.horizon: %v", ConfigFile, err)
		}
	}
	if c.Resolver.Scope != "" && !oneOfScope(c.Resolver.Scope) {
		return fmt.Errorf("%s: resolver.scope %q must be personal, organisation, or public", ConfigFile, c.Resolver.Scope)
	}
	if _, err := temporal.ParseDuration(c.Availability.DefaultHorizon); err != nil {
		return fmt.Errorf("%s: availability.default_horizon: %v", ConfigFile, err)
	}
	if strings.TrimSpace(c.Defaults.Source.Author) == "" {
		return fmt.Errorf("%s: defaults.source.author is required", ConfigFile)
	}
	if c.Defaults.Subject != "" && !model.ValidURI(c.Defaults.Subject) {
		return fmt.Errorf("%s: defaults.subject %q is not an absolute URI", ConfigFile, c.Defaults.Subject)
	}
	return nil
}

// Context builds the resolver context, with now injected.
func (c Config) Context(now time.Time) temporal.Context {
	loc, err := time.LoadLocation(c.Resolver.Timezone)
	if err != nil {
		loc = time.UTC
	}
	h, _ := temporal.ParseHemisphere(c.Resolver.Hemisphere)
	return temporal.Context{Location: loc, Hemisphere: h, Now: now}
}

// DefaultHorizon returns availability.default_horizon parsed.
func (c Config) DefaultHorizon() temporal.Duration {
	d, _ := temporal.ParseDuration(c.Availability.DefaultHorizon)
	return d
}

// GenerationHorizon is the horizon instance generation runs over: the same
// resolver.horizon resolution uses, so the two cannot drift.
func (c Config) GenerationHorizon() temporal.Duration {
	return c.ResolverHorizon()
}

// Step returns resolver.step parsed, PT15M when absent.
func (c Config) Step() temporal.Duration {
	if c.Resolver.Step == "" {
		return temporal.Duration{Minutes: 15}
	}
	d, _ := temporal.ParseDuration(c.Resolver.Step)
	return d
}

// ResolverHorizon returns resolver.horizon parsed, P4W when absent.
func (c Config) ResolverHorizon() temporal.Duration {
	if c.Resolver.Horizon == "" {
		return temporal.Duration{Weeks: 4}
	}
	d, _ := temporal.ParseDuration(c.Resolver.Horizon)
	return d
}

// ResolverScope returns resolver.scope, personal when absent.
func (c Config) ResolverScope() string {
	if c.Resolver.Scope == "" {
		return "personal"
	}
	return c.Resolver.Scope
}

func oneOfScope(s string) bool {
	for _, x := range model.Scopes {
		if x == s {
			return true
		}
	}
	return false
}

// Marshal renders the configuration, preserving unknown keys from the file
// it was read from and writing known keys in canonical order otherwise.
func (c Config) Marshal() ([]byte, error) {
	var root *yaml.Node
	if c.doc != nil {
		root = c.doc.Content[0]
	} else {
		root = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	}
	setPath(root, c.Format, "format")
	setPath(root, c.Hash, "hash")
	setPath(root, c.Resolver.Timezone, "resolver", "timezone")
	if c.Resolver.Hemisphere != "" {
		setPath(root, c.Resolver.Hemisphere, "resolver", "hemisphere")
	}
	if c.Resolver.Step != "" {
		setPath(root, c.Resolver.Step, "resolver", "step")
	}
	if c.Resolver.Horizon != "" {
		setPath(root, c.Resolver.Horizon, "resolver", "horizon")
	}
	if c.Resolver.Scope != "" {
		setPath(root, c.Resolver.Scope, "resolver", "scope")
	}
	setPath(root, c.Availability.DefaultHorizon, "availability", "default_horizon")
	if c.Defaults.Subject != "" {
		setPath(root, c.Defaults.Subject, "defaults", "subject")
	}
	setPath(root, c.Defaults.Source.Author, "defaults", "source", "author")
	if c.Defaults.Source.Harness != "" {
		setPath(root, c.Defaults.Source.Harness, "defaults", "source", "harness")
	}
	if c.Defaults.Source.Model != "" {
		setPath(root, c.Defaults.Source.Model, "defaults", "source", "model")
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return nil, err
	}
	_ = enc.Close()
	return buf.Bytes(), nil
}

// unknownKeys lists the keys of the mapping at path that are not in known.
func unknownKeys(m *yaml.Node, known []string, path ...string) []string {
	n := m
	for _, key := range path {
		var next *yaml.Node
		for i := 0; i+1 < len(n.Content); i += 2 {
			if n.Content[i].Value == key {
				next = n.Content[i+1]
			}
		}
		if next == nil || next.Kind != yaml.MappingNode {
			return nil
		}
		n = next
	}
	var out []string
	for i := 0; i+1 < len(n.Content); i += 2 {
		k := n.Content[i].Value
		isKnown := false
		for _, x := range known {
			if x == k {
				isKnown = true
			}
		}
		if !isKnown {
			out = append(out, k)
		}
	}
	return out
}

func getPath(m *yaml.Node, path ...string) string {
	n := m
	for _, key := range path {
		if n == nil || n.Kind != yaml.MappingNode {
			return ""
		}
		var next *yaml.Node
		for i := 0; i+1 < len(n.Content); i += 2 {
			if n.Content[i].Value == key {
				next = n.Content[i+1]
				break
			}
		}
		n = next
	}
	if n == nil || n.Kind != yaml.ScalarNode {
		return ""
	}
	return n.Value
}

func setPath(m *yaml.Node, value string, path ...string) {
	n := m
	for i, key := range path {
		last := i == len(path)-1
		var child *yaml.Node
		for j := 0; j+1 < len(n.Content); j += 2 {
			if n.Content[j].Value == key {
				child = n.Content[j+1]
				if last {
					child.Kind, child.Tag, child.Value, child.Style, child.Content = yaml.ScalarNode, "!!str", value, 0, nil
					return
				}
				break
			}
		}
		if child == nil {
			if last {
				child = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
			} else {
				child = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			}
			n.Content = append(n.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, child)
			if last {
				return
			}
		}
		if child.Kind != yaml.MappingNode {
			child.Kind, child.Tag, child.Value, child.Content = yaml.MappingNode, "!!map", "", nil
		}
		n = child
	}
}
