package common

import (
	"time"
)

const (
	//Year parse mask
	Year = "2006"
	//Month parse mask
	Month = "01"
	//Day parse mask
	Day = "02"
	//Hour parse mask
	Hour = "15"
	//Minute parse mask
	Minute = "04"
	//Second parse mask
	Second = "05"
	//Milli parse mask
	Msec = ".000"
	//DateSeparator parse mask
	DateSeparator = "."
	//TimeSeparator parse mask
	TimeSeparator = ":"
	//Separator parse mask
	Separator = " "
)

type asDateTime interface {
	Year() int
	Month() time.Month
	Day() int
	Hour() int
	Minute() int
	Second() int
}

func ToTime(v asDateTime) time.Time {
	return time.Date(v.Year(), v.Month(), v.Day(), 0, 0, 0, 0, time.Local)
}

var DateMask = Day + DateSeparator + Month + DateSeparator + Year
var TimeMask = Hour + TimeSeparator + Minute + TimeSeparator + Second

var DateTimeMask = DateMask + Separator + TimeMask
var DateTimeMilliMask = DateMask + Separator + TimeMask + Msec

var SortedDateMask = Year + DateSeparator + Month + DateSeparator + Day
var SortedDateTimeMilliMask = SortedDateMask + Separator + TimeMask + Msec

// ParseDateTime parses only date, but no time
func ParseDateTime(mask string, v string) (time.Time, error) {
	l, err := time.LoadLocation("Local")
	if Error(err) {
		return time.Time{}, err
	}

	return time.ParseInLocation(string(mask), v, l)
}

// ParseJsonDateTime parses only date, but no time
func ParseJsonDateTime(v string) (time.Time, error) {
	l, err := time.Parse(time.RFC3339, v)
	if Error(err) {
		return time.Time{}, err
	}

	return l, nil
}

// EqualDateTime checks for equality of parts
func CompareDate(t1 time.Time, t2 time.Time) time.Duration {
	return ClearTime(t1).Sub(ClearTime(t2))
}

// EqualTime checks for equality of time
func CompareTime(t1 time.Time, t2 time.Time) time.Duration {
	return ClearDate(t1).Sub(ClearDate(t2))
}

// ClearTime returns only date part, time part set to 0
func ClearTime(v time.Time) time.Time {
	return time.Date(v.Year(), v.Month(), v.Day(), 0, 0, 0, 0, v.Location())
}

// ClearDate returns only time part, date part set to 0
func ClearDate(v time.Time) time.Time {
	return time.Date(0, 0, 0, v.Hour(), v.Minute(), v.Second(), v.Nanosecond(), v.Location())
}

func SetTime(t time.Time, f string, v int) time.Time {
	y := t.Year()
	m := t.Month()
	d := t.Day()
	h := t.Hour()
	mi := t.Minute()
	s := t.Second()
	n := t.Nanosecond()

	if f == Year {
		y = v
	}
	if f == Month {
		m = time.Month(v)
	}
	if f == Day {
		d = v
	}
	if f == Hour {
		h = v
	}
	if f == Minute {
		mi = v
	}
	if f == Second {
		s = v
	}
	if f == Msec {
		n = v * 1000
	}

	return time.Date(y, m, d, h, mi, s, n, time.Local)
}

func TruncateTime(t time.Time, f string) time.Time {
	y := t.Year()
	m := t.Month()
	d := t.Day()
	h := t.Hour()
	mi := t.Minute()
	s := t.Second()
	n := t.Nanosecond()

	if f == Year {
		m = time.January
		d = 0
		h = 0
		mi = 0
		s = 0
		n = 0
	}
	if f == Month {
		d = 0
		h = 0
		mi = 0
		s = 0
		n = 0
	}
	if f == Day {
		h = 0
		mi = 0
		s = 0
		n = 0
	}
	if f == Hour {
		mi = 0
		s = 0
		n = 0
	}
	if f == Minute {
		s = 0
		n = 0
	}
	if f == Second {
		n = 0
	}

	return time.Date(y, m, d, h, mi, s, n, time.Local)
}

func MillisecondToDuration(msec int) time.Duration {
	return time.Millisecond * time.Duration(msec)
}

func DurationToMillisecond(d time.Duration) int {
	return int(d.Milliseconds())
}
func CalcDeadline(t time.Time, d time.Duration) time.Time {
	if d == 0 {
		return time.Time{}

	}

	return t.Add(d)
}
