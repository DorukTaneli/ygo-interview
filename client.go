package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// modelID is the LLM used for every call. Sonnet 4.6 per the task brief.
const modelID = anthropic.ModelClaudeSonnet4_6

// Client wraps the Anthropic SDK with a disk cache so repeated runs during
// development don't re-pay for identical requests.
type Client struct {
	api      anthropic.Client
	cacheDir string
}

// NewClient builds a client. The API key is read from ANTHROPIC_API_KEY.
func NewClient() Client {
	return Client{
		api:      anthropic.NewClient(),
		cacheDir: "cache",
	}
}

// Complete sends a single-turn request and returns the concatenated text of
// the response. Responses are cached on disk keyed by the full request, so an
// identical call is served from cache.
func (c Client) Complete(ctx context.Context, system, user string, temperature float64, maxTokens int64) (string, error) {
	key := cacheKey(string(modelID), system, user, fmt.Sprintf("%.2f", temperature))
	if cached, ok := c.readCache(key); ok {
		return cached, nil
	}

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
	out := sb.String()
	c.writeCache(key, out)
	return out, nil
}

// CompleteSchema constrains the model's reply to the JSON schema reflected from
// dest (a pointer to a struct) using structured outputs, then unmarshals the
// reply into dest. Because the output is schema-constrained, the model cannot
// wrap it in prose or append chain-of-thought — the failure mode that breaks
// parsing JSON out of free text. Structured outputs is GA on Sonnet 4.6; this
// SDK exposes it on the Beta endpoint. Responses are cached on disk like Complete.
func (c Client) CompleteSchema(ctx context.Context, system, user string, temperature float64, maxTokens int64, schemaName string, dest any) error {
	key := cacheKey(string(modelID), "schema:"+schemaName, system, user, fmt.Sprintf("%.2f", temperature))
	if cached, ok := c.readCache(key); ok {
		return json.Unmarshal([]byte(cached), dest)
	}

	res, err := c.api.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
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
	if err != nil {
		return err
	}

	// Passing a struct pointer as Schema makes the SDK both generate the schema
	// and unmarshal the response into dest; we only need the raw JSON to cache.
	var raw string
	for _, block := range res.Content {
		if block.Type == "text" {
			raw = block.Text
			break
		}
	}
	c.writeCache(key, raw)
	return nil
}

func cacheKey(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (c Client) readCache(key string) (string, bool) {
	b, err := os.ReadFile(filepath.Join(c.cacheDir, key+".txt"))
	if err != nil {
		return "", false
	}
	return string(b), true
}

func (c Client) writeCache(key, val string) {
	_ = os.MkdirAll(c.cacheDir, 0o755)
	_ = os.WriteFile(filepath.Join(c.cacheDir, key+".txt"), []byte(val), 0o644)
}
