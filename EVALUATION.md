# Evaluation

## Metrics and the bar
Two metrics, weighted unequally.

- **Cross-language consistency — the star, strict.** English is the reference: whatever atomic facts the English base features, German and French must carry the same set — nothing dropped, added, or invented. Score = mean fact-set agreement of DE/FR vs. EN, plus a count of ungrounded inventions. **Bar: 100% agreement and 0 inventions** — for trusted travel data, a single dropped or invented fact is a real defect.
- **Nativeness — softer.** An LLM rates each text 1–5 on idiom and fluency only. **Bar: every language ≥ 4.**

## Checking consistency without speaking the languages
The source facts are split into atomic claims with stable IDs (frozen in `atomic_facts.json`). A verifier LLM reads each description — in any language — and maps it back onto those English fact IDs: `present` / `missing` / `unsupported`. Comparing the three `present` sets exposes drift; `unsupported` catches inventions.

## What was wrong → what I changed
- Judges appended reasoning that broke free-text JSON parsing → **structured outputs** (schema-constrained, no prose).
- Compound facts (`spa with sauna (adults only)`) hid sub-fact drift and caused false inventions → **atomized** into separate claims; inventions 2 → 0, and the metric could finally see the drift.
- An all-Haiku judge flagged a *grounded* fact as an invention → **split models**: Haiku drafts, Sonnet does all judging (at low effort). A stronger, deliberately different model means the judge never grades its own output.

## Reliability (N runs, fresh samples)
Each run is independent sampling, so the numbers move run to run — reproduce the
*pattern*, not the digits. Observed across several N=3 runs:

| per N=3 run | naive | structured |
|---|---|---|
| consistency pass | 0/3 | 2–3 of 3 |
| mean consistency | ~65–70% | ~98–100% |
| mean inventions | ~1 | 0 |
| mean nativeness | ~4.6 | ~4.3 |

The structured pipeline clears the consistency bar on most runs. The naive baseline never clears it. What then limits structured is *nativeness*, not facts: packing all 16 facts into 2–4 sentences reads denser and occasionally dips below 4, while naive flows better by cherry-picking — the consistency/nativeness tradeoff.

## With more time in production

With more time in production, I would use a one higher tier for LLMs: Sonnet for generation, Opus as judge. Clearing these bars consistently is very important for business. Even a few misses would cause trust issues.

I would use frameworks rather than custom evaluation scripts.
Popular LLM evaluation frameworks are DeepEval, Promptfoo, and Langfuse. These can also be run on CI/CD to catch issues before code goes live.

For monitoring of cost, speed, latency, and also drift detection on how we perform over time, we can use open source LLM observability frameworks such as Langfuse, or Arize Phoenix if we are already using OpenTelemetry.
