package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestVersionOrder(t *testing.T) {
	cases := []struct {
		a, b  string
		newer bool
	}{
		{"1.2.0", "1.1.9", true},
		{"v1.10.0", "1.9.0", true},
		{"1.2.0", "1.2.0", false},
		{"1.2.0", "1.2.0-beta.1", true},
		{"1.2.0-beta.2", "1.2.0-beta.1", true},
		{"1.2.0-beta.1", "1.2.0", false},
	}
	for _, c := range cases {
		a, _ := parseVersion(c.a)
		b, _ := parseVersion(c.b)
		if a.newer(b) != c.newer {
			t.Errorf("%s newer than %s = %v, want %v", c.a, c.b, !c.newer, c.newer)
		}
	}
	if _, ok := parseVersion("dev"); ok {
		t.Error("dev should not parse")
	}
}

func rel(tag string, pre bool, assets ...string) ghRelease {
	r := ghRelease{Tag: tag, Prerelease: pre}
	for _, a := range assets {
		r.Assets = append(r.Assets, ghAsset{Name: a, URL: "https://x/" + tag + "/" + a})
	}
	return r
}

func TestPickRelease(t *testing.T) {
	rels := []ghRelease{
		rel("v1.3.0-beta.1", true, "ElectricEditor.exe", "ElectricEditor.exe.sig", "ElectricEditor-dev.exe", "ElectricEditor-dev.exe.sig"),
		rel("v1.2.0", false, "ElectricEditor.exe", "ElectricEditor.exe.sig", "ElectricEditor-amy.exe", "ElectricEditor-amy.exe.sig"),
		rel("v1.1.5", false, "ElectricEditor.exe", "ElectricEditor.exe.sig"),
		rel("v1.4.0", false, "ElectricEditor.exe"), // unsigned: must be ignored
	}
	std := editions["standard"]
	std.Version = "1.1.0"
	if r, _, _ := pickRelease(rels, std); r == nil || r.Tag != "v1.2.0" {
		t.Fatalf("stable should get v1.2.0, got %+v", r)
	}
	dev := editions["dev"]
	dev.Version = "1.1.0"
	if r, _, _ := pickRelease(rels, dev); r == nil || r.Tag != "v1.3.0-beta.1" {
		t.Fatalf("dev should get the beta, got %+v", r)
	}
	amy := editions["amy"]
	amy.Version = "1.2.0"
	if r, _, _ := pickRelease(rels, amy); r != nil {
		t.Fatalf("amy is current, got %+v", r)
	}
	amy.Version = "1.0.0"
	if _, a, _ := pickRelease(rels, amy); a == nil || a.Name != "ElectricEditor-amy.exe" {
		t.Fatal("amy must update from her own file")
	}
}

func TestVerifyAndApply(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	data := []byte("new exe bytes")
	sig := []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(priv, data)))
	if !verify(pub, data, sig) {
		t.Fatal("good signature rejected")
	}
	if verify(pub, []byte("tampered"), sig) {
		t.Fatal("tampered file accepted")
	}
	otherPub, _, _ := ed25519.GenerateKey(nil)
	if verify(otherPub, data, sig) {
		t.Fatal("wrong key accepted")
	}

	dir := t.TempDir()
	exe := filepath.Join(dir, "ElectricEditor.exe")
	os.WriteFile(exe, []byte("old exe"), 0o755)
	if err := applyUpdate(exe, data); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(exe)
	old, _ := os.ReadFile(exe + ".old")
	if string(got) != "new exe bytes" || string(old) != "old exe" {
		t.Fatalf("swap wrong: exe=%q old=%q", got, old)
	}
}

func TestUpdaterEndToEnd(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	newExe := []byte("v1.2.0 exe")
	goodSig := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, newExe))

	run := func(sig string) (UpdateStatus, string) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			base := "http://" + r.Host
			switch r.URL.Path {
			case "/releases":
				fmt.Fprintf(w, `[{"tag_name":"v1.2.0","assets":[
					{"name":"ElectricEditor.exe","browser_download_url":"%s/dl/exe"},
					{"name":"ElectricEditor.exe.sig","browser_download_url":"%s/dl/sig"}]}]`, base, base)
			case "/dl/exe":
				w.Write(newExe)
			case "/dl/sig":
				w.Write([]byte(sig))
			}
		}))
		defer srv.Close()
		exe := filepath.Join(t.TempDir(), "ElectricEditor.exe")
		os.WriteFile(exe, []byte("v1.0.0 exe"), 0o755)
		ed := editions["standard"]
		ed.Version = "1.0.0"
		u := &Updater{ed: ed, apiBase: srv.URL, client: srv.Client(), pub: pub,
			exePath: func() (string, error) { return exe, nil }}
		st := u.check()
		got, _ := os.ReadFile(exe)
		return st, string(got)
	}

	if st, got := run(goodSig); st.State != "ready" || st.Version != "1.2.0" || got != "v1.2.0 exe" {
		t.Fatalf("good update: %+v exe=%q", st, got)
	}
	badSig := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, []byte("something else")))
	if st, got := run(badSig); st.State != "error" || got != "v1.0.0 exe" {
		t.Fatalf("bad signature must not install: %+v exe=%q", st, got)
	}
}
