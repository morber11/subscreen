package main

import (
	"os"
	"testing"
	"time"
)

func TestParseSRTTime(t *testing.T) {
	cases := []struct {
		input string
		want  time.Duration
	}{
		{"00:00:00,000", 0},
		{"00:00:01,000", time.Second},
		{"00:01:20,000", 80 * time.Second},
		{"01:02:03,456", time.Hour + 2*time.Minute + 3*time.Second + 456*time.Millisecond},
	}

	for _, c := range cases {
		got, err := parseSRTTime(c.input)
		if err != nil {
			t.Errorf("parseSRTTime(%q) error: %v", c.input, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseSRTTime(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}

func TestParseSRTTimeInvalid(t *testing.T) {
	cases := []string{"", "00:00", "00:00:00", "ab:cd:ef,ghi"}
	for _, c := range cases {
		_, err := parseSRTTime(c)
		if err == nil {
			t.Errorf("parseSRTTime(%q) expected error, got nil", c)
		}
	}
}

func TestParseSRT(t *testing.T) {
	content := "1\n00:00:01,000 --> 00:00:04,000\nHello world\n\n2\n00:00:05,500 --> 00:00:08,000\nSecond subtitle\nwith two lines\n\n3\n00:01:00,000 --> 00:01:02,000\nThird\n"
	f, err := os.CreateTemp("", "*.srt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()

	entries, err := ParseSRT(f.Name())
	if err != nil {
		t.Fatalf("ParseSRT error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	e := entries[0]
	if e.Index != 1 {
		t.Errorf("entry 0 index: got %d, want 1", e.Index)
	}
	if e.Start != time.Second {
		t.Errorf("entry 0 start: got %v, want 1s", e.Start)
	}
	if e.End != 4*time.Second {
		t.Errorf("entry 0 end: got %v, want 4s", e.End)
	}
	if e.Text != "Hello world" {
		t.Errorf("entry 0 text: got %q, want %q", e.Text, "Hello world")
	}

	if entries[1].Text != "Second subtitle\nwith two lines" {
		t.Errorf("entry 1 text: got %q", entries[1].Text)
	}

	if entries[2].Start != time.Minute {
		t.Errorf("entry 2 start: got %v, want 1m", entries[2].Start)
	}
}

func TestParseSRTMissingFile(t *testing.T) {
	_, err := ParseSRT("nonexistent.srt")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}
