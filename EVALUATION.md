# Evaluation

## Making the output reliably good

**Atomizing the source facts.** Compound source lines hid cross-language drift: all three languages could mark `"spa with sauna (adults only)"` "present" while each dropped a different detail. A cached, temperature-0 step splits each line into atomic claims — `spa` / `sauna` / `adults-only` — which become the reference everything is scored against, so the metric finally sees sub-fact drift.
