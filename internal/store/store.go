package store

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"app/internal/fx"
)

// Profile maps part keys (fx.Key) to settings. Missing keys mean native.
type Profile map[string]fx.Setting

type file struct {
	Version  int                `json:"version"`
	Active   string             `json:"active"`
	Profiles map[string]Profile `json:"profiles"`
}

type Store struct {
	mu   sync.Mutex
	path string
	f    file
}

func Dir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base = "."
	}
	return filepath.Join(base, "ElectricEditor")
}

func Load() *Store {
	s := &Store{path: filepath.Join(Dir(), "profiles.json")}
	if b, err := os.ReadFile(s.path); err == nil {
		_ = json.Unmarshal(b, &s.f)
	}
	if s.f.Profiles == nil {
		s.f.Profiles = map[string]Profile{}
	}
	if len(s.f.Profiles) == 0 {
		s.f.Profiles["Main"] = Profile{}
	}
	if _, ok := s.f.Profiles[s.f.Active]; !ok {
		s.f.Active = s.Names()[0]
	}
	s.f.Version = 1
	return s
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(s.f, "", "  ")
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) Names() []string {
	var out []string
	for k := range s.f.Profiles {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (s *Store) ActiveName() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.f.Active
}

func (s *Store) ProfileNames() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Names()
}

// Active returns a copy of the active profile.
func (s *Store) Active() Profile {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := Profile{}
	for k, v := range s.f.Profiles[s.f.Active] {
		out[k] = v
	}
	return out
}

func (s *Store) Set(key string, v fx.Setting) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.f.Profiles[s.f.Active]
	if v.Mode == "native" {
		delete(p, key)
	} else {
		p[key] = v
	}
	return s.saveLocked()
}

func cleanName(n string) (string, error) {
	n = strings.TrimSpace(n)
	if n == "" || len(n) > 40 {
		return "", errors.New("profile names need 1 to 40 characters")
	}
	return n, nil
}

func (s *Store) Switch(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.f.Profiles[name]; !ok {
		return fmt.Errorf("no profile called %q", name)
	}
	s.f.Active = name
	return s.saveLocked()
}

// Create makes a new profile, copying the active one, and switches to it.
func (s *Store) Create(name string) error {
	name, err := cleanName(name)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.f.Profiles[name]; ok {
		return fmt.Errorf("a profile called %q already exists", name)
	}
	cp := Profile{}
	for k, v := range s.f.Profiles[s.f.Active] {
		cp[k] = v
	}
	s.f.Profiles[name] = cp
	s.f.Active = name
	return s.saveLocked()
}

func (s *Store) Rename(name string) error {
	name, err := cleanName(name)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.f.Profiles[name]; ok {
		return fmt.Errorf("a profile called %q already exists", name)
	}
	s.f.Profiles[name] = s.f.Profiles[s.f.Active]
	delete(s.f.Profiles, s.f.Active)
	s.f.Active = name
	return s.saveLocked()
}

func (s *Store) Delete() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.f.Profiles) <= 1 {
		return errors.New("you need at least one profile")
	}
	delete(s.f.Profiles, s.f.Active)
	s.f.Active = s.Names()[0]
	return s.saveLocked()
}

// ------------------------------------------------------------ share codes
//
// EE1.<base64url(deflate(json))>  - compact enough to paste in Discord.

const codePrefix = "EE1."

type shared struct {
	Name  string  `json:"n,omitempty"`
	Parts Profile `json:"p"`
}

// Export encodes the active profile, or only keys starting with prefix
// (e.g. "deviljin.") when prefix is set.
func (s *Store) Export(prefix string) (string, error) {
	s.mu.Lock()
	sh := shared{Name: s.f.Active, Parts: Profile{}}
	for k, v := range s.f.Profiles[s.f.Active] {
		if prefix == "" || strings.HasPrefix(k, prefix) {
			sh.Parts[k] = v
		}
	}
	s.mu.Unlock()
	if len(sh.Parts) == 0 {
		return "", errors.New("nothing to share yet - recolour something first")
	}
	raw, _ := json.Marshal(sh)
	var buf bytes.Buffer
	w, _ := flate.NewWriter(&buf, flate.BestCompression)
	w.Write(raw)
	w.Close()
	return codePrefix + base64.RawURLEncoding.EncodeToString(buf.Bytes()), nil
}

// Import merges a share code into the active profile. Returns parts applied.
func (s *Store) Import(code string, valid map[string]fx.Part) (int, error) {
	code = strings.TrimSpace(code)
	if !strings.HasPrefix(code, codePrefix) {
		return 0, errors.New("that isn't an Electric Editor code (they start with EE1.)")
	}
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(code, codePrefix))
	if err != nil {
		return 0, errors.New("the code is incomplete - copy the whole thing")
	}
	raw, err := io.ReadAll(io.LimitReader(flate.NewReader(bytes.NewReader(b)), 1<<20))
	if err != nil {
		return 0, errors.New("the code is damaged - copy it again")
	}
	var sh shared
	if err := json.Unmarshal(raw, &sh); err != nil {
		return 0, errors.New("the code is damaged - copy it again")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.f.Profiles[s.f.Active]
	n := 0
	for k, v := range sh.Parts {
		if _, ok := valid[k]; !ok {
			continue
		}
		if v.Mode != "solid" && v.Mode != "gradient" {
			continue
		}
		if v.Brightness <= 0 || v.Brightness > 50 {
			v.Brightness = 6
		}
		if len(v.Stops) > 8 {
			v.Stops = v.Stops[:8]
		}
		p[k] = v
		n++
	}
	if n == 0 {
		return 0, errors.New("the code didn't contain any parts this version knows")
	}
	return n, s.saveLocked()
}
