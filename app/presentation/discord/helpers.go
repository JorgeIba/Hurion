package discord

import (
	"unicode/utf16"
)

const maxMessageContentLength = 2000

// utf16Count returns the number of UTF-16 code units in a string.
// Discord's character limits (like the 1024 char field limit) are measured in UTF-16 code units.
func utf16Count(s string) int {
	return len(utf16.Encode([]rune(s)))
}
