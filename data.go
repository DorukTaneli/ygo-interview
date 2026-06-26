package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed source.json
var sourceJSON []byte

// Hotel mirrors one entry in the provided source data.
type Hotel struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	City      string   `json:"city"`
	Country   string   `json:"country"`
	Setting   string   `json:"setting"`
	Amenities []string `json:"amenities"`
	Rooms     []string `json:"rooms"`
	Nearby    []string `json:"nearby"`
	Policies  []string `json:"policies"`
	PriceBand string   `json:"price_band"`
}

// Fact is one atomic, source-grounded claim with a stable ID. The ID is what
// lets us compare languages back against the source without reading them.
type Fact struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// Facts flattens the source arrays into atomic facts with stable IDs. The
// source data is already atomic, so this needs no LLM call.
func (h Hotel) Facts() []Fact {
	var facts []Fact
	add := func(prefix string, items []string) {
		for i, it := range items {
			facts = append(facts, Fact{ID: fmt.Sprintf("%s-%d", prefix, i), Text: it})
		}
	}
	add("amenity", h.Amenities)
	add("room", h.Rooms)
	add("nearby", h.Nearby)
	add("policy", h.Policies)
	return facts
}

// JSON renders the hotel as indented JSON — the raw blob handed to the naive
// baseline generator. Marshalling a struct of strings cannot fail, so the
// error is intentionally dropped to keep call sites clean.
func (h Hotel) JSON() string {
	b, _ := json.MarshalIndent(h, "", "  ")
	return string(b)
}

// LoadHotels parses the embedded source data.
func LoadHotels() ([]Hotel, error) {
	var hotels []Hotel
	if err := json.Unmarshal(sourceJSON, &hotels); err != nil {
		return nil, err
	}
	return hotels, nil
}
