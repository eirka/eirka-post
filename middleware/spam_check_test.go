package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripNonAlpha(t *testing.T) {

	assert := assert.New(t)

	// Test with empty string
	assert.Equal("", stripNonAlpha(""), "should be equal")

	// Test with string with spaces
	assert.Equal("test", stripNonAlpha(" test "), "should be equal")

	// Test with string with weird characters
	assert.Equal("test", stripNonAlpha("t!e@s#t$"), "should be equal")

	// Test with string with weird characters and spaces
	assert.Equal("test", stripNonAlpha(" t!e@s#t$ "), "should be equal")

}

func TestContainsWords(t *testing.T) {

	assert := assert.New(t)

	// Test with empty string
	assert.False(containsWords("", wordPatterns...), "should be false")

	// Test with string with spaces
	assert.False(containsWords(" test ", wordPatterns...), "should be false")

	// Test with string with weird characters
	assert.False(containsWords("t!e@s#t$", wordPatterns...), "should be false")

	// Test with string with weird characters and spaces
	assert.False(containsWords(" t!e@s#t$ ", wordPatterns...), "should be false")

	// Test with string with weird characters and spaces
	assert.True(containsWords("loli", wordPatterns...), "should be true")

	// Test with string with spaces
	assert.True(containsWords(" loli ", wordPatterns...), "should be true")

	// Test with string with weird characters
	assert.True(containsWords(stripNonAlpha("l!o@l#i$"), wordPatterns...), "should be true")

	// Test with string with weird characters and spaces
	assert.True(containsWords(stripNonAlpha(" l!o@l#i$ "), wordPatterns...), "should be true")

}
