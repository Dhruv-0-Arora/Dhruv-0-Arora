package face

import "testing"

func TestStripInlineGap(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"   ##face##", "   ##face##"}, // leading indent untouched
		{"   ##face##      info text", "   ##face##"},
		{"##face## info", "##face## info"}, // single space gap, keep
		{"##a##  ##b##", "##a##  ##b##"},   // 2 space gap, keep
		{"##a##    ##b##", "##a##"},        // 4+ space gap, cut
		{"        ", "        "},           // all whitespace
		{"", ""},
	}
	for _, tc := range cases {
		if got := stripInlineGap(tc.in); got != tc.want {
			t.Errorf("stripInlineGap(%q) = %q want %q", tc.in, got, tc.want)
		}
	}
}
