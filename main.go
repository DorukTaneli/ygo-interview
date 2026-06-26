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

	// Thin slice: first hotel, every fact featured. The base language (English)
	// commits to a fact set; German and French must carry the same set.
	h := hotels[0]
	facts := h.Facts()
	languages := []string{"English", "German", "French"}

	fmt.Printf("=== %s (%s, %s) — %d facts featured ===\n\n", h.Name, h.City, h.Country, len(facts))

	for _, lang := range languages {
		text, err := generateDescription(ctx, c, h, facts, lang)
		if err != nil {
			fmt.Fprintf(os.Stderr, "generate %s: %v\n", lang, err)
			os.Exit(1)
		}
		vr, err := verify(ctx, c, facts, text)
		if err != nil {
			fmt.Fprintf(os.Stderr, "verify %s: %v\n", lang, err)
			os.Exit(1)
		}
		nr, err := nativeness(ctx, c, text, lang)
		if err != nil {
			fmt.Fprintf(os.Stderr, "nativeness %s: %v\n", lang, err)
			os.Exit(1)
		}
		printResult(langOutput{lang, text, vr, nr}, len(facts))
	}
}

func printResult(o langOutput, total int) {
	fmt.Printf("--- %s ---\n", o.Language)
	fmt.Println(o.Text)
	fmt.Printf("\nfacts carried: %d/%d\n", len(o.Verify.PresentIDs), total)
	if len(o.Verify.MissingIDs) > 0 {
		fmt.Printf("  MISSING:     %v\n", o.Verify.MissingIDs)
	}
	if len(o.Verify.UnsupportedClaims) > 0 {
		fmt.Printf("  UNSUPPORTED: %v\n", o.Verify.UnsupportedClaims)
	}
	fmt.Printf("nativeness:    %d/5 (%s)\n\n", o.Native.Score, o.Native.Reason)
}
