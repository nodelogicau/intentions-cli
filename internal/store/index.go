package store

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/projection"
)

// Entry is one index entry: nothing not derivable from the file.
type Entry struct {
	ID      string   `yaml:"id" json:"id"`
	Type    string   `yaml:"type" json:"type"`
	Subject string   `yaml:"subject,omitempty" json:"subject,omitempty"`
	Path    string   `yaml:"path" json:"path"`
	Version string   `yaml:"version" json:"version"`
	Retired string   `yaml:"retired,omitempty" json:"retired,omitempty"`
	Refs    []string `yaml:"refs,omitempty" json:"refs,omitempty"`
}

// Index is the derived cache of the object files.
type Index struct {
	Format  string  `yaml:"format" json:"format"`
	Entries []Entry `yaml:"entries" json:"entries"`
}

// EntryFor derives the entry for an object from the object alone.
func EntryFor(obj model.Object) Entry {
	e := Entry{
		ID:      obj.GetID(),
		Type:    string(obj.GetType()),
		Subject: obj.SubjectURI(),
		Path:    RelPath(obj.GetType(), obj.GetID()),
		Version: projection.MustVersion(obj),
		Refs:    obj.Refs(),
	}
	if r := obj.GetRetired(); r != nil {
		e.Retired = r.Kind
	}
	return e
}

// Upsert replaces or inserts an entry, keeping the list sorted by id. Entries
// of types this implementation does not know are left in place.
func (ix *Index) Upsert(e Entry) {
	for i := range ix.Entries {
		if ix.Entries[i].ID == e.ID {
			ix.Entries[i] = e
			ix.sort()
			return
		}
	}
	ix.Entries = append(ix.Entries, e)
	ix.sort()
}

func (ix *Index) sort() {
	sort.Slice(ix.Entries, func(i, j int) bool { return ix.Entries[i].ID < ix.Entries[j].ID })
}

// Rebuild derives the whole index from a loaded graph.
func Rebuild(g *Graph) *Index {
	ix := &Index{Format: model.Format, Entries: []Entry{}}
	// An object with value-level problems still gets an entry so the index
	// stays complete; its version reflects what parsed.
	for _, id := range g.Order {
		ix.Entries = append(ix.Entries, EntryFor(g.Objects[id]))
	}
	return ix
}

// Diff is the disagreement between two indexes.
type Diff struct {
	Missing []string `json:"missing,omitempty"` // in rebuilt, not in committed
	Extra   []string `json:"extra,omitempty"`   // in committed, not in rebuilt
	Changed []string `json:"changed,omitempty"` // in both, different
}

// Empty reports whether the indexes agree.
func (d Diff) Empty() bool { return len(d.Missing)+len(d.Extra)+len(d.Changed) == 0 }

// Compare diffs a committed index against a rebuilt one.
func Compare(committed, rebuilt *Index) Diff {
	var d Diff
	have := map[string]Entry{}
	for _, e := range committed.Entries {
		have[e.ID] = e
	}
	want := map[string]Entry{}
	for _, e := range rebuilt.Entries {
		want[e.ID] = e
	}
	for id, e := range want {
		h, ok := have[id]
		if !ok {
			d.Missing = append(d.Missing, id)
			continue
		}
		if !entryEqual(h, e) {
			d.Changed = append(d.Changed, id)
		}
	}
	for id := range have {
		if _, ok := want[id]; !ok {
			d.Extra = append(d.Extra, id)
		}
	}
	sort.Strings(d.Missing)
	sort.Strings(d.Extra)
	sort.Strings(d.Changed)
	return d
}

func entryEqual(a, b Entry) bool {
	if a.ID != b.ID || a.Type != b.Type || a.Subject != b.Subject || a.Path != b.Path || a.Version != b.Version || a.Retired != b.Retired || len(a.Refs) != len(b.Refs) {
		return false
	}
	for i := range a.Refs {
		if a.Refs[i] != b.Refs[i] {
			return false
		}
	}
	return true
}

// ReadIndex reads index.yaml. A missing file is an empty index; a file that
// does not parse (merge-conflict markers, say) is an error the caller may
// treat as "rebuild".
func (w *Workspace) ReadIndex() (*Index, error) {
	data, err := os.ReadFile(filepath.Join(w.Root, IndexFile))
	if err != nil {
		if os.IsNotExist(err) {
			return &Index{Format: model.Format}, nil
		}
		return nil, err
	}
	var ix Index
	if err := yaml.Unmarshal(data, &ix); err != nil {
		return nil, fmt.Errorf("%s: %v", IndexFile, err)
	}
	if ix.Entries == nil {
		ix.Entries = []Entry{}
	}
	ix.sort()
	return &ix, nil
}

// WriteIndex writes index.yaml deterministically.
func (w *Workspace) WriteIndex(ix *Index) error {
	if ix.Entries == nil {
		ix.Entries = []Entry{}
	}
	ix.sort()
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(ix); err != nil {
		return err
	}
	_ = enc.Close()
	return atomicWrite(filepath.Join(w.Root, IndexFile), buf.Bytes())
}
