package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Self-updater backed by GitHub Releases.
//
// Each release carries one exe per edition plus "<exe>.sig": an ed25519
// signature made with a private key only the maintainer has (tools/signer).
// The app refuses any download whose signature doesn't verify against the
// public key baked in at build time, so a compromised GitHub account or a
// tampered download can't push code to users.

type UpdateStatus struct {
	State   string `json:"state"` // disabled | checking | current | ready | error
	Version string `json:"version"`
	Message string `json:"message"`
}

type ghAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

type ghRelease struct {
	Tag        string    `json:"tag_name"`
	Draft      bool      `json:"draft"`
	Prerelease bool      `json:"prerelease"`
	Assets     []ghAsset `json:"assets"`
}

type Updater struct {
	version string
	channel func() string // stable | beta, from the active key
	apiBase string
	client  *http.Client
	pub     ed25519.PublicKey
	exePath func() (string, error)
	status  UpdateStatus
}

func NewUpdater(channel func() string) *Updater {
	return &Updater{
		version: version,
		channel: channel,
		apiBase: "https://api.github.com/repos/" + updateRepo,
		client:  &http.Client{Timeout: 60 * time.Second},
		exePath: os.Executable,
		pub:     pubKey(),
	}
}

// CleanupOld removes the previous exe left behind by the last update.
func (u *Updater) CleanupOld() {
	if exe, err := u.exePath(); err == nil {
		_ = os.Remove(exe + ".old")
	}
}

func (u *Updater) Run(ctx context.Context) {
	u.status = u.check()
	runtime.EventsEmit(ctx, "update", u.status)
}

func (u *Updater) Status() UpdateStatus { return u.status }

func (u *Updater) check() UpdateStatus {
	if u.version == "dev" || u.pub == nil {
		return UpdateStatus{State: "disabled", Version: u.version}
	}
	rel, asset, sig, err := u.findUpdate()
	if err != nil {
		return UpdateStatus{State: "error", Version: u.version, Message: "Couldn't check for updates: " + err.Error()}
	}
	if rel == nil {
		return UpdateStatus{State: "current", Version: u.version}
	}
	data, err := u.download(asset.URL, 150<<20)
	if err != nil {
		return UpdateStatus{State: "error", Version: u.version, Message: "Update download failed: " + err.Error()}
	}
	sigRaw, err := u.download(sig.URL, 4096)
	if err != nil {
		return UpdateStatus{State: "error", Version: u.version, Message: "Update signature missing"}
	}
	if !verify(u.pub, data, sigRaw) {
		return UpdateStatus{State: "error", Version: u.version, Message: "Update rejected: signature didn't match. Download it from GitHub instead."}
	}
	exe, err := u.exePath()
	if err != nil {
		return UpdateStatus{State: "error", Version: u.version, Message: err.Error()}
	}
	if err := applyUpdate(exe, data); err != nil {
		return UpdateStatus{State: "error", Version: u.version, Message: "Couldn't install the update: " + err.Error()}
	}
	return UpdateStatus{State: "ready", Version: strings.TrimPrefix(rel.Tag, "v")}
}

func (u *Updater) findUpdate() (*ghRelease, *ghAsset, *ghAsset, error) {
	req, _ := http.NewRequest("GET", u.apiBase+"/releases?per_page=30", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ElectricEditor-updater")
	resp, err := u.client.Do(req)
	if err != nil {
		return nil, nil, nil, errors.New("no connection")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, nil, nil, fmt.Errorf("GitHub said %s", resp.Status)
	}
	var rels []ghRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&rels); err != nil {
		return nil, nil, nil, errors.New("unexpected reply from GitHub")
	}
	r, a, s := pickRelease(rels, u.version, u.channel())
	return r, a, s, nil
}

// pickRelease returns the newest release newer than the running version that
// has the exe and its signature, respecting the update channel.
func pickRelease(rels []ghRelease, current, channel string) (*ghRelease, *ghAsset, *ghAsset) {
	cur, ok := parseVersion(current)
	if !ok {
		return nil, nil, nil
	}
	var best *ghRelease
	var bestV semver
	var bestA, bestS *ghAsset
	for i := range rels {
		r := &rels[i]
		if r.Draft || (r.Prerelease && channel != "beta") {
			continue
		}
		v, ok := parseVersion(r.Tag)
		if !ok || !v.newer(cur) || (best != nil && !v.newer(bestV)) {
			continue
		}
		var a, s *ghAsset
		for j := range r.Assets {
			switch r.Assets[j].Name {
			case releaseAsset:
				a = &r.Assets[j]
			case releaseAsset + ".sig":
				s = &r.Assets[j]
			}
		}
		if a != nil && s != nil {
			best, bestV, bestA, bestS = r, v, a, s
		}
	}
	return best, bestA, bestS
}

func (u *Updater) download(url string, limit int64) ([]byte, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "ElectricEditor-updater")
	resp, err := u.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s", resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, errors.New("file too large")
	}
	return b, nil
}

func verify(pub ed25519.PublicKey, data, sigFile []byte) bool {
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sigFile)))
	if err != nil || len(sig) != ed25519.SignatureSize {
		return false
	}
	return ed25519.Verify(pub, data, sig)
}

// applyUpdate swaps the running exe for the new one. Windows lets a running
// exe be renamed but not overwritten, so: exe -> exe.old, new -> exe.
func applyUpdate(exe string, data []byte) error {
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	tmp := exe + ".new"
	if err := os.WriteFile(tmp, data, 0o755); err != nil {
		return err
	}
	old := exe + ".old"
	_ = os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, exe); err != nil {
		_ = os.Rename(old, exe) // put the working version back
		os.Remove(tmp)
		return err
	}
	return nil
}

// Restart launches the (updated) exe and quits this one.
func (u *Updater) Restart(ctx context.Context) error {
	exe, err := u.exePath()
	if err != nil {
		return err
	}
	if err := exec.Command(exe).Start(); err != nil {
		return err
	}
	runtime.Quit(ctx)
	return nil
}

// ------------------------------------------------------------ versions

type semver struct {
	maj, min, pat int
	pre           string // "" sorts after any pre-release
}

func parseVersion(s string) (semver, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	core, pre, _ := strings.Cut(s, "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return semver{}, false
	}
	var n [3]int
	for i, p := range parts {
		v, err := strconv.Atoi(p)
		if err != nil || v < 0 {
			return semver{}, false
		}
		n[i] = v
	}
	return semver{n[0], n[1], n[2], pre}, true
}

func (a semver) newer(b semver) bool {
	if a.maj != b.maj {
		return a.maj > b.maj
	}
	if a.min != b.min {
		return a.min > b.min
	}
	if a.pat != b.pat {
		return a.pat > b.pat
	}
	switch {
	case a.pre == b.pre:
		return false
	case a.pre == "":
		return true
	case b.pre == "":
		return false
	default:
		return a.pre > b.pre
	}
}
