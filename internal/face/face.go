// Package face loads the ASCII portrait from ASCII_art.txt.
//
// The source file may have inline annotations (info text on the right
// side of certain rows). Trim() strips those by clipping every row to
// `maxCols` columns, so only the face survives.
package face

import (
	"bufio"
	"os"
	"strings"
)

// Load reads the ASCII portrait at path and returns one string per
// face row.
//
// `maxCols` is a hard upper-bound clip in case of stray content.
//
// In addition to the hard clip, each row is scanned for a "gap" — a
// run of >=4 internal spaces that follows at least one face character.
// Such a gap marks the boundary between the face and any inline
// annotation (the original ASCII file embeds a small info panel on a
// few rows). Everything from the gap onward is dropped.
//
// Trailing whitespace is stripped; fully-blank leading/trailing rows
// are dropped.
func Load(path string, maxCols int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var rows []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		line = stripInlineGap(line)
		runes := []rune(line)
		if len(runes) > maxCols {
			runes = runes[:maxCols]
		}
		rows = append(rows, strings.TrimRightFunc(string(runes), func(r rune) bool {
			return r == ' ' || r == '\t'
		}))
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	for len(rows) > 0 && rows[0] == "" {
		rows = rows[1:]
	}
	for len(rows) > 0 && rows[len(rows)-1] == "" {
		rows = rows[:len(rows)-1]
	}
	return rows, nil
}

// stripInlineGap returns the prefix of s up to (but not including) the
// first run of >=4 spaces that occurs after the first non-space rune.
// Leading indentation is preserved.
func stripInlineGap(s string) string {
	seenChar := false
	spaceRun := 0
	for i, r := range s {
		if r == ' ' {
			if seenChar {
				spaceRun++
				if spaceRun >= 4 {
					return s[:i-spaceRun+1] // keep one trailing space, drop the gap
				}
			}
			continue
		}
		seenChar = true
		spaceRun = 0
	}
	return s
}
