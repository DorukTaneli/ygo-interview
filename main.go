package main

import (
	"context"
	"fmt"
	"os"
)

// langOutput bundles everything produced for one language.
type langOutput struct {
	Language string
	Text     string
	Verify   VerifyResult
	Native   NativenessResult
}

// genFunc is a generation strategy: naiveGenerate or generateDescription.
type genFunc func(ctx context.Context, c Client, h Hotel, facts []Fact, language string) (string, error)

func main() {
	loadDotEnv(".env")
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		fmt.Fprintln(os.Stderr, "ANTHROPIC_API_KEY not set (put it in .env or the environment)")
		os.Exit(1)
	}

	ctx := context.Background()
	c := NewClient()

	hotels, err := LoadHotels()
	if err != nil {
		fmt.Fprintln(os.Stderr, "loading hotels:", err)
		os.Exit(1)
	}

	h := hotels[0]
	languages := []string{"English", "German", "French"}

	// The canonical reference: source facts split into atomic claims (cached).
	facts, err := atomizeFacts(ctx, c, h)
	if err != nil {
		fmt.Fprintln(os.Stderr, "atomizing facts:", err)
		os.Exit(1)
	}

	fmt.Printf("=== %s (%s, %s) — %d atomic facts ===\n", h.Name, h.City, h.Country, len(facts))
	for _, f := range facts {
		fmt.Printf("  [%s] %s\n", f.ID, f.Text)
	}

	// Two strategies, same eval harness, so the scores are directly comparable.
	naiveOut, naiveRep, err := runMode(ctx, c, h, facts, languages, naiveGenerate)
	if err != nil {
		fmt.Fprintln(os.Stderr, "naive mode:", err)
		os.Exit(1)
	}
	structOut, structRep, err := runMode(ctx, c, h, facts, languages, generateDescription)
	if err != nil {
		fmt.Fprintln(os.Stderr, "structured mode:", err)
		os.Exit(1)
	}

	printMode("NAIVE BASELINE — whole JSON, each language written independently", naiveOut, naiveRep, len(facts))
	printMode("STRUCTURED — facts pinned, same set for every language", structOut, structRep, len(facts))
	printSummary(naiveOut, naiveRep, structOut, structRep)
}

// runMode generates, verifies and rates every language with the given strategy,
// then derives the cross-language consistency report.
func runMode(ctx context.Context, c Client, h Hotel, facts []Fact, languages []string, gen genFunc) ([]langOutput, ConsistencyReport, error) {
	var outputs []langOutput
	results := map[string]VerifyResult{}
	for _, lang := range languages {
		text, err := gen(ctx, c, h, facts, lang)
		if err != nil {
			return nil, ConsistencyReport{}, fmt.Errorf("generate %s: %w", lang, err)
		}
		vr, err := verify(ctx, c, facts, text)
		if err != nil {
			return nil, ConsistencyReport{}, fmt.Errorf("verify %s: %w", lang, err)
		}
		nr, err := nativeness(ctx, c, text, lang)
		if err != nil {
			return nil, ConsistencyReport{}, fmt.Errorf("nativeness %s: %w", lang, err)
		}
		outputs = append(outputs, langOutput{lang, text, vr, nr})
		results[lang] = vr
	}
	return outputs, crossLanguageConsistency(languages[0], results), nil
}

func printMode(header string, outputs []langOutput, rep ConsistencyReport, total int) {
	fmt.Printf("\n########## %s ##########\n", header)
	for _, o := range outputs {
		fmt.Printf("\n--- %s ---\n%s\n", o.Language, o.Text)
		fmt.Printf("facts carried: %d/%d", len(o.Verify.PresentIDs), total)
		if len(o.Verify.MissingIDs) > 0 {
			fmt.Printf("  | missing: %v", o.Verify.MissingIDs)
		}
		if len(o.Verify.UnsupportedClaims) > 0 {
			fmt.Printf("  | UNSUPPORTED: %v", o.Verify.UnsupportedClaims)
		}
		fmt.Printf("\nnativeness: %d/5 (%s)\n", o.Native.Score, o.Native.Reason)
	}

	// Iterate outputs (not the map) so language order is stable.
	fmt.Printf("\nCross-language consistency (%s as reference): %.0f%%\n", rep.Base, rep.Agreement*100)
	for _, o := range outputs {
		if o.Language == rep.Base {
			continue
		}
		d := rep.PerLang[o.Language]
		fmt.Printf("  %-8s agreement %.0f%%", o.Language+":", d.Agreement*100)
		if len(d.Dropped) > 0 {
			fmt.Printf(" | dropped vs base: %v", d.Dropped)
		}
		if len(d.Added) > 0 {
			fmt.Printf(" | added vs base: %v", d.Added)
		}
		fmt.Println()
	}
	fmt.Printf("Ungrounded claims (inventions) across all languages: %d\n", rep.Inventions)
}

func printSummary(naiveOut []langOutput, naiveRep ConsistencyReport, structOut []langOutput, structRep ConsistencyReport) {
	fmt.Printf("\n########## SUMMARY: naive vs structured ##########\n")
	row := func(metric, naive, structured string) {
		fmt.Printf("%-28s %-10s %-10s\n", metric, naive, structured)
	}
	row("metric", "naive", "structured")
	row("cross-language consistency", pct(naiveRep.Agreement), pct(structRep.Agreement))
	row("ungrounded inventions", fmt.Sprint(naiveRep.Inventions), fmt.Sprint(structRep.Inventions))
	row("avg nativeness (1-5)", fmt.Sprintf("%.1f", avgNative(naiveOut)), fmt.Sprintf("%.1f", avgNative(structOut)))
}

func pct(f float64) string { return fmt.Sprintf("%.0f%%", f*100) }

func avgNative(outputs []langOutput) float64 {
	if len(outputs) == 0 {
		return 0
	}
	var s int
	for _, o := range outputs {
		s += o.Native.Score
	}
	return float64(s) / float64(len(outputs))
}
