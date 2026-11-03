package fx

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

// Checks catalog paths against a real curve list from the game, when present.
func TestPathsAgainstGameDump(t *testing.T) {
	f, err := os.Open(os.Getenv("EE_CURVES"))
	if err != nil {
		t.Skip("set EE_CURVES to a curves.txt to run")
	}
	defer f.Close()
	have := map[string]bool{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if line := sc.Text(); strings.HasPrefix(line, "/") {
			have[strings.SplitN(line, " | ", 2)[0]] = true
		}
	}
	for _, c := range Catalog {
		if c.ID != "kazuya" && c.ID != "deviljin" {
			continue // the dump was a Kazuya vs Devil Jin match
		}
		for _, e := range c.Effects {
			for _, p := range e.Parts {
				for _, path := range p.Paths {
					if !have[path] {
						t.Errorf("%s.%s.%s: not in game dump: %s", c.ID, e.ID, p.ID, path)
					}
				}
			}
		}
	}
}
