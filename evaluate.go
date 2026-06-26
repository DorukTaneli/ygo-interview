package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// VerifyResult is the structured judgement of one description against the
// source fact set: which facts it carries, which it dropped, and any claims it
// makes that the source does not support.
type VerifyResult struct {
	PresentIDs        []string `json:"present_ids"`
	MissingIDs        []string `json:"missing_ids"`
	UnsupportedClaims []string `json:"unsupported_claims"`
}

const verifySystem = `You are a meticulous fact-checker for hotel descriptions.
You are given a list of source facts (each with an ID) and a marketing description written in some language.
Map the description back onto the facts and respond with ONLY a JSON object (no markdown fences) of the form:
{"present_ids": ["amenity-0", ...], "missing_ids": ["policy-2", ...], "unsupported_claims": ["short English summary of a claim not supported by the source", ...]}

Rules:
- A fact is "present" if the description clearly states it; a faithful paraphrase counts.
- "missing_ids" are facts from the list the description does not state.
- "unsupported_claims" are concrete claims in the description (amenities, numbers, distances, policies) that NONE of the source facts support. Ignore generic marketing language ("a wonderful stay", "relax in comfort").
- Read the description in whatever language it is written; express every output in English fact IDs / English summaries.`

// verify checks one description against the source facts.
func verify(ctx context.Context, c Client, facts []Fact, text string) (VerifyResult, error) {
	user := fmt.Sprintf("Source facts:\n%s\nDescription:\n%s", factLines(facts), text)
	raw, err := c.Complete(ctx, verifySystem, user, 0.0, 1024)
	if err != nil {
		return VerifyResult{}, err
	}
	var vr VerifyResult
	if err := json.Unmarshal([]byte(extractJSON(raw)), &vr); err != nil {
		return VerifyResult{}, fmt.Errorf("parsing verify JSON: %w (raw: %q)", err, raw)
	}
	return vr, nil
}

// NativenessResult is the LLM-as-judge rating of how native a text reads.
type NativenessResult struct {
	Score  int    `json:"score"`
	Reason string `json:"reason"`
}

const nativeSystemTmpl = `You are a native-speaker language quality rater for %s.
Rate how natural the given hotel description reads, on a scale of 1 to 5:
5 = reads as if a native %s copywriter wrote it; 1 = obviously a stiff or literal machine translation.
Judge fluency and idiom only, not factual content.
Respond with ONLY a JSON object (no markdown fences): {"score": <1-5 integer>, "reason": "<one short sentence>"}`

// nativeness rates how idiomatic a description reads in its language.
func nativeness(ctx context.Context, c Client, text, language string) (NativenessResult, error) {
	system := fmt.Sprintf(nativeSystemTmpl, language, language)
	raw, err := c.Complete(ctx, system, text, 0.0, 512)
	if err != nil {
		return NativenessResult{}, err
	}
	var nr NativenessResult
	if err := json.Unmarshal([]byte(extractJSON(raw)), &nr); err != nil {
		return NativenessResult{}, fmt.Errorf("parsing nativeness JSON: %w (raw: %q)", err, raw)
	}
	return nr, nil
}

// extractJSON returns the substring from the first '{' to the last '}', so a
// stray markdown fence or preamble around the JSON doesn't break parsing.
func extractJSON(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || end < start {
		return s
	}
	return s[start : end+1]
}
