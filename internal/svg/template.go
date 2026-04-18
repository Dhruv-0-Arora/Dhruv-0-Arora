// Package svg builds and rewrites the README SVGs.
//
// The template generation lives here so that the ASCII face is the
// single source of truth: ASCII_art.txt -> dark_mode.svg / light_mode.svg
// (each with stable id'd <tspan>s for dynamic stats).
//
// The rewrite step (see rewrite.go) only mutates those id'd tspans and
// the leading dot-leader tspans next to them; ASCII rows are never touched.
package svg

import (
	"fmt"
	"os"
	"strings"
)

// Theme controls the color palette baked into the template.
type Theme struct {
	Name           string // "dark" or "light"
	BackgroundFill string
	TextFill       string
	KeyFill        string
	ValueFill      string
	AddFill        string
	DelFill        string
	CCFill         string // "consolas comment" / dot-leader color
}

// Dark matches Andrew6rant's dark palette (GitHub dark theme tokens).
var Dark = Theme{
	Name:           "dark",
	BackgroundFill: "#161b22",
	TextFill:       "#c9d1d9",
	KeyFill:        "#ffa657",
	ValueFill:      "#a5d6ff",
	AddFill:        "#3fb950",
	DelFill:        "#f85149",
	CCFill:         "#616e7f",
}

// Light matches the GitHub light palette.
var Light = Theme{
	Name:           "light",
	BackgroundFill: "#f6f8fa",
	TextFill:       "#24292f",
	KeyFill:        "#953800",
	ValueFill:      "#0a3069",
	AddFill:        "#1a7f37",
	DelFill:        "#cf222e",
	CCFill:         "#c2cfde",
}

// Profile is the static info baked into the right-hand panel.
// Values that change daily (age, repo counts, LOC, etc.) live in the
// rewrite step, not here.
type Profile struct {
	Header       string // e.g. "darora1@arch"
	OS           string
	Host         string
	Kernel       string
	IDE          string
	LangsProgram string
	LangsMarkup  string
	LangsReal    string
	HobbiesSW    string
	HobbiesHW    string
	EmailPersonal string
	LinkedIn     string
	GitHub       string
}

const (
	canvasW = 1000
	canvasH = 600
	asciiX  = 15
	infoX   = 540
	rowH    = 20
	firstY  = 30
)

// BuildTemplate writes a fresh SVG to outPath using the given theme,
// face (raw ASCII rows), and static profile info. Dynamic stats are
// initialised to placeholders that the rewrite step will overwrite.
func BuildTemplate(outPath string, theme Theme, faceRows []string, p Profile) error {
	var b strings.Builder

	fmt.Fprintf(&b, `<?xml version='1.0' encoding='UTF-8'?>`+"\n")
	fmt.Fprintf(&b,
		`<svg xmlns="http://www.w3.org/2000/svg" font-family="ConsolasFallback,Consolas,monospace" width="%dpx" height="%dpx" font-size="16px">`+"\n",
		canvasW, canvasH)
	b.WriteString(`<style>
@font-face {
src: local('Consolas'), local('Consolas Bold');
font-family: 'ConsolasFallback';
font-display: swap;
-webkit-size-adjust: 109%;
size-adjust: 109%;
}
`)
	fmt.Fprintf(&b, ".key {fill: %s;}\n", theme.KeyFill)
	fmt.Fprintf(&b, ".value {fill: %s;}\n", theme.ValueFill)
	fmt.Fprintf(&b, ".addColor {fill: %s;}\n", theme.AddFill)
	fmt.Fprintf(&b, ".delColor {fill: %s;}\n", theme.DelFill)
	fmt.Fprintf(&b, ".cc {fill: %s;}\n", theme.CCFill)
	b.WriteString("text, tspan {white-space: pre;}\n</style>\n")

	fmt.Fprintf(&b, `<rect width="%dpx" height="%dpx" fill="%s" rx="15"/>`+"\n",
		canvasW, canvasH, theme.BackgroundFill)

	// ASCII face block. Each row is its own tspan so positioning is exact.
	fmt.Fprintf(&b, `<text x="%d" y="%d" fill="%s" class="ascii">`+"\n", asciiX, firstY, theme.TextFill)
	for i, row := range faceRows {
		y := firstY + i*rowH
		fmt.Fprintf(&b, `<tspan x="%d" y="%d">%s</tspan>`+"\n", asciiX, y, escapeXML(row))
	}
	b.WriteString("</text>\n")

	// Right-hand info panel.
	fmt.Fprintf(&b, `<text x="%d" y="%d" fill="%s">`+"\n", infoX, firstY, theme.TextFill)

	y := firstY
	// Header
	fmt.Fprintf(&b, `<tspan x="%d" y="%d">%s</tspan> %s`+"\n",
		infoX, y, escapeXML(p.Header), strings.Repeat("-", 50))
	y += rowH

	// System block
	y = writeKV(&b, infoX, y, "OS", p.OS, "", "")
	y = writeKV(&b, infoX, y, "Uptime", "1 year, 0 months, 0 days", "age_data", "age_data_dots")
	y = writeKV(&b, infoX, y, "Host", p.Host, "", "")
	y = writeKV(&b, infoX, y, "Kernel", p.Kernel, "", "")
	y = writeKV(&b, infoX, y, "IDE", p.IDE, "", "")
	y = writeBlank(&b, infoX, y)

	// Languages
	y = writeKVPath(&b, infoX, y, "Languages", "Programming", p.LangsProgram)
	y = writeKVPath(&b, infoX, y, "Languages", "Markup", p.LangsMarkup)
	y = writeKVPath(&b, infoX, y, "Languages", "Real", p.LangsReal)
	y = writeBlank(&b, infoX, y)

	// Hobbies
	y = writeKVPath(&b, infoX, y, "Hobbies", "Software", p.HobbiesSW)
	y = writeKVPath(&b, infoX, y, "Hobbies", "Hardware", p.HobbiesHW)
	y += rowH // gap before contact

	// Contact section header
	fmt.Fprintf(&b, `<tspan x="%d" y="%d">- Contact</tspan> %s`+"\n",
		infoX, y, strings.Repeat("-", 50))
	y += rowH
	y = writeKVPath(&b, infoX, y, "Email", "Personal", p.EmailPersonal)
	y = writeKV(&b, infoX, y, "LinkedIn", p.LinkedIn, "", "")
	y = writeKV(&b, infoX, y, "GitHub", p.GitHub, "", "")
	y += rowH // gap before stats

	// GitHub Stats section header
	fmt.Fprintf(&b, `<tspan x="%d" y="%d">- GitHub Stats</tspan> %s`+"\n",
		infoX, y, strings.Repeat("-", 50))
	y += rowH

	// Repos | Contributed | Stars (multi-value row)
	fmt.Fprintf(&b,
		`<tspan x="%d" y="%d" class="cc">. </tspan>`+
			`<tspan class="key">Repos</tspan>:<tspan class="cc" id="repo_data_dots"> .... </tspan>`+
			`<tspan class="value" id="repo_data">0</tspan> `+
			`{<tspan class="key">Contributed</tspan>: <tspan class="value" id="contrib_data">0</tspan>} | `+
			`<tspan class="key">Stars</tspan>:<tspan class="cc" id="star_data_dots"> ... </tspan>`+
			`<tspan class="value" id="star_data">0</tspan>`+"\n",
		infoX, y)
	y += rowH

	// Commits | Followers
	fmt.Fprintf(&b,
		`<tspan x="%d" y="%d" class="cc">. </tspan>`+
			`<tspan class="key">Commits</tspan>:<tspan class="cc" id="commit_data_dots"> .... </tspan>`+
			`<tspan class="value" id="commit_data">0</tspan> | `+
			`<tspan class="key">Followers</tspan>:<tspan class="cc" id="follower_data_dots"> ... </tspan>`+
			`<tspan class="value" id="follower_data">0</tspan>`+"\n",
		infoX, y)
	y += rowH

	// LOC line
	fmt.Fprintf(&b,
		`<tspan x="%d" y="%d" class="cc">. </tspan>`+
			`<tspan class="key">Lines of Code on GitHub</tspan>:<tspan class="cc" id="loc_data_dots">. </tspan>`+
			`<tspan class="value" id="loc_data">0</tspan> ( `+
			`<tspan class="addColor" id="loc_add">0</tspan><tspan class="addColor">++</tspan>, `+
			`<tspan class="delColor" id="loc_del">0</tspan><tspan class="delColor">--</tspan> )`+"\n",
		infoX, y)

	b.WriteString("</text>\n")
	b.WriteString("</svg>\n")

	return os.WriteFile(outPath, []byte(b.String()), 0o644)
}

