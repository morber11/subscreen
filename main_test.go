package main

import (
	"testing"
	"time"
)

func TestOPSMidpoint(t *testing.T) {
	cases := []struct {
		start, end time.Duration
		want       time.Duration
	}{
		{0, 5 * time.Second, 2 * time.Second},
		{0, 4 * time.Second, 2 * time.Second},
		{0, 1 * time.Second, 0},
		{80 * time.Second, 85 * time.Second, 82 * time.Second},
		{0, 3 * time.Second, 1 * time.Second},
	}

	for _, c := range cases {
		got := opsMidpoint(c.start, c.end)
		if got != c.want {
			t.Errorf("opsMidpoint(%v, %v) = %v, want %v", c.start, c.end, got, c.want)
		}
	}
}

func TestFormatTimestamp(t *testing.T) {
	cases := []struct {
		input time.Duration
		want  string
	}{
		{0, "00-00-00-000"},
		{time.Second, "00-00-01-000"},
		{80*time.Second + 500*time.Millisecond, "00-01-20-500"},
		{time.Hour + 2*time.Minute + 3*time.Second + 456*time.Millisecond, "01-02-03-456"},
	}

	for _, c := range cases {
		got := formatTimestamp(c.input)
		if got != c.want {
			t.Errorf("formatTimestamp(%v) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestFormatSRTTime(t *testing.T) {
	cases := []struct {
		input time.Duration
		want  string
	}{
		{0, "00:00:00,000"},
		{time.Second, "00:00:01,000"},
		{80*time.Second + 500*time.Millisecond, "00:01:20,500"},
		{time.Hour + 2*time.Minute + 3*time.Second + 456*time.Millisecond, "01:02:03,456"},
	}

	for _, c := range cases {
		got := formatSRTTime(c.input)
		if got != c.want {
			t.Errorf("formatSRTTime(%v) = %q, want %q", c.input, got, c.want)
		}
	}
}
