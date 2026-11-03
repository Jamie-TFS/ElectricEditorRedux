package main

import (
	"bufio"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"app/internal/store"
)

// Keys unlock tiers (dev, amy, donator...) in the one public exe.
//
// A key is   EE-<base64url(payload)>.<base64url(signature)>
// signed with the maintainer's ed25519 key (tools/signer key ...). The
// signature covers licenseDomain+payload, so a key can never double as an
// update signature or vice versa. No key = regular user.
//
// Leaked keys are revoked by listing their ID in revoked.txt in the public
// repo; the app checks that file on launch.

const licenseDomain = "EE-LICENSE-v1:"

type License struct {
	ID     string `json:"i"`
	Tier   string `json:"t"`
	Name   string `json:"n,omitempty"`
	Issued string `json:"d,omitempty"` // YYYY-MM-DD
}

var errBadKey = errors.New("that key isn't valid - check you copied all of it")

func encodeKey(l License, priv ed25519.PrivateKey) string {
	payload, _ := json.Marshal(l)
	sig := ed25519.Sign(priv, append([]byte(licenseDomain), payload...))
	return "EE-" + base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func parseKey(key string, pub ed25519.PublicKey) (License, error) {
	key = strings.Join(strings.Fields(key), "") // tolerate line breaks from chat apps
	if pub == nil {
		return License{}, errors.New("keys aren't enabled in this build")
	}
	body, ok := strings.CutPrefix(key, "EE-")
	if !ok {
		return License{}, errBadKey
	}
	p64, s64, ok := strings.Cut(body, ".")
	if !ok {
		return License{}, errBadKey
	}
	payload, err1 := base64.RawURLEncoding.DecodeString(p64)
	sig, err2 := base64.RawURLEncoding.DecodeString(s64)
	if err1 != nil || err2 != nil || len(sig) != ed25519.SignatureSize {
		return License{}, errBadKey
	}
	if !ed25519.Verify(pub, append([]byte(licenseDomain), payload...), sig) {
		return License{}, errBadKey
	}
	var l License
	if err := json.Unmarshal(payload, &l); err != nil || l.ID == "" {
		return License{}, errBadKey
	}
	if _, ok := tiers[l.Tier]; !ok {
		return License{}, errors.New("this key is for a newer version - update the app first")
	}
	return l, nil
}

// ------------------------------------------------------------ keyring

type Keyring struct {
	mu      sync.Mutex
	pub     ed25519.PublicKey
	path    string
	lic     *License
	revoked map[string]bool
}

func NewKeyring(pub ed25519.PublicKey) *Keyring {
	k := &Keyring{pub: pub, path: filepath.Join(store.Dir(), "key.txt"), revoked: map[string]bool{}}
	if raw, err := os.ReadFile(k.path); err == nil {
		if l, err := parseKey(string(raw), pub); err == nil {
			k.lic = &l
		}
	}
	return k
}

func (k *Keyring) License() *License {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.lic == nil || k.revoked[k.lic.ID] {
		return nil
	}
	c := *k.lic
	return &c
}

func (k *Keyring) Activate(key string) (License, error) {
	l, err := parseKey(key, k.pub)
	if err != nil {
		return License{}, err
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.revoked[l.ID] {
		return License{}, errors.New("this key has been turned off")
	}
	_ = os.MkdirAll(filepath.Dir(k.path), 0o755)
	if err := os.WriteFile(k.path, []byte(strings.TrimSpace(key)), 0o600); err != nil {
		return License{}, err
	}
	k.lic = &l
	return l, nil
}

func (k *Keyring) Remove() {
	k.mu.Lock()
	defer k.mu.Unlock()
	_ = os.Remove(k.path)
	k.lic = nil
}

// LoadRevoked fetches revoked.txt (one key ID per line, # comments) from the
// public repo. Returns true if the active key was revoked.
func (k *Keyring) LoadRevoked(url string, client *http.Client) bool {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "ElectricEditor")
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return false
	}
	ids := map[string]bool{}
	sc := bufio.NewScanner(io.LimitReader(resp.Body, 1<<20))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			ids[line] = true
		}
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	k.revoked = ids
	return k.lic != nil && ids[k.lic.ID]
}

func revokedURL() string {
	return "https://raw.githubusercontent.com/" + updateRepo + "/main/revoked.txt"
}

var httpClient = &http.Client{Timeout: 20 * time.Second}
