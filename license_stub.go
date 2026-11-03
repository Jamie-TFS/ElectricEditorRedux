//go:build !keys

package main

// Open-source stand-in for the private key system (license_keys.go, built with
// -tags keys for official releases). Without it the app works fully for
// regular users; keys just aren't available.

import (
	"crypto/ed25519"
	"errors"
	"net/http"
	"time"
)

const keysAvailable = false

type License struct {
	ID     string `json:"i"`
	Tier   string `json:"t"`
	Name   string `json:"n,omitempty"`
	Issued string `json:"d,omitempty"`
}

type Keyring struct{}

func NewKeyring(ed25519.PublicKey) *Keyring { return &Keyring{} }

func (k *Keyring) License() *License { return nil }

func (k *Keyring) Activate(string) (License, error) {
	return License{}, errors.New("keys aren't available in this build - download the official release")
}

func (k *Keyring) Remove() {}

func (k *Keyring) LoadRevoked(string, *http.Client) bool { return false }

func revokedURL() string { return "" }

var httpClient = &http.Client{Timeout: 20 * time.Second}
