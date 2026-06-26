package main

import (
	"context"
	"fmt"
)

// AtomicClaim is one atomic fact emitted by the atomizer, tagged with the
// source category it came from.
type AtomicClaim struct {
	Category string `json:"category"`
	Text     string `json:"text"`
}

type atomizerOutput struct {
	Facts []AtomicClaim `json:"facts"`
}

const atomizeSystem = `You break hotel source facts into atomic claims for fact-checking.
You are given source fact lines, each prefixed with a category tag like [amenity-1] (the category is the word before the dash: amenity, room, nearby, or policy).
Split each line into the smallest distinct factual claims a traveller would verify separately, and keep each claim's category.

Rules:
- A line that bundles SEVERAL distinct things becomes several claims. Example: "spa with sauna (adults only)" -> "spa", "sauna", "spa is adults only". "indoor pool, sauna" -> "indoor pool", "sauna".
- A single thing with its qualifiers stays as ONE claim. Example: "family apartments for up to 4" stays "family apartments for up to 4"; "heated outdoor pool" stays one claim; "boot room with heated racks" stays one claim.
- Do NOT add any information not present in the source. Do NOT drop any information: every distinct fact in the source must survive in some claim.
- Keep each claim short and self-contained, in English.`

// atomizeFacts splits the source facts into atomic claims via the LLM. The call
// runs at temperature 0 and is cached, so the resulting reference fact set is
// stable for a given hotel — grounded in the source, just at finer granularity
// than the raw bundled lines.
func atomizeFacts(ctx context.Context, c Client, h Hotel) ([]Fact, error) {
	var out atomizerOutput
	user := "Source fact lines:\n" + factLines(h.Facts())
	if err := c.CompleteSchema(ctx, strongModel, atomizeSystem, user, 0.0, 2048, effortLow, &out); err != nil {
		return nil, fmt.Errorf("atomize: %w", err)
	}

	counts := map[string]int{}
	var facts []Fact
	for _, claim := range out.Facts {
		cat := claim.Category
		if cat == "" {
			cat = "fact"
		}
		facts = append(facts, Fact{ID: fmt.Sprintf("%s-%d", cat, counts[cat]), Text: claim.Text})
		counts[cat]++
	}
	return facts, nil
}
