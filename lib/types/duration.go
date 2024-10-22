package types

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Duration time.Duration

func (d Duration) String() string {
	if int64(d) == 0 {
		return "-"
	}

	s := time.Duration(int64(d)).String()
	// the regular duration stops at hours
	// so we may have xxxxxxh...
	// have to convert xxxxxxh to year(y), month(M), week(w), day(d)
	a := strings.Split(s, "h")
	if len(a) != 2 {
		return s
	}

	h, err := strconv.ParseInt(a[0], 10, 64)
	if err != nil {
		return s
	}

	s = ""
	if y := h / (365 * 24); y != 0 {
		h %= (365 * 24)
		s += fmt.Sprintf("%vy", y)
	}
	if m := h / (30 * 24); m != 0 {
		h %= (30 * 24)
		s += fmt.Sprintf("%vM", m)
	}
	if w := h / (7 * 24); w != 0 {
		h %= (7 * 24)
		s += fmt.Sprintf("%vw", w)
	}
	if h != 0 {
		s += fmt.Sprintf("%vh", h)
	}
	if a[1] != "0m0s" {
		s += a[1]
	}
	return s
}

func ParseDuration(s string) (Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("invalid duration")
	}

	sign := int64(1)
	if s[0] == '-' {
		sign, s = -1, s[1:]
	} else if s[0] == '+' {
		s = s[1:]
	}
	if s == "0" || s == "-" {
		return 0, nil
	}

	v, units, mul := int64(0), "yMwd", map[byte]int64{
		'y': 365 * 24 * int64(time.Hour),
		'M': 30 * 24 * int64(time.Hour),
		'w': 7 * 24 * int64(time.Hour),
		'd': 1 * 24 * int64(time.Hour),
	}
	for s != "" {
		a, p := int64(0), 0
		for p < len(s) {
			n := int(s[p]) - '0'
			if 0 > n || n > 9 {
				break
			}
			a, p = a*10+int64(n), p+1
		}
		if p >= len(s) {
			return 0, fmt.Errorf("invalid duration")
		}

		u := s[p]
		if !strings.ContainsAny(string(u), units) {
			a, err := time.ParseDuration(s)
			if err != nil {
				return 0, err
			}
			v += int64(a)
			break
		}

		v, s = v+a*mul[u], s[p+1:]
	}

	v = sign * v
	return Duration(v), nil
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}

	i, err := ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration: %v", err)
	}
	*d = i
	return nil
}

func (d Duration) MarshalYAML() (interface{}, error) {
	return d.String(), nil
}

func (d Duration) Value() (driver.Value, error) {
	s := fmt.Sprintf("%v", int64(d))
	return s, nil
}

func (d *Duration) Scan(value interface{}) error {
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("invalid type assertion")
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*d = Duration(i)
	return nil
}
