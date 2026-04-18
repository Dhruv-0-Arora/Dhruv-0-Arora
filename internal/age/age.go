// Package age computes a human-readable years/months/days difference,
// matching Python's dateutil.relativedelta semantics (forward-walk with
// day-of-month clamping).
package age

import (
	"fmt"
	"time"
)

// Diff is a calendar-aware duration broken down into years, months, days.
type Diff struct {
	Years, Months, Days int
}

// Since returns the calendar difference between birth and now.
//
// The walk is: birth + Y years (clamped) + M months (clamped) + D days = now,
// where Y and M are maximised in turn. This is the same algorithm
// dateutil.relativedelta uses for the (date1, date2) form.
func Since(birth, now time.Time) Diff {
	years := now.Year() - birth.Year()
	if addYearsClamp(birth, years).After(now) {
		years--
	}
	afterYears := addYearsClamp(birth, years)

	months := int(now.Month()) - int(afterYears.Month())
	if months < 0 {
		months += 12
	}
	if addMonthsClamp(afterYears, months).After(now) {
		months--
	}
	afterMonths := addMonthsClamp(afterYears, months)

	// Day arithmetic at midnight is exact in days because both timestamps
	// are normalised to UTC midnight by the caller.
	days := int(now.Sub(afterMonths).Hours() / 24)

	return Diff{Years: years, Months: months, Days: days}
}

// String formats the diff like "22 years, 5 months, 29 days",
// adding a cake on the user's exact birthday.
func (d Diff) String() string {
	cake := ""
	if d.Months == 0 && d.Days == 0 {
		cake = " 🎂"
	}
	return fmt.Sprintf("%d %s, %d %s, %d %s%s",
		d.Years, plural("year", d.Years),
		d.Months, plural("month", d.Months),
		d.Days, plural("day", d.Days),
		cake,
	)
}

func plural(unit string, n int) string {
	if n == 1 {
		return unit
	}
	return unit + "s"
}

// addYearsClamp adds n years, clamping Feb 29 -> Feb 28 in non-leap targets.
func addYearsClamp(t time.Time, n int) time.Time {
	return addMonthsClamp(t, n*12)
}

// addMonthsClamp adds n calendar months. If the source day-of-month doesn't
// exist in the target month (e.g. Jan 31 + 1 month), it clamps to the last
// valid day of the target month. Time-of-day and location are preserved.
func addMonthsClamp(t time.Time, n int) time.Time {
	y, m, d := t.Date()
	total := int(m) - 1 + n
	targetYear := y + floorDiv(total, 12)
	targetMonth := time.Month(modPos(total, 12) + 1)
	last := daysIn(targetYear, targetMonth)
	if d > last {
		d = last
	}
	return time.Date(targetYear, targetMonth, d, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

func daysIn(year int, month time.Month) int {
	// Day 0 of the next month == last day of `month`.
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

func modPos(a, b int) int {
	r := a % b
	if r < 0 {
		r += b
	}
	return r
}
