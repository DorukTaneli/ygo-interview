package main

import (
	"context"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// Models. Haiku drafts the descriptions; Sonnet does all the judging — both the
// factual verifier and the nativeness rating — and the atomization. Using a
// stronger, deliberately different model for every judgement keeps the judge from
// grading its own output.
const (
	fastModel   = anthropic.ModelClaudeHaiku4_5_20251001
	strongModel = anthropic.ModelClaudeSonnet4_6
)

// effortLow runs the Sonnet structured-output calls at low effort: it handles
// these judging and atomizing tasks well without extra effort, keeping runs fast.
const effortLow = anthropic.BetaOutputConfigEffortLow

// Client wraps the Anthropic SDK. Calls are made fresh every time — the
// reliability experiment depends on independent samples, so there is no response
// cache. The one thing that must stay stable across runs, the atomic fact set,
// is frozen in atomic_facts.json instead.
type Client struct {
	api anthropic.Client
}

// NewClient builds a client. The API key is read from ANTHROPIC_API_KEY.
func NewClient() Client {
	return Client{api: anthropic.NewClient()}
}

// Complete sends a single-turn request and returns the concatenated text of the
// response.
func (c Client) Complete(ctx context.Context, model anthropic.Model, system, user string, temperature float64, maxTokens int64) (string, error) {
	resp, err := c.api.Messages.New(ctx, anthropic.MessageNewParams{
		Model:       model,
		MaxTokens:   maxTokens,
		Temperature: anthropic.Float(temperature),
		System:      []anthropic.TextBlockParam{{Text: system}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(user)),
		},
	})
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			sb.WriteString(t.Text)
		}
	}
	return sb.String(), nil
}

// CompleteSchema constrains the model's reply to the JSON schema reflected from
// dest (a pointer to a struct) using structured outputs, then unmarshals the
// reply into dest. Because the output is schema-constrained, the model cannot
// wrap it in prose or append chain-of-thought — the failure mode that breaks
// parsing JSON out of free text. effort may be effortLow to cap the work the
// model does; the zero value leaves it at the API default. Structured outputs is
// GA on these models; this SDK exposes it on the Beta endpoint, which unmarshals
// into dest for us.
func (c Client) CompleteSchema(ctx context.Context, model anthropic.Model, system, user string, temperature float64, maxTokens int64, effort anthropic.BetaOutputConfigEffort, dest any) error {
	_, err := c.api.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
		Model:       model,
		MaxTokens:   maxTokens,
		Temperature: anthropic.Float(temperature),
		System:      []anthropic.BetaTextBlockParam{{Text: system}},
		Messages: []anthropic.BetaMessageParam{
			anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(user)),
		},
		OutputConfig: anthropic.BetaOutputConfigParam{
			Effort: effort,
			Format: anthropic.BetaJSONOutputFormatParam{Schema: dest},
		},
	})
	return err
}
