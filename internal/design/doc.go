package design

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Doc is the Markdown structure of a proposal body or the spec.
type Doc struct {
	Title         string // text after "# "
	TitleLine     int
	Signature     string // the line under the title, if it looks like a signature
	SignatureLine int
	Sections      []*Section
}

// Section is a "## " heading and the lines under it.
type Section struct {
	Heading string
	Line    int
	Lines   []DocLine
}

// DocLine is one body line with its 1-based line number in the file.
type DocLine struct {
	N    int
	Text string
}

// Section returns the first section with the given heading, ignoring case.
func (d *Doc) Section(heading string) *Section {
	for _, s := range d.Sections {
		if strings.EqualFold(s.Heading, heading) {
			return s
		}
	}
	return nil
}

// ParseDoc parses body, whose first line is line startLine of its file.
func ParseDoc(body string, startLine int) *Doc {
	d := &Doc{}
	lines := strings.Split(body, "\n")
	var cur *Section
	inFence := false
	for i, text := range lines {
		n := startLine + i
		if strings.HasPrefix(text, "```") || strings.HasPrefix(text, "~~~") {
			inFence = !inFence
		}
		if !inFence {
			if t, ok := strings.CutPrefix(text, "# "); ok && d.TitleLine == 0 {
				d.Title, d.TitleLine = strings.TrimSpace(t), n
				// the signature, if any, is the first non-blank line after the title
				for j := i + 1; j < len(lines); j++ {
					if strings.TrimSpace(lines[j]) == "" {
						continue
					}
					if strings.Contains(lines[j], "·") && !strings.HasPrefix(lines[j], "#") {
						d.Signature, d.SignatureLine = lines[j], startLine+j
					}
					break
				}
				continue
			}
			if h, ok := strings.CutPrefix(text, "## "); ok {
				cur = &Section{Heading: strings.TrimSpace(h), Line: n}
				d.Sections = append(d.Sections, cur)
				continue
			}
		}
		if cur != nil {
			cur.Lines = append(cur.Lines, DocLine{n, text})
		}
	}
	return d
}

// DoneItem is one top-level item of a Done criteria section.
type DoneItem struct {
	Line int
	Text string
	Mark string // " ", "x", or "~"; empty if the item isn't a checkbox
}

var (
	topItemRe  = regexp.MustCompile(`^([-*+]|\d+[.)]) `)
	checkboxRe = regexp.MustCompile(`^- \[([ x~])\] \S`)
)

// DoneItems returns the top-level list items of the Done criteria section,
// and whether the section exists.
func (d *Doc) DoneItems() ([]DoneItem, bool) {
	s := d.Section("Done criteria")
	if s == nil {
		return nil, false
	}
	var items []DoneItem
	inFence := false
	for _, l := range s.Lines {
		if strings.HasPrefix(l.Text, "```") || strings.HasPrefix(l.Text, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence || !topItemRe.MatchString(l.Text) {
			continue
		}
		it := DoneItem{Line: l.N, Text: l.Text}
		if m := checkboxRe.FindStringSubmatch(l.Text); m != nil {
			it.Mark = m[1]
		}
		items = append(items, it)
	}
	return items, true
}

// SignatureSuffix marks AI-assisted work substantially directed by the
// named person, who takes responsibility for it.
const SignatureSuffix = ", h.d.a."

const sigDateLayout = "Jan 2, 2006"

var sigRe = regexp.MustCompile(`^([A-Z][a-z]{2} \d{1,2}, \d{4}) · (\S(?:.*\S)?)$`)

// ParsedSignature is a signature line split into its parts.
type ParsedSignature struct {
	Date time.Time
	Name string
	HDA  bool
}

// ParseSignature parses a signature line in one of the two accepted forms:
// "Nov 11, 2026 · Full Name, h.d.a." or "Nov 11, 2026 · Full Name".
func ParseSignature(line string) (ParsedSignature, error) {
	m := sigRe.FindStringSubmatch(line)
	if m == nil {
		return ParsedSignature{}, fmt.Errorf("signature %q is not in the form \"Nov 11, 2026 · Full Name, h.d.a.\"", line)
	}
	t, err := time.Parse(sigDateLayout, m[1])
	if err != nil || t.Format(sigDateLayout) != m[1] {
		return ParsedSignature{}, fmt.Errorf("signature %q has an invalid date", line)
	}
	name, hda := strings.CutSuffix(m[2], SignatureSuffix)
	if name == "" || strings.Contains(name, "·") {
		return ParsedSignature{}, fmt.Errorf("signature %q has no name", line)
	}
	return ParsedSignature{Date: t, Name: name, HDA: hda}, nil
}

// FormatSignature writes a signature line.
func FormatSignature(date time.Time, name string, hda bool) string {
	s := date.Format(sigDateLayout) + " · " + name
	if hda {
		s += SignatureSuffix
	}
	return s
}

// ApplySignature puts sig on the line under the document's title in src,
// replacing an existing signature there. doc must have been parsed from src,
// since its line numbers are used to find the title.
func ApplySignature(src string, doc *Doc, sig string) (string, error) {
	if doc.TitleLine == 0 {
		return "", fmt.Errorf("document has no \"# \" title line")
	}
	lines := strings.Split(src, "\n")
	if doc.SignatureLine != 0 {
		lines[doc.SignatureLine-1] = sig
	} else {
		ti := doc.TitleLine - 1
		rest := append([]string{"", sig}, lines[ti+1:]...)
		lines = append(lines[:ti+1], rest...)
		// keep exactly one blank line between the signature and what follows
		if len(lines) > ti+3 && strings.TrimSpace(lines[ti+3]) != "" {
			lines = append(lines[:ti+3], append([]string{""}, lines[ti+3:]...)...)
		} else if len(lines) > ti+4 && lines[ti+3] == "" && lines[ti+4] == "" {
			lines = append(lines[:ti+3], lines[ti+4:]...)
		}
	}
	return strings.Join(lines, "\n"), nil
}
