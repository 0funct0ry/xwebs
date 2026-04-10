package template

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestColorToggle(t *testing.T) {
	e := New(false)
	
	// Default is on
	res, err := e.Execute("test", "{{red \"hello\"}}", nil)
	assert.NoError(t, err)
	assert.Equal(t, "\033[31mhello\033[0m", res)
	
	// Turn off
	e.SetColorsEnabled(false)
	res, err = e.Execute("test", "{{red \"hello\"}}", nil)
	assert.NoError(t, err)
	assert.Equal(t, "hello", res)
	
	// Nested styles with off
	res, err = e.Execute("test", "{{bold (green \"hello\")}}", nil)
	assert.NoError(t, err)
	assert.Equal(t, "hello", res)
	
	// Turn back on
	e.SetColorsEnabled(true)
	res, err = e.Execute("test", "{{bold (green \"hello\")}}", nil)
	assert.NoError(t, err)
	assert.Contains(t, res, "\033[1m")
	assert.Contains(t, res, "\033[32m")
}

func TestShortHelper(t *testing.T) {
	e := New(false)
	
	tests := []struct {
		input    string
		expected string
	}{
		{"abcdefgh", "abcdefgh"},
		{"abcdefghi", "abcdefgh"},
		{"abc", "abc"},
		{"", ""},
		{"1234567890", "12345678"},
	}
	
	for _, tt := range tests {
		res, err := e.Execute("test", "{{short .}}", tt.input)
		assert.NoError(t, err)
		assert.Equal(t, tt.expected, res)
	}
}

func TestFormattingHelpers(t *testing.T) {
	e := New(false)
	
	// Truncate (already exists but good to verify it's registered)
	res, err := e.Execute("test", "{{truncate 5 \"hello world\"}}", nil)
	assert.NoError(t, err)
	assert.Equal(t, "hello...", res)
	
	// padLeft
	res, err = e.Execute("test", "{{padLeft 10 \"hi\"}}", nil)
	assert.NoError(t, err)
	assert.Equal(t, "        hi", res)
	
	// padRight
	res, err = e.Execute("test", "{{padRight 10 \"hi\"}}", nil)
	assert.NoError(t, err)
	assert.Equal(t, "hi        ", res)
}
