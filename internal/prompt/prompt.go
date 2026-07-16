// Package prompt builds the interactive text form Novelist shows the user.
package prompt

import "charm.land/huh/v2"

// CharLimit caps the story length at 10 KB.
const CharLimit = 10 * 1024

// New returns a text form bound to story.
func New(story *string) *huh.Text {
	return huh.NewText().
		Title("Tell me a story.").
		Value(story).
		Placeholder("What's on your mind?").
		CharLimit(CharLimit)
}
