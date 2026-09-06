package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nodelogicau/intentions-cli/internal/apperr"
	"github.com/nodelogicau/intentions-cli/internal/model"
	"github.com/nodelogicau/intentions-cli/internal/projection"
)

// Resolution records how a workspace was found.
type Resolution struct {
	Root    string `json:"root"`
	FoundBy string `json:"found_by"`          // flag | env | marker | pointer
	Pointer string `json:"pointer,omitempty"` // absolute path of the .intentions file
}

// Workspace is an opened Intentions directory.
type Workspace struct {
	Root   string
	Config Config
}

// Discover locates a workspace: explicit dir, then $INTENTIONS_WORKSPACE, then
// an upward search from the working directory for intentions.yaml or a
// .intentions pointer. An environment variable naming a directory without the
// marker is an error, never a fallback to the search.
func Discover(explicit string) (*Workspace, *Resolution, error) {
	if explicit != "" {
		return openExplicit(explicit, "flag")
	}
	if env := os.Getenv(EnvWorkspace); env != "" {
		return openExplicit(env, "env")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}
	return DiscoverFrom(cwd)
}

// DiscoverFrom walks up from dir.
func DiscoverFrom(dir string) (*Workspace, *Resolution, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, nil, err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ConfigFile)); err == nil {
			ws, err := Open(dir)
			if err != nil {
				return nil, nil, err
			}
			return ws, &Resolution{Root: ws.Root, FoundBy: "marker"}, nil
		}
		pointer := filepath.Join(dir, PointerFile)
		if target, ok, perr := readPointer(pointer); perr != nil {
			return nil, nil, perr
		} else if ok {
			if _, err := os.Stat(filepath.Join(target, ConfigFile)); err != nil {
				return nil, nil, apperr.NoWorkspace("%s points at %s, which has no %s", pointer, target, ConfigFile)
			}
			ws, err := Open(target)
			if err != nil {
				return nil, nil, err
			}
			return ws, &Resolution{Root: ws.Root, FoundBy: "pointer", Pointer: pointer}, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, nil, apperr.ErrNoWorkspace
		}
		dir = parent
	}
}

func openExplicit(dir, foundBy string) (*Workspace, *Resolution, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, nil, err
	}
	if _, err := os.Stat(filepath.Join(abs, ConfigFile)); err != nil {
		return nil, nil, apperr.NoWorkspace("%s has no %s", abs, ConfigFile)
	}
	ws, err := Open(abs)
	if err != nil {
		return nil, nil, err
	}
	return ws, &Resolution{Root: ws.Root, FoundBy: foundBy}, nil
}

// readPointer reads a .intentions file: its first non-blank, non-comment line
// is a path, resolved against the file's own directory.
func readPointer(path string) (string, bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !filepath.IsAbs(line) {
			line = filepath.Join(filepath.Dir(path), line)
		}
		return filepath.Clean(line), true, nil
	}
	return "", false, apperr.NoWorkspace("%s is empty", path)
}

// Open reads a workspace at root and validates its configuration.
func Open(root string) (*Workspace, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(abs, ConfigFile))
	if err != nil {
		return nil, apperr.NoWorkspace("%s has no %s", abs, ConfigFile)
	}
	cfg, err := ParseConfig(data)
	if err != nil {
		return nil, apperr.Usage("%v", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, apperr.Usage("%v", err)
	}
	return &Workspace{Root: abs, Config: cfg}, nil
}

