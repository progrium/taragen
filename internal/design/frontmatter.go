// Package design implements the proposal-driven design workflow described in
// docs/design/README.md: parsing proposal frontmatter, validating proposals,
// and computing their state from git.
package design

import (
	"fmt"
	"regexp"
	"strings"
)

// Field is one frontmatter line. Exactly one of Scalar or List is meaningful,
// depending on IsList.
type Field struct {
	Key    string
	Scalar string
	List   []string
	IsList bool
	Line   int // 1-based line number in the file
}

// Frontmatter is the parsed block between the opening and closing "---" lines.
type Frontmatter struct {
	Fields []*Field
	byKey  map[string]*Field
}

// Get returns the field for key, or nil.
func (fm *Frontmatter) Get(key string) *Field {
	if fm == nil {
		return nil
	}
	return fm.byKey[key]
}

// FrontmatterError is a problem on one frontmatter line.
type FrontmatterError struct {
	Line int
	Key  string // the line's key, if it has a readable one
	Msg  string
}

func (e FrontmatterError) Error() string { return fmt.Sprintf("%d: %s", e.Line, e.Msg) }

var (
	keyRe       = regexp.MustCompile(`^[a-z][a-z-]*$`)
	listItemRe  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*$`)
	yamlSpecial = map[string]bool{
		"true": true, "false": true, "null": true, "~": true,
		"yes": true, "no": true, "on": true, "off": true, "y": true, "n": true,
	}
)

// yamlIndicators are characters that change YAML's reading of a plain scalar
// when they start it.
const yamlIndicators = "-?:,[]{}#&*!|>'\"%@`"

// ParseFrontmatter splits src into frontmatter and body. The frontmatter is a
// strict subset of YAML: one "key: value" per line, scalars or flow lists, no
// nesting, quoting, comments, or multi-line values. Anything a full YAML
// parser might read differently is rejected, so every language reads it the
// same way.
//
// bodyLine is the 1-based line number where the body starts.
func ParseFrontmatter(src string) (fm *Frontmatter, body string, bodyLine int, errs []FrontmatterError) {
	lines := strings.Split(src, "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return nil, src, 1, []FrontmatterError{{Line: 1, Msg: `missing frontmatter: file must start with a "---" line`}}
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return nil, src, 1, []FrontmatterError{{Line: 1, Msg: `unterminated frontmatter: no closing "---" line`}}
	}

	fm = &Frontmatter{byKey: map[string]*Field{}}
	for i := 1; i < end; i++ {
		lineNo := i + 1
		f, err := parseLine(lines[i], lineNo)
		if err != "" {
			key, _, _ := strings.Cut(lines[i], ":")
			errs = append(errs, FrontmatterError{lineNo, key, err})
			continue
		}
		if fm.byKey[f.Key] != nil {
			errs = append(errs, FrontmatterError{lineNo, f.Key, fmt.Sprintf("duplicate key %q", f.Key)})
			continue
		}
		fm.Fields = append(fm.Fields, f)
		fm.byKey[f.Key] = f
	}
	body = strings.Join(lines[end+1:], "\n")
	return fm, body, end + 2, errs
}

func parseLine(line string, lineNo int) (*Field, string) {
	if strings.TrimSpace(line) == "" {
		return nil, "blank line in frontmatter"
	}
	if strings.ContainsAny(line, "\t\r") {
		return nil, "tab or carriage return in frontmatter"
	}
	if strings.HasPrefix(line, " ") {
		return nil, "indented line in frontmatter (nesting is not allowed)"
	}
	var key, val string
	if k, ok := strings.CutSuffix(line, ":"); ok && !strings.Contains(k, ": ") {
		key = k
	} else {
		k, v, found := strings.Cut(line, ": ")
		if !found {
			return nil, fmt.Sprintf("expected \"key: value\", got %q", line)
		}
		key, val = k, v
	}
	if !keyRe.MatchString(key) {
		return nil, fmt.Sprintf("invalid key %q", key)
	}
	f := &Field{Key: key, Line: lineNo}
	if val != strings.TrimSpace(val) {
		return nil, fmt.Sprintf("%s: leading or trailing spaces in value", key)
	}
	if strings.HasPrefix(val, "[") {
		if !strings.HasSuffix(val, "]") {
			return nil, fmt.Sprintf("%s: unterminated list", key)
		}
		f.IsList = true
		inner := val[1 : len(val)-1]
		if inner == "" {
			return f, ""
		}
		for _, item := range strings.Split(inner, ", ") {
			if !listItemRe.MatchString(item) {
				return nil, fmt.Sprintf("%s: invalid list item %q (lists are written [a, b])", key, item)
			}
			f.List = append(f.List, item)
		}
		return f, ""
	}
	if val != "" {
		if strings.ContainsAny(val[:1], yamlIndicators) {
			return nil, fmt.Sprintf("%s: value starts with YAML indicator %q", key, val[:1])
		}
		if strings.Contains(val, ": ") || strings.HasSuffix(val, ":") {
			return nil, fmt.Sprintf("%s: value contains \": \"", key)
		}
		if strings.Contains(val, " #") {
			return nil, fmt.Sprintf("%s: value contains \" #\"", key)
		}
		if yamlSpecial[strings.ToLower(val)] {
			return nil, fmt.Sprintf("%s: value %q would be read as a YAML boolean or null", key, val)
		}
	}
	f.Scalar = val
	return f, ""
}

// FormatList writes items as a flow list: [a, b].
func FormatList(items []string) string {
	return "[" + strings.Join(items, ", ") + "]"
}
