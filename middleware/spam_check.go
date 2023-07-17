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
}

// urlPatterns has a regex for bad urls (shorteners)
var urlPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i:(https?:\/\/)?(www\.)?([A-Za-z0-9\-]+\.)+[A-Za-z]{2,6}\/(.\/)?[\_\-A-Za-z0-9]+)$`),
}

// SpamFilter will check for banned words in the post
func SpamFilter() gin.HandlerFunc {
	return func(c *gin.Context) {

		//we strip all the weird characters and spacing here to check for bad words
		if containsWords(stripNonAlpha(c.PostForm("comment")), wordPatterns...) {
			c.JSON(e.ErrorMessage(e.ErrForbidden))
			c.Error(errors.New(c.PostForm("comment"))).SetMeta("SpamFilter.containsWords")
			c.Abort()
			return
		}

		//we check for bad urls here
		if containsWords(c.PostForm("comment"), urlPatterns...) {
			c.JSON(e.ErrorMessage(e.ErrForbidden))
			c.Error(errors.New(c.PostForm("comment"))).SetMeta("SpamFilter.containsWords")
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
	for _, pattern := range patterns {
		if pattern.MatchString(s) {
			return true
		}
	}
	return false
}
