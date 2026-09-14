// Package filter compiles case-insensitive title and filename pattern lists
// shared by the torrent processor and the webhook publisher.
package filter

import (
	"fmt"
	"regexp"
)

// Patterns holds compiled case-insensitive regex patterns. Patterns within a
// list are OR-ed by the callers.
type Patterns struct {
	titles    []*regexp.Regexp
	filenames []*regexp.Regexp
}

// Compile compiles title and filename patterns case-insensitively.
func Compile(titlePatterns, filenamePatterns []string) (*Patterns, error) {
	p := &Patterns{}

	for _, pattern := range titlePatterns {
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid title pattern %q: %w", pattern, err)
		}

		p.titles = append(p.titles, re)
	}

	for _, pattern := range filenamePatterns {
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid filename pattern %q: %w", pattern, err)
		}

		p.filenames = append(p.filenames, re)
	}

	return p, nil
}

func (p *Patterns) Titles() []*regexp.Regexp {
	return p.titles
}

func (p *Patterns) Filenames() []*regexp.Regexp {
	return p.filenames
}

// AnyMatch reports whether s matches any of the patterns.
func AnyMatch(patterns []*regexp.Regexp, s string) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(s) {
			return true
		}
	}

	return false
}
