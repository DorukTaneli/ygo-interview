# Evaluation

## Metrics and the bar
Two metrics, weighted unequally.

- **Cross-language consistency — the star, strict.** English is the reference: whatever atomic facts the English base features, German and French must carry the same set — nothing dropped, added, or invented. Score = mean fact-set agreement of DE/FR vs. EN, plus a count of ungrounded inventions. **Bar: 100% agreement and 0 inventions** — for trusted travel data, a single dropped or invented fact is a real defect.
- **Nativeness — softer.** An LLM rates each text 1–5 on idiom and fluency only. **Bar: every language ≥ 4.**

## Checking consistency without speaking the languages
I don't compare prose; I reduce it to a grounded set comparison. The source facts are split into atomic claims with stable IDs (frozen in `atomic_facts.json`). A verifier LLM reads each description — in any language — and maps it back onto those English fact IDs: `present` / `missing` / `unsupported`. Comparing the three `present` sets exposes drift; `unsupported` catches inventions. German is graded against an English fact list, so I never read German.

## What was wrong → what I changed
- Judges appended reasoning that broke free-text JSON parsing → **structured outputs** (schema-constrained, no prose).
- Compound facts (`spa with sauna (adults only)`) hid sub-fact drift and caused false inventions → **atomized** into separate claims; inventions 2 → 0, and the metric could finally see the drift.
- An all-Haiku judge flagged a *grounded* fact as an invention → **split models**: Haiku drafts, Sonnet does all judging (at low effort). A stronger, deliberately different model means the judge never grades its own output.

## Reliability (N runs, fresh samples)
| over 3 runs (representative) | naive | structured |
|---|---|---|
| consistency pass | 0/3 | 3/3 |
| mean consistency | 69% | 100% |
| mean inventions | 0.7 | 0.0 |
| mean nativeness | 4.6 | 4.3 |

The structured pipeline reaches the strict consistency bar reliably (≈100%, 0 inventions); the naive baseline never does (≈70%, drifting on different facts each run). Each run is fresh sampling, so the exact figures move — re-run to reproduce the pattern, not the digits. What now limits structured is *nativeness*, not facts: packing all 16 facts into 2–4 sentences reads denser and occasionally dips below 4, while naive flows better by cherry-picking — the consistency/nativeness tradeoff.

## With more time
Per-fact severity weighting; a regenerate-on-failure loop to lift the pass rate; a second judge or human spot-check to validate the LLM judge; more hotels.