// writeKV emits a single "  Key: ..... value" row. id and dotsID are
// optional; when set, the value/dots tspans get those ids so the
// rewrite step can find and update them.
func writeKV(b *strings.Builder, x, y int, key, value, id, dotsID string) int {
	dotsAttr := ""
	if dotsID != "" {
		dotsAttr = fmt.Sprintf(` id=%q`, dotsID)
	}
	valAttr := ""
	if id != "" {
		valAttr = fmt.Sprintf(` id=%q`, id)
	}
	dots := defaultDots(key, value)
	fmt.Fprintf(b,
		`<tspan x="%d" y="%d" class="cc">. </tspan>`+
			`<tspan class="key">%s</tspan>:<tspan class="cc"%s> %s </tspan>`+
			`<tspan class="value"%s>%s</tspan>`+"\n",
		x, y, escapeXML(key), dotsAttr, dots, valAttr, escapeXML(value),
	)
	return y + rowH
}

// writeKVPath emits a "  Group.Sub: ..... value" row, no dynamic id.
func writeKVPath(b *strings.Builder, x, y int, group, sub, value string) int {
	dots := defaultDots(group+"."+sub, value)
	fmt.Fprintf(b,
		`<tspan x="%d" y="%d" class="cc">. </tspan>`+
			`<tspan class="key">%s</tspan>.<tspan class="key">%s</tspan>:<tspan class="cc"> %s </tspan>`+
			`<tspan class="value">%s</tspan>`+"\n",
		x, y, escapeXML(group), escapeXML(sub), dots, escapeXML(value),
	)
	return y + rowH
}

func writeBlank(b *strings.Builder, x, y int) int {
	fmt.Fprintf(b, `<tspan x="%d" y="%d" class="cc">. </tspan>`+"\n", x, y)
	return y + rowH
}

// defaultDots returns a dot-leader of length chosen so the row
// (key + dots + value) is roughly `targetWidth` characters wide.
// The rewrite step will recompute these for dynamic rows; this is
// only the static initial padding.
func defaultDots(key, value string) string {
	const targetWidth = 50
	used := len(key) + 2 /* ": " */ + len(value)
	pad := targetWidth - used
	if pad < 2 {
		return " "
	}
	return strings.Repeat(".", pad)
}

func escapeXML(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
	return r.Replace(s)
}
