package age

import (
	"testing"
	"time"
)

func d(y int, m time.Month, day int) time.Time {
	return time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
}

func TestSince(t *testing.T) {
	cases := []struct {
		name        string
		birth, now  time.Time
		want        Diff
		wantStr     string
	}{
		{
			name:    "exact years",
			birth:   d(2000, time.January, 1),
			now:     d(2025, time.January, 1),
			want:    Diff{25, 0, 0},
			wantStr: "25 years, 0 months, 0 days 🎂",
		},
		{
			// Jan30 + 25y = Jan30 2025; +1mo clamps to Feb28; +1d = Mar1.
			name:    "month clamp from february",
			birth:   d(2000, time.January, 30),
			now:     d(2025, time.March, 1),
			want:    Diff{25, 1, 1},
			wantStr: "25 years, 1 month, 1 day",
		},
		{
			// Matches Andrew's example: 2002-07-05 -> 2025-01-03.
			name:    "borrow days and months",
			birth:   d(2002, time.July, 5),
			now:     d(2025, time.January, 3),
			want:    Diff{22, 5, 29},
			wantStr: "22 years, 5 months, 29 days",
		},
		{
			// Leap day birthday on a non-leap year: clamp to Feb 28.
			name:    "leap day birthday clamp",
			birth:   d(2000, time.February, 29),
			now:     d(2023, time.February, 28),
			want:    Diff{23, 0, 0},
			wantStr: "23 years, 0 months, 0 days 🎂",
		},
		{
			name:    "one day singular",
			birth:   d(2000, time.January, 1),
			now:     d(2001, time.January, 2),
			want:    Diff{1, 0, 1},
			wantStr: "1 year, 0 months, 1 day",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Since(tc.birth, tc.now)
			if got != tc.want {
				t.Fatalf("Since=%+v want %+v", got, tc.want)
			}
			if s := got.String(); s != tc.wantStr {
				t.Fatalf("String=%q want %q", s, tc.wantStr)
			}
		})
	}
}
