package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
)

// atomicFactsJSON is the frozen, committed reference fact set. It is generated
// once by the LLM atomizer (go run . -atomize) and checked into the repo, so
// every machine scores against the identical set of atomic facts rather than
// re-deriving them — the canonical reference must not wobble between runs.
//
//go:embed atomic_facts.json
var atomicFactsJSON []byte

// loadAtomicFacts parses the frozen reference into a hotel-ID -> facts map.
func loadAtomicFacts() (map[string][]Fact, error) {
	var m map[string][]Fact
	if err := json.Unmarshal(atomicFactsJSON, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// regenerateAtomicFacts runs the LLM atomizer over every source hotel and writes
// the result to path. This is a deliberate prep step (go run . -atomize), not
// part of the scored pipeline: run it once, then commit the file.
func regenerateAtomicFacts(ctx context.Context, c Client, hotels []Hotel, path string) error {
	out := map[string][]Fact{}
	for _, h := range hotels {
		facts, err := atomizeFacts(ctx, c, h)
		if err != nil {
			return fmt.Errorf("atomizing %s: %w", h.ID, err)
		}
		out[h.ID] = facts
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}
