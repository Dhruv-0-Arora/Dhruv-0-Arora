// Package profile holds the static personal info baked into the SVGs.
//
// Edit Me() to change anything that doesn't come from the GitHub API.
package profile

import (
	"time"

	"github.com/Dhruv-0-Arora/Dhruv-0-Arora/internal/svg"
)

// Birthday is used for the "Uptime" line (years, months, days alive).
// Set to your real DOB at midnight UTC.
var Birthday = time.Date(2003, time.January, 1, 0, 0, 0, 0, time.UTC)

// Login is the GitHub username used for all queries.
const Login = "Dhruv-0-Arora"

// Me returns the static info panel. Tweak freely — none of these
// values come from the API.
func Me() svg.Profile {
	return svg.Profile{
		Header:        "darora1@arch",
		OS:            "MacOS, Arch Linux + i3, iOS",
		Host:          "Autodesk",
		Kernel:        "UW-Madison Computer Science & Math",
		IDE:           "Neovim",
		LangsProgram:  "Go, Rust, Python, C++, TypeScript",
		LangsMarkup:   "HTML, CSS, JSON, LaTeX, YAML",
		LangsReal:     "English, Hindi",
		HobbiesSW:     "Systems programming, dotfiles tinkering",
		HobbiesHW:     "Mechanical keyboards, custom PC builds",
		EmailPersonal: "you@example.com",
		LinkedIn:      "Dhruv-0-Arora",
		GitHub:        "Dhruv-0-Arora",
	}
}
