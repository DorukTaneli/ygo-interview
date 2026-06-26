package main

import (
	"context"
	"fmt"
	"strings"
)

// factLines renders facts as a numbered, ID-tagged list for prompts.
func factLines(facts []Fact) string {
	var sb strings.Builder
	for _, f := range facts {
		fmt.Fprintf(&sb, "- [%s] %s\n", f.ID, f.Text)
	}
	return sb.String()
}

// hotelContext is the framing (name, location, setting, tier) shared by every
// language so they describe the same property.
func hotelContext(h Hotel) string {
	return fmt.Sprintf("Hotel: %s\nLocation: %s, %s\nSetting: %s\nPrice band: %s",
		h.Name, h.City, h.Country, h.Setting, h.PriceBand)
}

const genSystem = `You are a travel copywriter producing short marketing descriptions for hotels.
Write naturally and idiomatically in the target language — as a native speaker would, not as a literal translation.
You must feature EXACTLY the facts provided, no more and no fewer:
- Do not omit any provided fact.
- Do not invent amenities, numbers, distances, or policies that are not in the provided facts.
- You may rephrase facts attractively, but every fact must remain clearly stated.
Write 2-4 sentences. Output only the description itself, with no preamble, title, or quotation marks.`

// generateDescription produces a native-sounding description for one language
// that features exactly the given facts.
func generateDescription(ctx context.Context, c Client, h Hotel, facts []Fact, language string) (string, error) {
	user := fmt.Sprintf(`%s

Write the description in %s.

Feature exactly these facts:
%s`, hotelContext(h), language, factLines(facts))
	return c.Complete(ctx, genSystem, user, 0.7, 1024)
}
