package types

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"time"

	"github.com/moisespsena-go/aorm"
)

type Time struct {
	h, m, s int
}

func NewTime(h, m, s int) Time {
	return Time{h, m, s}
}

func (this Time) Time() time.Time {
	return time.Date(0, 0, 0, this.h, this.m, this.s, 0, time.UTC)
}

func (this Time) Hour() int {
	return this.h
}

func (this Time) Minute() int {
	return this.m
}

func (this Time) Second() int {
	return this.s
}

func (Time) AormDataType(aorm.Dialector) string {
	return "TIME"
}

func (this Time) Value() (driver.Value, error) {
	return this.String(), nil
}

func (this *Time) Scan(src interface{}) (err error) {
	var t time.Time
	if src != nil {
		switch st := src.(type) {
		case string:
			t, err = time.Parse("15:04:05", st)
			this.h, this.m, this.s = t.Hour(), t.Minute(), t.Second()
			return
		case []byte:
			return this.Scan(string(st))
		}
	}
	return nil
}

func (this Time) String() string {
	return fmt.Sprintf("%02d:%02d:%02d", this.h, this.m, this.s)
}

type TimeZ struct {
	h, m, s, zh, zm int
}

func NewTimeZ(h, m, s, zh, zm int) TimeZ {
	return TimeZ{h, m, s, zh, zm}
}

func (this TimeZ) Hour() int {
	return this.h
}

func (this TimeZ) Minute() int {
	return this.m
}

func (this TimeZ) Second() int {
	return this.s
}

func (this TimeZ) Zone() (h, m int) {
	return this.zh, this.zm
}

func (this TimeZ) Time() time.Time {
	loc, _ := time.Parse("-0700", fmt.Sprintf("%03d%02d", this.zh, this.zm))
	return time.Date(0, 0, 0, this.h, this.m, this.s, 0, loc.Location())
}

func (TimeZ) AormDataType(aorm.Dialector) string {
	return "TIMETZ"
}

func (this TimeZ) Value() (driver.Value, error) {
	return this.String(), nil
}

func (this *TimeZ) Scan(src interface{}) (err error) {
	var t time.Time
	if src != nil {
		switch st := src.(type) {
		case string:
			if t, err = time.Parse("15:04:05 -07:00", st); err != nil {
				if t, err = time.Parse("15:04:05 -0700", st); err != nil {
					return
				}
			}
			this.h, this.m, this.s = t.Hour(), t.Minute(), t.Second()
			this.zh, this.zm = TZOfTime(t)
			return
		case []byte:
			return this.Scan(string(st))
		}
	}
	return nil
}

func (this TimeZ) String() string {
	var signal string
	var zh = this.zh
	if zh < 0 {
		signal = "-"
		zh *= -1
	} else {
		signal = "+"
	}
	return fmt.Sprintf("%02d:%02d:%02d %s%02d:%02d", this.h, this.m, this.s, signal, zh, this.zm)
}

func ParseTZ(tzs string) (zh, zm int, err error) {
	var t time.Time
	if t, err = time.Parse("-07:00", tzs); err != nil {
		if t, err = time.Parse("-0700", tzs); err != nil {
			return
		}
	}
	zh, zh = TZOfTime(t)
	return
}

func TZOfTime(t time.Time) (zh, zm int) {
	loc := t.Format("-0700")
	zh, _ = strconv.Atoi(loc[0:3])
	zm, _ = strconv.Atoi(loc[3:])
	return
}