// Init creates a workspace at dir. It refuses when the marker exists.
func Init(dir string, cfg Config) (*Workspace, []string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, nil, err
	}
	if _, err := os.Stat(filepath.Join(abs, ConfigFile)); err == nil {
		return nil, nil, apperr.Runtime(fmt.Errorf("%s already contains %s", abs, ConfigFile))
	}
	if err := cfg.Validate(); err != nil {
		return nil, nil, apperr.Usage("%v", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, nil, err
	}
	var created []string
	data, err := cfg.Marshal()
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(filepath.Join(abs, ConfigFile), data, 0o644); err != nil {
		return nil, nil, err
	}
	created = append(created, ConfigFile)
	for _, t := range model.Types {
		if err := os.MkdirAll(filepath.Join(abs, t.Dir()), 0o755); err != nil {
			return nil, nil, err
		}
		created = append(created, t.Dir()+"/")
	}
	ws := &Workspace{Root: abs, Config: cfg}
	if err := ws.WriteIndex(&Index{Format: model.Format}); err != nil {
		return nil, nil, err
	}
	created = append(created, IndexFile)
	if err := os.WriteFile(filepath.Join(abs, ConventionsFile), []byte(conventionsStub), 0o644); err != nil {
		return nil, nil, err
	}
	created = append(created, ConventionsFile)
	return ws, created, nil
}

const conventionsStub = `# Conventions

Prose for agents and people. Validation never reads this file.

## Activity terms

Terms are lowercase kebab-case, matched by exact equality between an
intention's ` + "`activity`" + ` and an availability's ` + "`conditional`" + `.

- deep-work
- meeting
`

// WritePointer writes a .intentions pointer file in dir naming target.
func WritePointer(dir, target string) (string, error) {
	p := filepath.Join(dir, PointerFile)
	if err := os.WriteFile(p, []byte(target+"\n"), 0o644); err != nil {
		return "", err
	}
	return p, nil
}

// Path returns the absolute path of an object file.
func (w *Workspace) Path(t model.Type, id string) string {
	return filepath.Join(w.Root, t.Dir(), id+".yaml")
}

// RelPath returns the workspace-relative path of an object file, with
// forward slashes.
func RelPath(t model.Type, id string) string {
	return t.Dir() + "/" + id + ".yaml"
}

// ReadObject reads one object by id.
func (w *Workspace) ReadObject(id string) (model.Object, model.Problems, error) {
	t, ok := model.TypeOfID(id)
	if !ok {
		return nil, nil, apperr.NotFound("%q is not an identifier", id)
	}
	data, err := os.ReadFile(w.Path(t, id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, apperr.NotFound("%s does not exist", id)
		}
		return nil, nil, err
	}
	obj, probs, err := model.Decode(data, t)
	if err != nil {
		return nil, nil, apperr.Runtime(fmt.Errorf("%s: %v", RelPath(t, id), err))
	}
	return obj, probs, nil
}

// WriteObject stamps the object's version, serialises it, writes the file
// atomically, and upserts the index entry.
func (w *Workspace) WriteObject(obj model.Object) error {
	if _, err := projection.Stamp(obj); err != nil {
		return err
	}
	data, err := model.Encode(obj)
	if err != nil {
		return err
	}
	path := w.Path(obj.GetType(), obj.GetID())
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := atomicWrite(path, data); err != nil {
		return err
	}
	idx, err := w.ReadIndex()
	if err != nil {
		idx = &Index{Format: model.Format}
	}
	idx.Upsert(EntryFor(obj))
	return w.WriteIndex(idx)
}

// atomicWrite writes data to a temporary file beside path and renames it
// over path, so a failure leaves the previous file intact.
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return err
	}
	return nil
}

// FileProblem is a file that could not be read as an object.
type FileProblem struct {
	Path string `json:"path"`
	ID   string `json:"id,omitempty"`
	Err  string `json:"error"`
}

// Loaded is one object with where it came from and what was wrong with it.
type Loaded struct {
	Object   model.Object
	Path     string // workspace-relative
	FileID   string // id implied by the file name
	Problems model.Problems
}

// Graph is the whole workspace in memory.
type Graph struct {
	Objects    map[string]model.Object
	Loaded     map[string]*Loaded
	Order      []string // ids sorted
	Unreadable []FileProblem
	inbound    map[string][]model.Inbound
	referrers  map[string][]string
}

// Get implements model.Graph.
func (g *Graph) Get(id string) (model.Object, bool) {
	o, ok := g.Objects[id]
	return o, ok
}

// Inbound implements model.Graph: serves references targeting id.
func (g *Graph) Inbound(id string) []model.Inbound { return g.inbound[id] }

