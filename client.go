package main

import (
	"context"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// modelID is the LLM used for every call. Sonnet 4.6 per the task brief.
const modelID = anthropic.ModelClaudeSonnet4_6

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
func (c Client) Complete(ctx context.Context, system, user string, temperature float64, maxTokens int64) (string, error) {
	resp, err := c.api.Messages.New(ctx, anthropic.MessageNewParams{
		Model:       modelID,
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
// parsing JSON out of free text. Structured outputs is GA on Sonnet 4.6; this
// SDK exposes it on the Beta endpoint, which unmarshals into dest for us.
func (c Client) CompleteSchema(ctx context.Context, system, user string, temperature float64, maxTokens int64, dest any) error {
	_, err := c.api.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
		Model:       modelID,
		MaxTokens:   maxTokens,
		Temperature: anthropic.Float(temperature),
		System:      []anthropic.BetaTextBlockParam{{Text: system}},
		Messages: []anthropic.BetaMessageParam{
			anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(user)),
		},
		OutputConfig: anthropic.BetaOutputConfigParam{
			Format: anthropic.BetaJSONOutputFormatParam{Schema: dest},
		},
	})
	return err
}
