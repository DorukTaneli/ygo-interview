package main

import (
	"context"
	"fmt"
)

// VerifyResult is the structured judgement of one description against the
// source fact set: which facts it carries, which it dropped, and any claims it
// makes that the source does not support. The json tags become the structured-
// output schema, so the model fills exactly these fields.
type VerifyResult struct {
	PresentIDs        []string `json:"present_ids"`
	MissingIDs        []string `json:"missing_ids"`
	UnsupportedClaims []string `json:"unsupported_claims"`
}

const verifySystem = `You are a meticulous fact-checker for hotel descriptions.
You are given a numbered list of source facts (each with an ID) and a marketing description written in some language. Read the description in whatever language it is written and map it back onto the source facts. Fill exactly three fields:

- present_ids: IDs of source facts the description clearly states. A faithful paraphrase counts; wording need not match.
- missing_ids: IDs of source facts the description does NOT state.
- unsupported_claims: concrete, checkable claims the description ASSERTS that no source fact supports, or that contradict a source fact — a fabricated amenity, or a wrong number, distance, or policy.

Strict rules for unsupported_claims:
- Include ONLY things the text positively asserts that are absent from, or conflict with, the source.
- NEVER list a fact the text merely left out: an omission belongs in missing_ids, not here.
- NEVER write commentary, hedges, or notes (e.g. "this is minor", "no unsupported claims found"). If there are none, return an empty array.
- Ignore generic marketing language ("a wonderful stay", "relax in comfort"); it asserts no checkable fact.
- Each entry is a short English summary of the offending claim.

Example: if the source says "dogs allowed in ground-floor rooms only" and the text says "dogs welcome throughout", that is a contradiction and belongs in unsupported_claims. If the text simply does not mention dogs, that is an omission (a missing_id), not an unsupported claim.`

// verify checks one description against the source facts via structured output,
// so the result is always schema-shaped JSON with no surrounding prose.
func verify(ctx context.Context, c Client, facts []Fact, text string) (VerifyResult, error) {
	user := fmt.Sprintf("Source facts:\n%s\nDescription:\n%s", factLines(facts), text)
	var vr VerifyResult
	if err := c.CompleteSchema(ctx, verifySystem, user, 0.0, 1024, &vr); err != nil {
		return VerifyResult{}, fmt.Errorf("verify: %w", err)
	}
	return vr, nil
}

// NativenessResult is the LLM-as-judge rating of how native a text reads.
type NativenessResult struct {
	Score  int    `json:"score"`
	Reason string `json:"reason"`
}

const nativeSystemTmpl = `You are a native-speaker language quality rater for %s.
Rate how natural the given hotel description reads on a scale of 1 to 5, judging fluency and idiom only, not factual content:
5 = reads as if a native %s copywriter wrote it; 1 = obviously a stiff or literal machine translation.
Give the integer score and a one-sentence reason.`

// nativeness rates how idiomatic a description reads in its language.
func nativeness(ctx context.Context, c Client, text, language string) (NativenessResult, error) {
	system := fmt.Sprintf(nativeSystemTmpl, language, language)
	var nr NativenessResult
	if err := c.CompleteSchema(ctx, system, text, 0.0, 512, &nr); err != nil {
		return NativenessResult{}, fmt.Errorf("nativeness: %w", err)
	}
	return nr, nil
}
