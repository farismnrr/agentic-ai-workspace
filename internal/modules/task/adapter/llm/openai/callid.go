package openai

import (
	"crypto/rand"
	"encoding/hex"
)

const callIDBytes = 8

// callID keeps the provider's tool-call id, or invents one when a gateway (Gemini's
// compatible endpoint can) leaves it empty, so the result can be matched to its call.
func callID(id string) string {
	if id != "" {
		return id
	}

	buf := make([]byte, callIDBytes)
	_, _ = rand.Read(buf)

	return "call_" + hex.EncodeToString(buf)
}
