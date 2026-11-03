package main

// Set at build time with -ldflags "-X main.version=1.2.0 -X main.edition=amy ..."
// (see build.ps1). Defaults are for local dev builds.
var (
	version      = "dev"
	edition      = "standard"
	updateRepo   = "Jamie-TFS/electric-editor"
	updatePubKey = "" // base64 ed25519 public key; empty = updates disabled
)

// Edition is one flavour of the app built from the same code.
type Edition struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Theme    string `json:"theme"`
	Asset    string `json:"asset"`   // release file this edition updates from
	Channel  string `json:"channel"` // stable | beta (beta also takes pre-releases)
	DevTools bool   `json:"devTools"`
	Greeting string `json:"greeting"`
	Version  string `json:"version"`
}

var editions = map[string]Edition{
	"standard": {Title: "Electric Editor", Theme: "storm", Asset: "ElectricEditor.exe", Channel: "stable"},
	"dev":      {Title: "Electric Editor dev", Theme: "storm", Asset: "ElectricEditor-dev.exe", Channel: "beta", DevTools: true},
	"amy":      {Title: "Amy's Electric Editor", Theme: "amy", Asset: "ElectricEditor-amy.exe", Channel: "stable", Greeting: "Hi Amy"},
}

func currentEdition() Edition {
	e, ok := editions[edition]
	if !ok {
		e = editions["standard"]
		edition = "standard"
	}
	e.ID = edition
	e.Version = version
	return e
}
