// Writes the fx catalog as JSON for the frontend's browser preview mode.
package main

import (
	"encoding/json"
	"os"

	"app/internal/fx"
)

func main() {
	b, _ := json.MarshalIndent(fx.Catalog, "", "  ")
	os.WriteFile(os.Args[1], b, 0o644)
}
