package prompt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewContainsTitle(t *testing.T) {
	var story string
	p := New(&story)
	assert.Contains(t, p.View(), "Tell me a story", "prompt renders the title")
}
