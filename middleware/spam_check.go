package middleware

import (
	"errors"
	"regexp"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"

	e "github.com/eirka/eirka-libs/errors"
)

// wordPatterns has all the bad words we want to block
var wordPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)loli`),      // Match 'loli' anywhere in the text
	regexp.MustCompile(`(?i)lola`),      // Match 'lola' anywhere in the text
	regexp.MustCompile(`(?i)lolita`),    // Match 'lolita' anywhere in the text
	regexp.MustCompile(`(?i)lolafan`),   // Match 'lolafan' anywhere in the text
	regexp.MustCompile(`(?i)children`),  // Match 'children' anywhere in the text
	regexp.MustCompile(`(?i)jailbait`),  // Match 'jailbait' anywhere in the text
	regexp.MustCompile(`(?i)pedo`),      // Match 'pedo' anywhere in the text
	regexp.MustCompile(`(?i)cunny`),     // Match 'cunny' anywhere in the text
	regexp.MustCompile(`(?i)rape`),      // Match 'rape' anywhere in the text
	regexp.MustCompile(`(?i)torture`),   // Match 'torture' anywhere in the text
	regexp.MustCompile(`(?i)\bkid\b`),   // Match 'kid' as a whole word only
	regexp.MustCompile(`(?i)\bchild\b`), // Match 'child' as a whole word only
	regexp.MustCompile(`(?i)\bcp\b`),    // Match 'cp' as a whole word only
}

// urlPatterns has a regex for bad urls (shorteners)
var urlPatterns = []*regexp.Regexp{
	// Common URL shortener domains - explicit list of known URL shorteners
	regexp.MustCompile(`(?i)(https?:\/\/)?(bit\.ly|tinyurl\.com|t\.co|goo\.gl|is\.gd|buff\.ly|ow\.ly|tiny\.cc|shorturl\.at|cutt\.ly)\/[A-Za-z0-9_-]+`),
	// General pattern for very short domains (1-4 chars) with short paths - likely shorteners
	regexp.MustCompile(`(?i)(https?:\/\/)?([A-Za-z0-9][A-Za-z0-9-]{0,3})\.[A-Za-z]{2,3}\/[A-Za-z0-9]{1,7}(\s|$)`),
}

// SpamFilter will check for banned words in the post
func SpamFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the comment from the form
		comment := c.PostForm("comment")

		// Empty comments are valid if there's an image attached
		// The model validation will handle the case where both comment and image are missing
		if comment == "" {
			c.Next()
			return
		}

		// Check for banned words by stripping all non-alphanumeric characters first
		if containsWords(stripNonAlpha(comment), wordPatterns...) {
			c.JSON(e.ErrorMessage(e.ErrForbidden))
			c.Error(errors.New("banned word pattern detected")).SetMeta("SpamFilter.containsWords")
			c.Abort()
			return
		}

		// Check for bad URLs (like URL shorteners)
		if containsWords(comment, urlPatterns...) {
			c.JSON(e.ErrorMessage(e.ErrForbidden))
			c.Error(errors.New("URL pattern detected")).SetMeta("SpamFilter.containsUrls")
			c.Abort()
			return
		}

		c.Next()
	}
}

// stripNonAlpha removes all non-alphanumeric characters
func stripNonAlpha(input string) string {
	// Early return for empty strings
	if input == "" {
		return ""
	}

	// Pre-allocate the result to avoid multiple allocations
	// Estimated final size is the same as input (worst case)
	var builder strings.Builder
	builder.Grow(len(input))

	for _, r := range input {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

// containsWords checks the comment for all the regex filters
func containsWords(s string, patterns ...*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(s) {
			return true
		}
	}
	return false
}
