// build-templates regenerates dark_mode.svg and light_mode.svg from
// ASCII_art.txt + the static profile. Run this whenever you change
// the face or any non-API info.
//
//	go run ./cmd/build-templates
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Dhruv-0-Arora/Dhruv-0-Arora/internal/face"
	"github.com/Dhruv-0-Arora/Dhruv-0-Arora/internal/profile"
	"github.com/Dhruv-0-Arora/Dhruv-0-Arora/internal/svg"
)

func main() {
	asciiPath := flag.String("ascii", "ASCII_art.txt", "path to ASCII art source")
	maxCols := flag.Int("max-cols", 64, "hard cap on face row width")
	darkOut := flag.String("dark", "dark_mode.svg", "dark template output path")
	lightOut := flag.String("light", "light_mode.svg", "light template output path")
	flag.Parse()

	rows, err := face.Load(*asciiPath, *maxCols)
	if err != nil {
		fail("load face: %v", err)
	}
	if len(rows) == 0 {
		fail("no face rows found in %s", *asciiPath)
	}

	p := profile.Me()
	if err := svg.BuildTemplate(*darkOut, svg.Dark, rows, p); err != nil {
		fail("dark template: %v", err)
	}
	if err := svg.BuildTemplate(*lightOut, svg.Light, rows, p); err != nil {
		fail("light template: %v", err)
	}
	fmt.Printf("wrote %s and %s (%d face rows)\n", *darkOut, *lightOut, len(rows))
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "build-templates: "+format+"\n", args...)
	os.Exit(1)
}
