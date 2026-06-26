package main

import "sort"

// LangDiff describes how one language's fact set differs from the base.
type LangDiff struct {
	Agreement  float64  // Jaccard overlap of present facts with the base set
	Dropped    []string // facts the base features that this language omits
	Added      []string // facts this language features that the base does not
	Inventions []string // claims in this language unsupported by any source fact
}

// ConsistencyReport scores cross-language factual consistency — the heaviest-
// weighted "star" metric. English is the reference: whatever facts it features,
// German and French should feature the same set, with no ungrounded inventions.
type ConsistencyReport struct {
	Base       string              // reference language (English)
	Agreement  float64             // mean fact-set agreement of non-base languages vs. base
	Inventions int                 // total ungrounded claims across all languages
	PerLang    map[string]LangDiff // keyed by non-base language
	Invented   map[string][]string // language -> ungrounded claims (all languages, base included)
}

// crossLanguageConsistency derives the metric from the per-language verify
// results. No extra LLM calls: it compares the present_ids sets the fact-checker
// already produced, treating the base language's set as ground truth.
func crossLanguageConsistency(base string, results map[string]VerifyResult) ConsistencyReport {
	baseSet := toSet(results[base].PresentIDs)
	rep := ConsistencyReport{Base: base, PerLang: map[string]LangDiff{}, Invented: map[string][]string{}}

	var agreements []float64
	for lang, vr := range results {
		rep.Inventions += len(vr.UnsupportedClaims)
		if len(vr.UnsupportedClaims) > 0 {
			rep.Invented[lang] = vr.UnsupportedClaims
		}
		if lang == base {
			continue
		}
		set := toSet(vr.PresentIDs)
		diff := LangDiff{
			Agreement:  jaccard(baseSet, set),
			Dropped:    sortedDiff(baseSet, set),
			Added:      sortedDiff(set, baseSet),
			Inventions: vr.UnsupportedClaims,
		}
		rep.PerLang[lang] = diff
		agreements = append(agreements, diff.Agreement)
	}
	rep.Agreement = mean(agreements)
	return rep
}

func toSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, it := range items {
		s[it] = true
	}
	return s
}

// sortedDiff returns the sorted elements of a that are not in b.
func sortedDiff(a, b map[string]bool) []string {
	var out []string
	for k := range a {
		if !b[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// jaccard is |a∩b| / |a∪b|; two empty sets count as identical (1.0).
func jaccard(a, b map[string]bool) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0
	}
	inter, union := 0, len(a)
	for k := range b {
		if a[k] {
			inter++
		} else {
			union++
		}
	}
	return float64(inter) / float64(union)
}

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 1.0
	}
	var s float64
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}
