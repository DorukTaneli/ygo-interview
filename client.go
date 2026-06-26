package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
