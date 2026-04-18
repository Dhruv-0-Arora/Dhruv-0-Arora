package svg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommaInt(t *testing.T) {
	cases := map[int]string{
		0:        "0",
		7:        "7",
		1000:     "1,000",
		12345:    "12,345",
		1234567:  "1,234,567",
		-12345:   "-12,345",
	}
	for in, want := range cases {
		if got := commaInt(in); got != want {
			t.Errorf("commaInt(%d) = %q want %q", in, got, want)
		}
	}
}

func TestBuildAndRewrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dark.svg")

	face := []string{"  AAA  ", " AAAAA ", "  AAA  "}
	p := Profile{
		Header:        "test@host",
		OS:            "Linux",
		Host:          "Acme",
		Kernel:        "k",
		IDE:           "vim",
		LangsProgram:  "Go",
		LangsMarkup:   "HTML",
		LangsReal:     "English",
		HobbiesSW:     "x",
		HobbiesHW:     "y",
		EmailPersonal: "a@b.c",
		LinkedIn:      "me",
		GitHub:        "me",
	}
	if err := BuildTemplate(path, Dark, face, p); err != nil {
		t.Fatal(err)
	}

	stats := Stats{
		Age:       "23 years, 1 month, 4 days",
		Repos:     12,
		Contrib:   34,
		Stars:     56,
		Commits:   789,
		Followers: 5,
		LOCAdd:    1000,
		LOCDel:    250,
	}
	if err := Rewrite(path, stats); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc := string(got)
	for _, want := range []string{
		`id="age_data">23 years, 1 month, 4 days<`,
		`id="repo_data">12<`,
		`id="contrib_data">34<`,
		`id="star_data">56<`,
		`id="commit_data">789<`,
		`id="follower_data">5<`,
		`id="loc_add">1,000<`,
		`id="loc_del">250<`,
		`id="loc_data">750<`,
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("rewritten SVG missing %q", want)
		}
	}
	// Face should still be there, untouched.
	for _, row := range face {
		if !strings.Contains(doc, row) {
			t.Errorf("face row %q lost", row)
		}
	}
}
