package nbalive

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestDurationUnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{
			name:  "standard NBA clock",
			input: `"PT11M58.00S"`,
			want:  11*time.Minute + 58*time.Second,
		},
		{
			name:  "expired clock zero seconds",
			input: `"PT0S"`,
			want:  0,
		},
		{
			name:  "all components",
			input: `"PT1H2M3.5S"`,
			want:  1*time.Hour + 2*time.Minute + 3*time.Second + 500*time.Millisecond,
		},
		{
			name:  "minutes only",
			input: `"PT5M"`,
			want:  5 * time.Minute,
		},
		{
			name:  "seconds only",
			input: `"PT30S"`,
			want:  30 * time.Second,
		},
		{
			name:  "hours only",
			input: `"PT1H"`,
			want:  1 * time.Hour,
		},
		{
			name:  "empty string becomes zero",
			input: `""`,
			want:  0,
		},
		{
			name:  "null becomes zero",
			input: `null`,
			want:  0,
		},
		{
			name:    "invalid format",
			input:   `"INVALID"`,
			wantErr: true,
		},
		{
			name:    "not a duration",
			input:   `"not a duration"`,
			wantErr: true,
		},
		{
			name:    "wrong JSON type",
			input:   `123`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var got Duration
			err := json.Unmarshal([]byte(tc.input), &got)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), "nbalive") {
					t.Fatalf("expected error to contain %q, got %q", "nbalive", err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Duration != tc.want {
				t.Fatalf("duration mismatch: got %v, want %v", got.Duration, tc.want)
			}
		})
	}
}

func TestDurationMarshalJSONRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   time.Duration
	}{
		{
			name: "zero",
			in:   0,
		},
		{
			name: "minutes and seconds",
			in:   11*time.Minute + 58*time.Second,
		},
		{
			name: "hour equivalent",
			in:   1*time.Hour + 2*time.Minute + 3*time.Second + 500*time.Millisecond,
		},
		{
			name: "seconds only",
			in:   30 * time.Second,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orig := Duration{Duration: tc.in}
			b, err := json.Marshal(orig)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			var roundTripped Duration
			if err := json.Unmarshal(b, &roundTripped); err != nil {
				t.Fatalf("unmarshal after marshal failed: %v", err)
			}

			if roundTripped.Duration != orig.Duration {
				t.Fatalf("roundtrip mismatch: marshaled %s, got %v, want %v", string(b), roundTripped.Duration, orig.Duration)
			}
		})
	}
}

func TestDurationString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   time.Duration
		want string
	}{
		{
			name: "zero",
			in:   0,
			want: "0s",
		},
		{
			name: "standard clock",
			in:   11*time.Minute + 58*time.Second,
			want: "11m58s",
		},
		{
			name: "with fractional seconds",
			in:   1*time.Hour + 2*time.Minute + 3*time.Second + 500*time.Millisecond,
			want: "1h2m3.5s",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := Duration{Duration: tc.in}
			if got := d.String(); got != tc.want {
				t.Fatalf("String() mismatch: got %q, want %q", got, tc.want)
			}
		})
	}
}
