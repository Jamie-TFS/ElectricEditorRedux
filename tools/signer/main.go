// signer creates the update signing key and signs release files.
//
//	go run ./tools/signer keygen            -> writes signing.key (KEEP SECRET), prints the public key
//	go run ./tools/signer sign file.exe     -> writes file.exe.sig next to it
//
// Never commit signing.key. Anyone holding it can push updates to every user.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

const keyFile = "signing.key"

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "keygen":
		if _, err := os.Stat(keyFile); err == nil {
			die("signing.key already exists - refusing to overwrite it (old releases would stop verifying)")
		}
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			die(err.Error())
		}
		if err := os.WriteFile(keyFile, []byte(base64.StdEncoding.EncodeToString(priv)), 0o600); err != nil {
			die(err.Error())
		}
		fmt.Println("Wrote", keyFile, "- back it up somewhere safe and never commit it.")
		fmt.Println("Public key (put this in build.ps1 as $PubKey):")
		fmt.Println(base64.StdEncoding.EncodeToString(pub))
	case "sign":
		if len(os.Args) < 3 {
			usage()
		}
		raw, err := os.ReadFile(keyFile)
		if err != nil {
			die("signing.key not found - run keygen first")
		}
		k, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
		if err != nil || len(k) != ed25519.PrivateKeySize {
			die("signing.key is damaged")
		}
		for _, f := range os.Args[2:] {
			data, err := os.ReadFile(f)
			if err != nil {
				die(err.Error())
			}
			sig := base64.StdEncoding.EncodeToString(ed25519.Sign(ed25519.PrivateKey(k), data))
			if err := os.WriteFile(f+".sig", []byte(sig), 0o644); err != nil {
				die(err.Error())
			}
			fmt.Println("signed", f)
		}
	default:
		usage()
	}
}

func usage() { die("usage: signer keygen | signer sign <file>...") }

func die(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
