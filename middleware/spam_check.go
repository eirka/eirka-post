package middleware

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"

	e "github.com/eirka/eirka-libs/errors"
)

// patterns has all the bad words we want to block
var patterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i:kid)`),
	regexp.MustCompile(`(?i:loli)`),
	regexp.MustCompile(`(?i:lola)`),
	regexp.MustCompile(`(?i:lolita)`),
	regexp.MustCompile(`(?i:lolafan)`),
	regexp.MustCompile(`(?i:child)`),
	regexp.MustCompile(`(?i:children)`),
	regexp.MustCompile(`(?i:jailbait)`),
	regexp.MustCompile(`(?i:pedo)`),
	regexp.MustCompile(`(?i:cp)`),
	regexp.MustCompile(`(?i:rape)`),
	regexp.MustCompile(`(?i:torture)`),
	regexp.MustCompile(`(?i:(https?:\/\/)?(www\.)?([A-Za-z0-9\-]+\.)+[A-Za-z]{2,6}(\/[A-Za-z]{1})?\/[\-A-Za-z0-9]+)`),
}

// SpamFilter will check for banned words in the post
func SpamFilter() gin.HandlerFunc {
	return func(c *gin.Context) {

		if containsWords(stripNonAlpha(c.PostForm("comment")), patterns...) {
			c.JSON(e.ErrorMessage(e.ErrForbidden))
			c.Error(e.ErrBlacklist).SetMeta("SpamFilter.containsWords")
			c.Abort()
			return
		}

		c.Next()

	}
}

// stripNonAlpha removes all weird characters
func stripNonAlpha(input string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		return -1
	}, input)
}

// containsWords checks the comment for all the regex filters
func containsWords(s string, patterns ...*regexp.Regexp) bool {
	println(s)
	for _, pattern := range patterns {
		if pattern.MatchString(s) {
			return true
		}
	}
	return false
}
