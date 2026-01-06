package goignore

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func bufferLengthForPathComponents() int {
	return 2048
}

func maxPathLength() int {
	// You need atleast 1 character between each separator character for mySplitBuf() to use up a path component
	// e.g. "a/a/a/a/a/..."
	return 2 * bufferLengthForPathComponents()
}

// this is my own implementation of strings.Split()
// for my use case, this is way faster than the stdlib one
// the function expects a slice of sufficient length to get passed to it,
// this avoids unnecessary memory allocation
func mySplitBuf(s string, sep byte, pathComponentsBuf []string) []string {
	idx := 0
	l := 0
	for {
		pos := strings.IndexByte(s[l:], sep)

		if pos == -1 {
			break
		}

		absolutePos := l + pos
		if absolutePos > l {
			pathComponentsBuf[idx] = s[l:absolutePos]
			idx++
		}
		l = absolutePos + 1
	}
	// handle the last part separately
	if l < len(s) {
		pathComponentsBuf[idx] = s[l:]
		idx++
	}

	// truncate the slice to the actual number of components
	return pathComponentsBuf[:idx]
}

// this is my own implementation of strings.Split()
// for my use case, this is better than the stdlib one
func mySplit(s string, sep byte) []string {
	l := 0
	buf := make([]string, 0, 32)
	for {
		pos := strings.IndexByte(s[l:], sep)

		if pos == -1 {
			break
		}

		absolutePos := l + pos
		if absolutePos > l {
			buf = append(buf, s[l:absolutePos])
		}
		l = absolutePos + 1
	}

	// handle the last part separately
	if l < len(s) {
		buf = append(buf, s[l:])
	}

	return buf
}

// Represents a single rule in a .gitignore file
// Components is a list of path components to match against
// Negate is true if the rule negates the match (i.e. starts with '!')
// OnlyDirectory is true if the rule matches only directories (i.e. ends with '/')
// Relative is true if the rule is relative (i.e. starts with '/')
type Rule struct {
	Components    []string
	Negate        bool
	OnlyDirectory bool
	Relative      bool
}

func selectorMatch(c byte, selector string) bool {
	switch selector {
	case "alnum":
		return ('0' <= c && c <= '9') || ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z')
	case "alpha":
		return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z')
	case "blank":
		return c == ' ' || c == '\t'
	case "cntrl":
		return c < 32 || c == 127
	case "digit":
		return '0' <= c && c <= '9'
	case "graph":
		return 33 <= c && c <= 126
	case "lower":
		return 'a' <= c && c <= 'z'
	case "print":
		return 32 <= c && c <= 126
	case "punct":
		return (33 <= c && c <= 47) || (58 <= c && c <= 64) || (91 <= c && c <= 96) || (123 <= c && c <= 126)
	case "space":
		return (9 <= c && c <= 13) || c == 32
	case "upper":
		return 'A' <= c && c <= 'Z'
	case "xdigit":
		return ('0' <= c && c <= '9') || ('A' <= c && c <= 'F') || ('a' <= c && c <= 'f')
	default:
		return false
	}
}

func stringMatch(str string, pattern string) bool {
	// i is the index in str, j is the index in pattern
	i, j := 0, 0
	lastStarIdx := -1
	lastStrIdx := -1

	matchCharClass := func(j int, ch byte) (match bool, newJ int, ok bool) {
		j++ // skip '['
		if j >= len(pattern) {
			return false, j, false
		}
		negate := false
		matched := false
		if pattern[j] == '!' || pattern[j] == '^' {
			negate = true
			j++
			if j >= len(pattern) {
				return false, j, false
			}
		}

		// special-case leading ']'
		if pattern[j] == ']' {
			if ch == ']' {
				matched = true
			}
			j++
		}

		for j < len(pattern) && pattern[j] != ']' {
			// handle escaping
			if pattern[j] == '\\' && j+1 < len(pattern) {
				j += 2
				continue
			}
			// handle special [:class:] character classes
			if j+2 < len(pattern) && pattern[j] == '[' && pattern[j+1] == ':' {
				j += 2
				s := j
				for s < len(pattern) && (pattern[s] != ']' || pattern[s-1] != ':') {
					s++
				}

				// unclosed character class
				if s >= len(pattern) || s < j+2 {
					return false, j, false
				}

				selector := pattern[j : s-1]
				if selectorMatch(ch, selector) {
					matched = true
				}
				j = s + 1
				continue
			}
			// handle ranges
			if j+2 < len(pattern) && pattern[j+1] == '-' && pattern[j+2] != ']' {
				a := pattern[j]
				b := pattern[j+2]
				if a <= ch && ch <= b {
					matched = true
				}
				j += 3
				continue
			}
			if pattern[j] == ch {
				matched = true
			}
			j++
		}

		if j >= len(pattern) || pattern[j] != ']' {
			// unclosed character class
			return false, j, false
		}

		j++ // skip closing ']'
		if negate {
			return !matched, j, true
		}
		return matched, j, true
	}

	for i < len(str) {
		if j < len(pattern) {
			pChar := pattern[j]
			if pChar == '?' {
				i++
				j++
				continue
			}
			if pChar == '*' {
				// record star position and advance pattern
				lastStarIdx = j
				lastStrIdx = i
				j++
				continue
			}
			if pChar == '[' {
				okMatch, newJ, ok := matchCharClass(j, str[i])
				if !ok {
					// unclosed class -> no match
					return false
				}
				if okMatch {
					i++
					j = newJ
					continue
				}
				// class did not match, go to star-backtrack logic
			} else {
				// handle escaping
				if pChar == '\\' && j+1 < len(pattern) {
					j++
					pChar = pattern[j]
				}
				if str[i] == pChar {
					i++
					j++
					continue
				}
			}
		}

		if lastStarIdx != -1 {
			j = lastStarIdx + 1
			lastStrIdx++
			i = lastStrIdx
			continue
		}

		// we can't backtrack, so no match
		return false
	}

	// consume remaining stars in pattern
	for j < len(pattern) && pattern[j] == '*' {
		j++
	}

	// if we ran out of pattern, return true
	return j >= len(pattern)
}

