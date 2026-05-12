package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Entry struct {
	Index int
	Start time.Duration
	End   time.Duration
	Text  string
}

func ParseSRT(path string) ([]Entry, error) {
	f, err := os.Open(path)

	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var blocks [][]string
	var block []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			if len(block) > 0 {
				blocks = append(blocks, block)
				block = nil
			}
		} else {
			block = append(block, line)
		}
	}

	if len(block) > 0 {
		blocks = append(blocks, block)
	}

	var entries []Entry
	for _, b := range blocks {
		if len(b) < 3 {
			continue
		}

		index, err := strconv.Atoi(strings.TrimSpace(b[0]))
		if err != nil {
			continue
		}

		parts := strings.SplitN(b[1], " --> ", 2)
		if len(parts) != 2 {
			continue
		}

		start, err := parseSRTTime(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}

		end, err := parseSRTTime(strings.TrimSpace(parts[1]))
		if err != nil {
			continue
		}

		entries = append(entries, Entry{
			Index: index,
			Start: start,
			End:   end,
			Text:  strings.Join(b[2:], "\n"),
		})
	}
	return entries, nil
}

func parseSRTTime(s string) (time.Duration, error) {
	// format: HH:MM:SS,mmm
	parts := strings.SplitN(s, ":", 3)

	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid SRT time: %s", s)
	}

	// hour
	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid SRT time: %s", s)
	}

	// minutes
	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid SRT time: %s", s)
	}

	// get seconds and milliseconds
	secParts := strings.SplitN(parts[2], ",", 2)
	if len(secParts) != 2 {
		return 0, fmt.Errorf("invalid SRT time: %s", s)
	}

	// seconds
	sec, err := strconv.Atoi(secParts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid SRT time: %s", s)
	}

	// milliseconds
	ms, err := strconv.Atoi(secParts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid SRT time: %s", s)
	}

	return time.Duration(h)*time.Hour +
		time.Duration(m)*time.Minute +
		time.Duration(sec)*time.Second +
		time.Duration(ms)*time.Millisecond, nil
}
