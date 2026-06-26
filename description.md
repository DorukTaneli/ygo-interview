# YGO - AI/ML Product Engineering Challenge (Localization)

# 🧠 Task: localized hotel descriptions that stay true to the source

**Context:** YGO turns travel content into trustworthy, AI-friendly data. When an AI writes a hotel description in several languages from the same source, the versions **drift**: the German text mentions the pool, the French one forgets it, the English one invents a spa. We need every language to carry the **same facts** while reading **natively**.

**Your task:** from one set of source facts per hotel, produce a short marketing description in **English, German and French** that reads natively in each language **and** states exactly the same facts across all three. Then prove it.

---

## ‼️ AI-only development (CRITICAL)

🔴 **You write zero code. Not one line.** Everything is produced by **Claude Code**. Any manually written code means disqualification.

🔴 **Prompt log is mandatory.** Have Claude Code append every prompt you give and every response it returns to a `/prompt-log.md` file in the repo, as you go. **No prompt log, no review.** This is how we see how you actually work.

---

## ⏳ Time: 3 hours, strict

Reply with your exact start date, time and timezone before you begin. Send the GitHub link within 3 hours of that. Commits after the 3-hour mark are checked by git timestamp and ignored.

---

## 📌 What to build

**Source data (provided, nothing to fetch):**

```json
[
  {
    "id": "hotel-001",
    "name": "Strandhaus Aurora",
    "city": "Sylt",
    "country": "Germany",
    "setting": "beachfront",
    "amenities": ["heated outdoor pool", "spa with sauna (adults only)", "two restaurants", "free bike rental", "EV charging"],
    "rooms": ["double rooms", "sea-view suites", "family apartments for up to 4"],
    "nearby": ["3 minute walk to the beach", "15 minutes to Westerland centre", "guided tidal-flat hikes"],
    "policies": ["breakfast included", "dogs allowed in ground-floor rooms only", "free cancellation up to 14 days before arrival"],
    "price_band": "premium"
  },
  {
    "id": "hotel-002",
    "name": "Pinecrest Lodge",
    "city": "Chamonix",
    "country": "France",
    "setting": "mountain village",
    "amenities": ["ski-in ski-out access", "indoor pool", "sauna", "boot room with heated racks", "on-site bistro"],
    "rooms": ["standard twins", "chalet-style family rooms for up to 5", "one accessible ground-floor room"],
    "nearby": ["100 m to the nearest ski lift", "10 minutes to Chamonix centre by free shuttle"],
    "policies": ["half board available for a surcharge", "pets welcome", "ski storage included", "free cancellation up to 30 days before arrival"],
    "price_band": "premium"
  }
]
```

1. **Generate** a short, native-sounding description for each hotel in English (base), German and French. Native, not a literal translation. You choose which facts from the source to feature in the English base; you do not have to use every fact.
2. **Define what "good" means and measure it automatically.** Two things matter and they are not weighted equally. **(a) Factual consistency is the star, weighted heaviest:** the German and French versions state exactly the facts the English version states, with nothing dropped, added, invented or contradicted **between languages**. Whatever facts the English base features, all three languages carry that same set, and none of the three states anything the source does not support. **(b) Nativeness is the softer check, weighted less:** each text reads as if a native speaker wrote it, not a literal translation. You are not judged on speaking the languages, so the checks must run without you (an LLM-as-judge is fine). Decide what score counts as good for each.
3. **Make the output reliably good and show how you got there.** A single generation will drift. We want descriptions that consistently meet your bar, plus evidence of the path: what your checks caught, what you changed, how the scores moved.

Use any LLM provider. Include setup steps so we can run it with our own API key.

---

## 📦 Deliverables

1. A public **GitHub repository**
2. **`/prompt-log.md`**, the full Claude Code prompt and response log (mandatory)
3. **`EVALUATION.md`** (about half a page): the metrics you chose and why, how you check factual consistency across languages without speaking them, what was wrong at first, what you changed, how the scores improved, what you would do with more time
4. **`README.md`**: setup, how to pass an API key, how to run everything

---

## 🛠️ Notes

- Any source code must be written in Go.
- No UI required. A script that runs it and prints the scores is enough.

---

## 💡 What we look for

- **Measuring quality (most important):** automatic checks grounded in the source facts, real cross-language consistency, a sensible bar
- **Getting to good:** evidence the descriptions reliably reach that bar, not one lucky output
- **AI-agent workflow:** effective Claude Code use, a clear and complete prompt log and evidence you reviewed and steered what the agent produced rather than accepting it blindly
- **Product sense:** descriptions a real traveller would trust
- A clear **EVALUATION.md**

🔴 Zero manual code · 🔴 Prompt log mandatory · 🔴 Confirm start time · 🔴 3 hours, no extensions
