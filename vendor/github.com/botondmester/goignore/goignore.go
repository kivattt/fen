package goignore

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

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

type ruleInstructionType byte

const (
	raw ruleInstructionType = iota
	star
	starStar
	questionmark
	charClass
)

// Represents a single instruction in a rule component
type ruleInstruction struct {
	Type    ruleInstructionType
	Pattern string
}

// Represents a single component of a rule (a rule is a series of components separated by '/')
type ruleComponent struct {
	Instructions []ruleInstruction
	Starstar     bool
	Star         bool
}

// Represents a single rule in a .gitignore file
// Components is a list of path components to match against
// Negate is true if the rule negates the match (i.e. starts with '!')
// OnlyDirectory is true if the rule matches only directories (i.e. ends with '/')
// Relative is true if the rule is relative (i.e. starts with '/')
type rule struct {
	Components    []ruleComponent
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

func makeRuleComponent(component string) (ruleComponent, error) {
	instructions := make([]ruleInstruction, 0, 8)
	r := 0

	if component == "*" {
		instructions = append(instructions, ruleInstruction{
			Type: star,
		})
		return ruleComponent{
			Instructions: instructions,
			Starstar:     false,
			Star:         true,
		}, nil
	}
	if component == "**" {
		instructions = append(instructions, ruleInstruction{
			Type: starStar,
		})
		return ruleComponent{
			Instructions: instructions,
			Starstar:     true,
			Star:         false,
		}, nil
	}

	for r < len(component) {
		switch component[r] {
		case '*':
			r++
			instructions = append(instructions, ruleInstruction{
				Type: star,
			})
			continue
		case '?':
			r++
			instructions = append(instructions, ruleInstruction{
				Type: questionmark,
			})
			continue
		case '[':
			r++
			var bitset [32]byte

			if r >= len(component) {
				return ruleComponent{}, errors.New("unclosed character class")
			}

			negate := false

			if component[r] == '!' || component[r] == '^' {
				negate = true
				r++
				if r >= len(component) {
					return ruleComponent{}, errors.New("unclosed character class")
				}
			}

			// special-case leading ']'
			if component[r] == ']' {
				bitset[']'/8] |= (1 << (']' % 8))
				r++
			}

			for r < len(component) && component[r] != ']' {
				// handle escaping
				if component[r] == '\\' && r+1 < len(component) {
					r += 2
					continue
				}
				// handle special [:class:] character classes
				if r+2 < len(component) && component[r] == '[' && component[r+1] == ':' {
					r += 2
					s := r
					for s < len(component) && (component[s] != ']' || component[s-1] != ':') {
						s++
					}

					if s >= len(component) || s < r+2 {
						return ruleComponent{}, errors.New("unclosed character class")
					}

					selector := component[r : s-1]

					for i := 0; i < 256; i++ {
						if selectorMatch(byte(i), selector) {
							bitset[i/8] |= (1 << (uint(i) % 8))
						}
					}

					r = s + 1
					continue
				}
				// handle ranges
				if r+2 < len(component) && component[r+1] == '-' && component[r+2] != ']' {
					a := component[r]
					b := component[r+2]
					if a <= b {
						for i := a; i < b; i++ {
							bitset[i/8] |= (1 << (uint(i) % 8))
						}
						bitset[b/8] |= (1 << (uint(b) % 8))
					}
					r += 3
					continue
				}
				// add to LUT
				bitset[component[r]/8] |= (1 << (component[r] % 8))
				r++
			}

			if r >= len(component) || component[r] != ']' {
				return ruleComponent{}, errors.New("unclosed character class")
			}

			r++ // skip closing ']'

			if negate {
				for i := 0; i < len(bitset); i++ {
					bitset[i] = ^bitset[i]
				}
			}

			instructions = append(instructions, ruleInstruction{
				Type:    charClass,
				Pattern: string(bitset[:]),
			})
			continue
		}

		patternBuilder := strings.Builder{}

		for r < len(component) && component[r] != '*' && component[r] != '?' && component[r] != '[' {
			if component[r] == '\\' && r+1 < len(component) {
				patternBuilder.WriteByte(component[r+1])
				r += 2
				continue
			}
			patternBuilder.WriteByte(component[r])
			r++
		}

		instructions = append(instructions, ruleInstruction{
			Type:    raw,
			Pattern: patternBuilder.String(),
		})
	}

	return ruleComponent{
		Instructions: instructions,
		Starstar:     false,
		Star:         false,
	}, nil
}

func matchComponent(str string, component ruleComponent) bool {
	// i is the index in str, j is the index in pattern
	i, j := 0, 0
	lastStarIdx := -1
	lastStrIdx := -1
	strLen := len(str)
	instrLen := len(component.Instructions)

	for i < strLen {
		if j < instrLen {
			instruction := component.Instructions[j]
			switch instruction.Type {
			case questionmark:
				i++
				j++
				continue
			case star:
				lastStarIdx = j
				lastStrIdx = i
				j++
				continue
			case starStar:
				return true
			case charClass:
				char := str[i]
				if (instruction.Pattern[char/8] & (1 << (char % 8))) != 0 {
					i++
					j++
					continue
				}
			case raw:
				patLen := len(instruction.Pattern)
				if i+patLen > strLen {
					break
				}
				if str[i] != instruction.Pattern[0] {
					break
				}
				if str[i:i+patLen] == instruction.Pattern {
					i += patLen
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

	// consume remaining stars in component
	for j < instrLen && component.Instructions[j].Type == star {
		j++
	}

	// if we ran out of instructions, return true
	return j >= instrLen
}

// Tries to match the path components against the rule components
// matches is true if the path matches the rule, final is true if the rule matched the whole path
// the final parameter is used for rules that match directories only
func matchAllComponents(path []string, components []ruleComponent) (matches bool, final bool) {
	i := 0
	for ; i < len(components); i++ {
		if i >= len(path) {
			// we ran out of path components, but still have components to match
			return false, false
		}
		if components[i].Starstar {
			// stinky recursive step
			for j := len(path) - 1; j >= i; j-- {
				match, final := matchAllComponents(path[j:], components[i+1:])
				if match {
					// pass final trough
					return true, final
				}
			}
			return false, false
		}

		if !matchComponent(path[i], components[i]) {
			return false, false
		}
	}
	return true, i == len(path) // if we matched all components, check if we are at the end of the path
}

// Tries to match the path against the rule
// the function expects a buffer of sufficient size to get passed to it, this avoids excessive memory allocation
func (r *rule) matchesPath(isDirectory bool, pathComponents []string) bool {
	if !r.Relative {
		// stinky recursive step
		for j := 0; j < len(pathComponents); j++ {
			match, final := matchAllComponents(pathComponents[j:], r.Components)
			if match {
				return !final || !r.OnlyDirectory || isDirectory
			}
		}

		return false
	}

	match, final := matchAllComponents(pathComponents, r.Components)

	return match && (!final || !r.OnlyDirectory || isDirectory)
}

// Stores a list of rules for matching paths against .gitignore patterns
type GitIgnore struct {
	rules []rule
}

func trimUnescapedTrailingSpaces(s string) string {
	var i int
	for i = len(s) - 1; i >= 0; i-- {
		if s[i] != ' ' {
			break
		}
	}

	// Required to prevent indexing into index -1
	if i < 0 {
		return ""
	}

	if s[i] == '\\' {
		if i < len(s)-1 {
			i++
		}
	}

	return s[:i+1]
}

// Creates a Gitignore from a list of patterns (lines in a .gitignore file)
func CompileIgnoreLines(patterns ...string) *GitIgnore {
	gitignore := &GitIgnore{
		rules: make([]rule, 0, len(patterns)),
	}

	for _, pattern := range patterns {
		// skip empty lines, comments, '!', '/', and trailing spaces which aren't escaped with a backslash like "\ ".
		pattern = beforeFirstNullByte(pattern) // Remove anything after and including the first null-byte
		pattern = strings.TrimRight(pattern, "\r\n")
		pattern = trimUnescapedTrailingSpaces(pattern)
		if pattern == "" || pattern == "!" || pattern == "/" || pattern[0] == '#' {
			continue
		}

		rule := createRule(pattern)

		gitignore.rules = append(gitignore.rules, rule)
	}

	return gitignore
}

// Same as CompileIgnoreLines, but reads from a file
func CompileIgnoreFile(filename string) (*GitIgnore, error) {
	lines, err := os.ReadFile(filename)

	if err != nil {
		return nil, err
	}
	return CompileIgnoreLines(strings.Split(string(lines), "\n")...), nil
}

// create a rule from a pattern
func createRule(pattern string) rule {
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
	components := mySplit(pattern, '/')

	ruleComponents := make([]ruleComponent, len(components))

	for i := 0; i < len(components); i++ {
		comp, err := makeRuleComponent(components[i])
		if err == nil {
			ruleComponents[i] = comp
		}
	}

	return rule{
		Components:    ruleComponents,
		Negate:        negate,
		OnlyDirectory: onlyDirectory,
		Relative:      relative || len(components) > 1,
	}
}

// Copied from: https://cs.opensource.google/go/go/+/refs/tags/go1.26.0:src/io/fs/fs.go;l=64
// Modified to allow invalid UTF-8
func validPathBadUtf8Allowed(name string) bool {
	/*if !utf8.ValidString(name) {
		return false
	}*/

	if name == "." {
		// special case
		return true
	}

	// Iterate over elements in name, checking each.
	for {
		i := 0
		for i < len(name) && name[i] != '/' {
			i++
		}
		elem := name[:i]
		if elem == "" || elem == "." || elem == ".." {
			return false
		}
		if i == len(name) {
			return true // reached clean ending
		}
		name = name[i+1:]
	}
}

func beforeFirstNullByte(s string) string {
	firstNullByte := strings.IndexByte(s, '\x00')
	if firstNullByte == -1 {
		return s
	}

	return s[:firstNullByte]
}

// Tries to match the path to all the rules in the gitignore
func (g *GitIgnore) MatchesPath(path string) bool {
	if strings.IndexByte(path, '\x00') != -1 {
		return false
	}

	// TODO: check if path actually points to a directory on the filesystem
	isDir := strings.HasSuffix(path, "/")
	path = filepath.Clean(path) // Removes trailing slashes, except for roots like "/", "C:\"
	path = filepath.ToSlash(path)
	if path == "." {
		path = "/"
		isDir = true
	}
	if !validPathBadUtf8Allowed(path) {
		return false
	}
	pathComponents := mySplit(path, '/')

	// First, if there are any parent directories (more than 1 path component), check if they match.
	for j := 0; j < len(pathComponents)-1; j++ {
		for i := len(g.rules) - 1; i >= 0; i-- {
			rule := g.rules[i]
			if rule.matchesPath(true /* Makes no difference? */, pathComponents[:j+1]) {
				if rule.Negate {
					break // Undecided.
				} else {
					return true
				}
			}
		}
	}

	// If no parent directories match, we must check if the whole path matches.
	for i := len(g.rules) - 1; i >= 0; i-- {
		rule := g.rules[i]

		if rule.matchesPath(isDir, pathComponents) {
			if rule.Negate {
				return false
			} else {
				return true
			}
		}
	}

	return false
}
