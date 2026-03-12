package nbalive

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var durRe = regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:([\d.]+)S)?$`)

// Duration wraps time.Duration with NBA ISO 8601 parsing ("PT11M58.00S").
// Zero value is valid (0s). Handles "", null, and malformed values gracefully.
type Duration struct{ time.Duration }

func (d Duration) String() string { return d.Duration.String() }

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s *string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("nbalive: duration: %w", err)
	}
	if s == nil || *s == "" {
		d.Duration = 0
		return nil
	}
	m := durRe.FindStringSubmatch(*s)
	if m == nil {
		return fmt.Errorf("nbalive: invalid duration %q", *s)
	}
	var total time.Duration
	if m[1] != "" {
		h, _ := strconv.Atoi(m[1])
		total += time.Duration(h) * time.Hour
	}
	if m[2] != "" {
		min, _ := strconv.Atoi(m[2])
		total += time.Duration(min) * time.Minute
	}
	if m[3] != "" {
		sec, _ := strconv.ParseFloat(m[3], 64)
		total += time.Duration(sec * float64(time.Second))
	}
	d.Duration = total
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	totalSec := d.Duration.Seconds()
	min := int(totalSec) / 60
	sec := totalSec - float64(min*60)
	return json.Marshal(fmt.Sprintf("PT%dM%.2fS", min, sec))
}