// Referrers returns the ids of every object with an outbound reference to id.
func (g *Graph) Referrers(id string) []string { return g.referrers[id] }

// Intentions returns every intention in id order.
func (g *Graph) Intentions() []*model.Intention {
	var out []*model.Intention
	for _, id := range g.Order {
		if o, ok := g.Objects[id].(*model.Intention); ok {
			out = append(out, o)
		}
	}
	return out
}

// Availabilities returns every availability in id order.
func (g *Graph) Availabilities() []*model.Availability {
	var out []*model.Availability
	for _, id := range g.Order {
		if o, ok := g.Objects[id].(*model.Availability); ok {
			out = append(out, o)
		}
	}
	return out
}

// Commitments returns every commitment in id order.
func (g *Graph) Commitments() []*model.Commitment {
	var out []*model.Commitment
	for _, id := range g.Order {
		if o, ok := g.Objects[id].(*model.Commitment); ok {
			out = append(out, o)
		}
	}
	return out
}

// Resolutions returns every resolution record in id order.
func (g *Graph) Resolutions() []*model.Resolution {
	var out []*model.Resolution
	for _, id := range g.Order {
		if o, ok := g.Objects[id].(*model.Resolution); ok {
			out = append(out, o)
		}
	}
	return out
}

// Instances returns the intentions carrying instance-of the given recurring
// intention, retired ones included, in id order.
func (g *Graph) Instances(recurring string) []*model.Intention {
	var out []*model.Intention
	for _, in := range g.Intentions() {
		if in.InstanceOf() == recurring {
			out = append(out, in)
		}
	}
	return out
}

// Add registers an object (used by write paths to check a proposed state).
func (g *Graph) Add(obj model.Object) {
	if _, exists := g.Objects[obj.GetID()]; !exists {
		g.Order = append(g.Order, obj.GetID())
		sort.Strings(g.Order)
	}
	g.Objects[obj.GetID()] = obj
	g.index()
}

func (g *Graph) index() {
	g.inbound = map[string][]model.Inbound{}
	g.referrers = map[string][]string{}
	for _, id := range g.Order {
		obj := g.Objects[id]
		for _, ref := range obj.Refs() {
			g.referrers[ref] = append(g.referrers[ref], id)
		}
		if in, ok := obj.(*model.Intention); ok {
			for _, r := range in.Serves {
				g.inbound[r.ID] = append(g.inbound[r.ID], model.Inbound{From: id, Role: r.Role})
			}
		}
	}
}

// Load reads every object file in the four type directories. Files that do
// not parse are recorded in Unreadable rather than aborting the load; files
// whose id disagrees with their name are loaded under the file's id so that
// validation can report the mismatch.
func (w *Workspace) Load() (*Graph, error) {
	g := &Graph{Objects: map[string]model.Object{}, Loaded: map[string]*Loaded{}}
	for _, t := range model.Types {
		dir := filepath.Join(w.Root, t.Dir())
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".yaml") || strings.HasPrefix(name, ".") {
				continue
			}
			fileID := strings.TrimSuffix(name, ".yaml")
			rel := t.Dir() + "/" + name
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				g.Unreadable = append(g.Unreadable, FileProblem{Path: rel, ID: fileID, Err: err.Error()})
				continue
			}
			obj, probs, err := model.Decode(data, t)
			if err != nil {
				g.Unreadable = append(g.Unreadable, FileProblem{Path: rel, ID: fileID, Err: err.Error()})
				continue
			}
			key := fileID
			if _, dup := g.Objects[key]; dup {
				g.Unreadable = append(g.Unreadable, FileProblem{Path: rel, ID: fileID, Err: "duplicate id across directories"})
				continue
			}
			g.Objects[key] = obj
			g.Loaded[key] = &Loaded{Object: obj, Path: rel, FileID: fileID, Problems: probs}
			g.Order = append(g.Order, key)
		}
	}
	sort.Strings(g.Order)
	g.index()
	return g, nil
}
