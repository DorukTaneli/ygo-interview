package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

// langOutput bundles everything produced for one language.
type langOutput struct {
	Language string
	Text     string
	Verify   VerifyResult
	Native   NativenessResult
}

// runResult is one strategy's output and scores for a single run.
type runResult struct {
	outputs []langOutput
	report  ConsistencyReport
}

// genFunc is a generation strategy: naiveGenerate or generateDescription.
type genFunc func(ctx context.Context, c Client, h Hotel, facts []Fact, language string) (string, error)

func main() {
	atomize := flag.Bool("atomize", false, "regenerate atomic_facts.json from source via the LLM, then exit")
	runs := flag.Int("runs", 5, "number of fresh generation+eval runs per strategy")
	flag.Parse()

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

	// Prep step: regenerate the frozen reference and exit. Run deliberately, then
	// commit atomic_facts.json. The scored pipeline below never calls the atomizer.
	if *atomize {
		if err := regenerateAtomicFacts(ctx, c, hotels, "atomic_facts.json"); err != nil {
			fmt.Fprintln(os.Stderr, "regenerating atomic facts:", err)
			os.Exit(1)
		}
		fmt.Println("wrote atomic_facts.json")
		return
	}

	factsByHotel, err := loadAtomicFacts()
	if err != nil {
		fmt.Fprintln(os.Stderr, "loading atomic facts:", err)
		os.Exit(1)
	}

	h := hotels[0]
	languages := []string{"English", "German", "French"}
	facts := factsByHotel[h.ID]
	if len(facts) == 0 {
		fmt.Fprintf(os.Stderr, "no atomic facts for %s — run `go run . -atomize` first\n", h.ID)
		os.Exit(1)
	}

	fmt.Printf("=== %s (%s, %s) — %d atomic facts ===\n", h.Name, h.City, h.Country, len(facts))
	for _, f := range facts {
		fmt.Printf("  [%s] %s\n", f.ID, f.Text)
	}

	// Each strategy runs -runs times with fresh samples. Run 1 is shown in full
	// for eyeballing; all runs feed the reliability distribution.
	strategies := []struct {
		name string
		gen  genFunc
	}{
		{"naive", naiveGenerate},
		{"structured", generateDescription},
	}

	byMode := map[string][]runResult{}
	for i := 0; i < *runs; i++ {
		for _, s := range strategies {
			out, rep, err := runMode(ctx, c, h, facts, languages, s.gen)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s run %d: %v\n", s.name, i+1, err)
				os.Exit(1)
			}
			byMode[s.name] = append(byMode[s.name], runResult{out, rep})
		}
	}

	printMode("NAIVE BASELINE — whole JSON, each language written independently (run 1)", byMode["naive"][0].outputs, byMode["naive"][0].report, len(facts))
	printMode("STRUCTURED — facts pinned, same set for every language (run 1)", byMode["structured"][0].outputs, byMode["structured"][0].report, len(facts))
	printDistribution(*runs, byMode)
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

// The bar. Consistency is the star and strict: a set passes only if every
// language carries exactly the base fact set with no ungrounded inventions.
// Nativeness is the softer check: every language must read at >= 4/5.
func passesConsistency(r ConsistencyReport) bool { return r.Agreement >= 0.999 && r.Inventions == 0 }

func minNativeness(outputs []langOutput) int {
	m := 5
	for _, o := range outputs {
		if o.Native.Score < m {
			m = o.Native.Score
		}
	}
	return m
}

// printDistribution reports how reliably each strategy clears the bar across all
// runs — the "getting to good" evidence — and names the facts that drifted on
// failing runs.
func printDistribution(runs int, byMode map[string][]runResult) {
	fmt.Printf("\n########## RELIABILITY OVER %d RUNS ##########\n", runs)
	fmt.Println("Bar: consistency = 100% agreement AND 0 inventions (the star, strict);")
	fmt.Println("     nativeness = every language >= 4/5 (softer).")

	fmt.Printf("\nper-run  (agreement%% / inventions)\n")
	fmt.Printf("  %-5s %-16s %-16s\n", "run", "naive", "structured")
	for i := 0; i < runs; i++ {
		n, s := byMode["naive"][i].report, byMode["structured"][i].report
		fmt.Printf("  %-5d %-16s %-16s\n", i+1,
			fmt.Sprintf("%.0f%% / %d", n.Agreement*100, n.Inventions),
			fmt.Sprintf("%.0f%% / %d", s.Agreement*100, s.Inventions))
	}

	fmt.Printf("\n%-26s %-12s %-12s\n", "metric", "naive", "structured")
	row := func(label string, f func(string) string) {
		fmt.Printf("%-26s %-12s %-12s\n", label, f("naive"), f("structured"))
	}
	row("consistency pass rate", func(m string) string { return passRate(byMode[m], func(r runResult) bool { return passesConsistency(r.report) }) })
	row("mean consistency", func(m string) string {
		var s float64
		for _, r := range byMode[m] {
			s += r.report.Agreement
		}
		return pct(s / float64(runs))
	})
	row("mean inventions", func(m string) string {
		var s int
		for _, r := range byMode[m] {
			s += r.report.Inventions
		}
		return fmt.Sprintf("%.1f", float64(s)/float64(runs))
	})
	row("nativeness pass rate", func(m string) string { return passRate(byMode[m], func(r runResult) bool { return minNativeness(r.outputs) >= 4 }) })
	row("mean nativeness", func(m string) string {
		var s float64
		for _, r := range byMode[m] {
			s += avgNative(r.outputs)
		}
		return fmt.Sprintf("%.1f", s/float64(runs))
	})
	row("overall pass rate", func(m string) string {
		return passRate(byMode[m], func(r runResult) bool { return passesConsistency(r.report) && minNativeness(r.outputs) >= 4 })
	})

	fmt.Printf("\nconsistency-bar failures (facts that drifted):\n")
	any := false
	for _, m := range []string{"naive", "structured"} {
		for i, r := range byMode[m] {
			if passesConsistency(r.report) {
				continue
			}
			any = true
			fmt.Printf("  %-11s run %d: %s\n", m, i+1, failureSummary(r.report))
		}
	}
	if !any {
		fmt.Println("  (none)")
	}
}

func passRate(rs []runResult, ok func(runResult) bool) string {
	p := 0
	for _, r := range rs {
		if ok(r) {
			p++
		}
	}
	return fmt.Sprintf("%d/%d", p, len(rs))
}

// failureSummary names, per language, the facts dropped/added vs. the base and
// any ungrounded inventions — so a failing run is human-readable at a glance.
func failureSummary(r ConsistencyReport) string {
	var parts []string
	for _, lang := range sortedKeys(r.PerLang) {
		d := r.PerLang[lang]
		var seg []string
		if len(d.Dropped) > 0 {
			seg = append(seg, "drop"+fmt.Sprint(d.Dropped))
		}
		if len(d.Added) > 0 {
			seg = append(seg, "add"+fmt.Sprint(d.Added))
		}
		if len(seg) > 0 {
			parts = append(parts, lang+" "+strings.Join(seg, " "))
		}
	}
	for _, lang := range sortedKeys(r.Invented) {
		parts = append(parts, fmt.Sprintf("%s invented %v", lang, r.Invented[lang]))
	}
	if len(parts) == 0 {
		return "(agreement below bar; see run-1 detail)"
	}
	return strings.Join(parts, " | ")
}

func sortedKeys[V any](m map[string]V) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
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