// Tries to match the path components against the rule components
// matches is true if the path matches the rule, final is true if the rule matched the whole path
// the final parameter is used for rules that match directories only
func matchComponents(path []string, components []string) (matches bool, final bool) {
	i := 0
	for ; i < len(components); i++ {
		if i >= len(path) {
			// we ran out of path components, but still have components to match
			return false, false
		}
		if components[i] == "**" {
			// stinky recursive step
			for j := len(path) - 1; j >= i; j-- {
				match, final := matchComponents(path[j:], components[i+1:])
				if match {
					// pass final trough
					return true, final
				}
			}
			return false, false
		}

		if !stringMatch(path[i], components[i]) {
			return false, false
		}
	}
	return true, i == len(path) // if we matched all components, check if we are at the end of the path
}

// Tries to match the path against the rule
// the function expects a buffer of sufficient size to get passed to it, this avoids excessive memory allocation
func (r *Rule) matchesPath(isDirectory bool, pathComponents []string) bool {
	if !r.Relative {
		// stinky recursive step
		for j := 0; j < len(pathComponents); j++ {
			match, final := matchComponents(pathComponents[j:], r.Components)
			if match {
				return !r.OnlyDirectory || r.OnlyDirectory && (!final || final && isDirectory)
			}
		}

		return false
	}

	match, final := matchComponents(pathComponents, r.Components)

	return match && (!r.OnlyDirectory || r.OnlyDirectory && (!final || final && isDirectory))
}

// Stores a list of rules for matching paths against .gitignore patterns
// PathComponentsBuf is a temporary buffer for mySplit calls, this avoids excessive allocation
type GitIgnore struct {
	Rules             []Rule
	pathComponentsBuf []string
}

// Creates a Gitignore from a list of patterns (lines in a .gitignore file)
func CompileIgnoreLines(patterns []string) *GitIgnore {
	gitignore := &GitIgnore{
		Rules:             make([]Rule, 0, len(patterns)),
		pathComponentsBuf: make([]string, bufferLengthForPathComponents()),
	}

	for _, pattern := range patterns {
		// skip empty lines, comments, and trailing/leading whitespace
		pattern = strings.Trim(pattern, " \t\r\n")
		if pattern == "" || pattern == "!" || pattern[0] == '#' {
			continue
		}

		rule := createRule(pattern)

		gitignore.Rules = append(gitignore.Rules, rule)
	}

	return gitignore
}

// Same as CompileIgnoreLines, but reads from a file
func CompileIgnoreFile(filename string) (*GitIgnore, error) {
	lines, err := os.ReadFile(filename)

	if err != nil {
		return nil, err
	}
	return CompileIgnoreLines(strings.Split(string(lines), "\n")), nil
}

// create a rule from a pattern
func createRule(pattern string) Rule {
	negate := false
	onlyDirectory := false
	relative := false
	if pattern[0] == '!' {
		negate = true
		pattern = pattern[1:] // skip the '!'
	}

	if pattern[0] == '/' {
		relative = true
		pattern = pattern[1:] // skip the '/'
	}

	// check if the pattern ends with a '/', which means it only matches directories
	if len(pattern) > 0 && pattern[len(pattern)-1] == '/' {
		onlyDirectory = true
	}

	// split the pattern into components
	// we use the default split function because this only runs once for each rule
	// this saves memory compared to using mySplit
	components := mySplit(pattern, '/')

	return Rule{
		Components:    components,
		Negate:        negate,
		OnlyDirectory: onlyDirectory,
		Relative:      relative || len(components) > 1,
	}
}

func (g *GitIgnore) matchesPathNoError(path string) bool {
	result, _ := g.MatchesPath(path)
	return result
}

// Tries to match the path to all the rules in the gitignore
// Returns an error if the path is longer than 4096 bytes.
func (g *GitIgnore) MatchesPath(path string) (bool, error) {
	// Guard against out-of-bounds panic in mySplitBuf()
	if len(path) > maxPathLength() {
		return false, errors.New("path cannot be longer than " + strconv.Itoa(maxPathLength()) + " bytes")
	}

	// TODO: check if path actually points to a directory on the filesystem
	isDir := strings.HasSuffix(path, "/")
	path = filepath.Clean(path)
	path = filepath.ToSlash(path)
	if path == "." {
		path = "/"
		isDir = true
	}
	if path == "*" {
		return false, nil
	}
	if !fs.ValidPath(path) {
		return false, nil
	}
	pathComponents := mySplitBuf(path, '/', g.pathComponentsBuf)
	matched := false

	for _, rule := range g.Rules {
		if rule.matchesPath(isDir, pathComponents) {
			if !rule.Negate {
				matched = true
			} else {
				matched = false
			}
		}
	}
	return matched, nil
}
