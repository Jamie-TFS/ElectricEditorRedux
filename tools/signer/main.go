// signer creates the update signing key and signs release files.
//
//	go run ./tools/signer keygen            -> writes signing.key (KEEP SECRET), prints the public key
//	go run ./tools/signer sign file.exe     -> writes file.exe.sig next to it
//	go run ./tools/signer pubkey            -> prints the public key again (for build.ps1)
//	go run ./tools/signer key amy Amy       -> prints a key for tier "amy" named "Amy"
//	                                           (tiers: dev, amy, donator) and logs it to issued-keys.txt
//
// Never commit signing.key. Anyone holding it can push updates to every user.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
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
	case "pubkey":
		fmt.Println(base64.StdEncoding.EncodeToString(loadKey().Public().(ed25519.PublicKey)))
	case "key":
		if len(os.Args) < 3 {
			usage()
		}
		priv := loadKey()
		tier := os.Args[2]
		name := strings.Join(os.Args[3:], " ")
		id := make([]byte, 6)
		rand.Read(id)
		lic := map[string]string{"i": hex.EncodeToString(id), "t": tier, "d": time.Now().Format("2006-01-02")}
		if name != "" {
			lic["n"] = name
		}
		payload, _ := json.Marshal(lic)
		sig := ed25519.Sign(priv, append([]byte("EE-LICENSE-v1:"), payload...))
		key := "EE-" + base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(sig)
		f, err := os.OpenFile("issued-keys.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err == nil {
			fmt.Fprintf(f, "%s\t%s\t%s\t%s\n", lic["i"], lic["d"], tier, name)
			f.Close()
		}
		fmt.Println(key)
		fmt.Fprintf(os.Stderr, "\nID %s (to revoke it, add this ID to revoked.txt in the repo)\n", lic["i"])
	case "sign":
		if len(os.Args) < 3 {
			usage()
		}
		k := loadKey()
		for _, f := range os.Args[2:] {
			data, err := os.ReadFile(f)
			if err != nil {
				die(err.Error())
			}
			sig := base64.StdEncoding.EncodeToString(ed25519.Sign(k, data))
			if err := os.WriteFile(f+".sig", []byte(sig), 0o644); err != nil {
				die(err.Error())
			}
			fmt.Println("signed", f)
		}
	default:
		usage()
	}
}

func loadKey() ed25519.PrivateKey {
	raw, err := os.ReadFile(keyFile)
	if err != nil {
		die("signing.key not found - run keygen first (in the app folder)")
	}
	k, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(k) != ed25519.PrivateKeySize {
		die("signing.key is damaged")
	}
	return ed25519.PrivateKey(k)
}

func usage() {
	die("usage: signer keygen | signer pubkey | signer sign <file>... | signer key <tier> [name]")
}

func die(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
