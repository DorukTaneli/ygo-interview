# Cross-language hotel localization

From one set of source facts per hotel, generate a short marketing description in
**English, German, and French** that reads natively in each language and states
**exactly the same facts** across all three — then measure it automatically.

The pipeline compares a **naive** baseline (hand the model the whole hotel JSON,
write each language independently) against a **structured** approach (pin every
language to the same atomic fact set), scores both for cross-language factual
consistency and nativeness, and runs the comparison repeatedly to show how
*reliably* each reaches the bar. See [EVALUATION.md](EVALUATION.md) for the
metrics and findings.

## Requirements

- **Go 1.26+** (`go version`)
- An **Anthropic API key**

## Setup: API key

Provide `ANTHROPIC_API_KEY` either way (an existing environment variable wins
over the file):

```sh
# Option A — .env file in the repo root
echo 'ANTHROPIC_API_KEY=sk-ant-...' > .env

# Option B — environment variable
export ANTHROPIC_API_KEY=sk-ant-...        # PowerShell: $env:ANTHROPIC_API_KEY="sk-ant-..."
```

## Run

```sh
go run . -runs 3    # choose the number of runs per strategy, used 3 during testing, defaults to 5
```

This prints the atomic fact set, the full descriptions + scores for the first
run of each strategy, and a distribution over all runs (pass rates against the
bar, means, and the specific facts that drifted on failing runs).

### Regenerating the fact set (optional)

The canonical reference — the source facts split into atomic claims — is frozen
in [atomic_facts.json](atomic_facts.json) so results are identical on every
machine. Regenerate it only if `source.json` changes:

```sh
go run . -atomize
```

## Models

Generation uses **Claude Haiku 4.5** (fast). All judging — the factual verifier
and the nativeness rating — plus the atomizer use **Claude Sonnet 4.6** at low
effort: a stronger, deliberately different model, so the judge never grades its
own output. Each run is a fresh API call (no cache), so a run of N≈3–5 costs a
few hundred short calls.
