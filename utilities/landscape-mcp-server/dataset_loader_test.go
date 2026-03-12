package main

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
		wantErr bool
		wantY   int
		wantM   time.Month
		wantD   int
	}{
		{
			name:    "empty string returns nil",
			input:   "",
			wantNil: true,
		},
		{
			name:    "whitespace-only returns nil",
			input:   "  ",
			wantNil: true,
		},
		{
			name:  "valid date parses correctly",
			input: "2024-06-15",
			wantY: 2024,
			wantM: time.June,
			wantD: 15,
		},
		{
			name:    "invalid string returns error",
			input:   "not-a-date",
			wantErr: true,
		},
		{
			name:    "wrong format returns error",
			input:   "06/15/2024",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDate(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("expected non-nil time, got nil")
			}
			if got.Year() != tt.wantY {
				t.Errorf("year: got %d, want %d", got.Year(), tt.wantY)
			}
			if got.Month() != tt.wantM {
				t.Errorf("month: got %v, want %v", got.Month(), tt.wantM)
			}
			if got.Day() != tt.wantD {
				t.Errorf("day: got %d, want %d", got.Day(), tt.wantD)
			}
		})
	}
}
