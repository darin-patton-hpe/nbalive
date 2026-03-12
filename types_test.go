package nbalive

import (
	"encoding/json"
	"testing"
)

func TestBoolStringUnmarshalJSON(t *testing.T) {
	type wrapper struct {
		V BoolString `json:"v"`
	}

	tests := []struct {
		name    string
		input   string
		want    bool
		wantErr bool
	}{
		{name: `"1" string maps to true`, input: `{"v":"1"}`, want: true},
		{name: `"0" string maps to false`, input: `{"v":"0"}`, want: false},
		{name: `empty string maps to false`, input: `{"v":""}`, want: false},
		{name: `bool true maps to true`, input: `{"v":true}`, want: true},
		{name: `bool false maps to false`, input: `{"v":false}`, want: false},
		{name: `number is invalid`, input: `{"v":123}`, wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var got wrapper
			err := json.Unmarshal([]byte(tc.input), &got)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.V.Bool() != tc.want {
				t.Fatalf("got %v, want %v", got.V.Bool(), tc.want)
			}
		})
	}
}

func TestBoolStringBool(t *testing.T) {
	tests := []struct {
		name string
		in   BoolString
		want bool
	}{
		{name: "true value", in: BoolString(true), want: true},
		{name: "false value", in: BoolString(false), want: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.in.Bool(); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGameStatusString(t *testing.T) {
	tests := []struct {
		name string
		in   GameStatus
		want string
	}{
		{name: "scheduled", in: GameScheduled, want: "Scheduled"},
		{name: "in progress", in: GameInProgress, want: "In Progress"},
		{name: "final", in: GameFinal, want: "Final"},
		{name: "zero unknown", in: GameStatus(0), want: "Unknown"},
		{name: "arbitrary unknown", in: GameStatus(99), want: "Unknown"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.in.String(); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
