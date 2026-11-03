package main

import (
	"crypto/ed25519"
	"encoding/base64"
)

// Set at build time by build.ps1:
//
//	-X main.version=1.2.0 -X main.updateRepo=owner/repo -X main.publicKey=<base64>
var (
	version    = "dev"
	updateRepo = "Jamie-TFS/ElectricEditor"
	publicKey  = "" // base64 ed25519 public key; verifies updates AND keys. Empty = both off.
)

const releaseAsset = "ElectricEditor.exe"

func pubKey() ed25519.PublicKey {
	k, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil || len(k) != ed25519.PublicKeySize {
		return nil
	}
	return k
}

// Tier is what a key unlocks. Add new tiers here; old app versions will tell
// people with a new-tier key to update.
type Tier struct {
	Title    string
	Theme    string
	Channel  string // stable | beta
	DevTools bool
	Badge    string
	Greet    bool // show "Hi <name>"
}

var tiers = map[string]Tier{
	"":        {Title: "Electric Editor", Theme: "storm", Channel: "stable"},
	"donator": {Title: "Electric Editor", Theme: "storm", Channel: "stable", Badge: "Supporter"},
	"dev":     {Title: "Electric Editor dev", Theme: "storm", Channel: "beta", DevTools: true, Badge: "Dev"},
	"amy":     {Title: "Amy's Electric Editor", Theme: "amy", Channel: "stable", Greet: true},
}

// Edition is what the UI needs to know about the running app.
type Edition struct {
	Tier     string `json:"tier"`
	Title    string `json:"title"`
	Theme    string `json:"theme"`
	Channel  string `json:"channel"`
	DevTools bool   `json:"devTools"`
	Badge    string `json:"badge"`
	Greeting string `json:"greeting"`
	KeyName  string `json:"keyName"`
	KeysOn   bool   `json:"keysOn"`
	Version  string `json:"version"`
}

func editionFor(l *License) Edition {
	id := ""
	name := ""
	if l != nil {
		id, name = l.Tier, l.Name
	}
	t := tiers[id]
	e := Edition{Tier: id, Title: t.Title, Theme: t.Theme, Channel: t.Channel, DevTools: t.DevTools,
		Badge: t.Badge, KeyName: name, KeysOn: keysAvailable && pubKey() != nil, Version: version}
	if t.Greet && name != "" {
		e.Greeting = "Hi " + name
	}
	return e
}
