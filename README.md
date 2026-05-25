# Probability Engine

`probability-engine` is a Go library and standalone service for probabilistic event classification.

It converts raw events into:

- assessments
- signals
- confidence scores
- market intelligence labels

Designed for:

- financial news
- trade policy
- geopolitical events
- macroeconomic analysis

---

## Features

- Rule-based classification
- Naive Bayes classification
- Keyword scoring
- Signal generation
- Loki integration
- S3 persistence
- Embeddable Go module
- Standalone worker mode

---

## Install

```bash
go get github.com/galois26/probability-engine
```

---

## Quick Example

```go
import (
    "context"

    prob "github.com/galois26/probability-engine/pkg/probability"
)

func main() {
    engine, err := prob.NewDefaultEngine(prob.Config{
        RulesDir: "./rules",
    })
    if err != nil {
        panic(err)
    }

    events := []prob.Event{
        {
            ID:      "evt-1",
            Source:  "newsdata",
            Title:   "US announces new semiconductor export controls",
            Summary: "New trade restrictions introduced...",
        },
    }

    assessments, err := engine.Assess(context.Background(), events)
    if err != nil {
        panic(err)
    }

    _ = assessments
}
```

---

## Example Output

```json
{
  "eventId": "evt-1",
  "decision": {
    "accepted": true,
    "primaryClass": "sanctions",
    "confidence": 0.91
  }
}
```

---

## Structure

```text
cmd/engine          Standalone service
pkg/probability     Public API
internal/           Engine internals
rules/              Classification rules
```

---

## Modes

### Embedded Mode (Recommended)

Use inside another Go service such as:

- multi-ingester
- streaming pipelines
- event processors

Benefits:

- shared deduplication
- simpler infrastructure
- lower latency
- no duplicate polling

### Standalone Mode

Run independently with Loki polling and persistence.

Useful for:

- replay
- batch processing
- experimentation

---

## Run Standalone

```bash
go run ./cmd/engine
```

---

## Development

```bash
go test ./...
go fmt ./...
```

---

## License

MIT