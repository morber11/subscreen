package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type screenshotEntry struct {
	Index       int      `json:"index"`
	Start       string   `json:"start"`
	End         string   `json:"end"`
	Text        string   `json:"text"`
	Screenshots []string `json:"screenshots"`
}

type config struct {
	video    string
	srt      string
	start    time.Duration
	end      time.Duration
	delay    time.Duration
	format   string
	ops      bool
	outDir   string
	outJSON  string
	fastSeek bool
}

type work struct {
	entry      Entry
	timestamps []time.Duration
}

// appendEntriesKey strips the closing } from a JSON object and opens an entries array.
// e.g. {"video":"x","subtitles":"y"} -> {"video":"x","subtitles":"y","entries":[
// only reason we need is it because the JSON is written incrementally instead of
// all at the end
func appendEntriesKey(b []byte) string {
	return string(b[:len(b)-1]) + ",\n  \"entries\": ["
}

func parseFlags() config {
	video := flag.String("video", "", "video file path (required)")
	srtPath := flag.String("srt", "", "SRT subtitle file path (required)")
	start := flag.Duration("start", 0, "time range start (e.g. 30m); default is beginning")
	end := flag.Duration("end", 0, "time range end (e.g. 35m); 0 means no limit")
	delay := flag.Duration("delay", time.Second, "interval between screenshots per subtitle")
	format := flag.String("format", "jpeg", "image format: jpeg or png")
	ops := flag.Bool("one-per-subtitle", false, "take one screenshot per subtitle")
	flag.BoolVar(ops, "ops", false, "alias for -one-per-subtitle")
	outDir := flag.String("out-dir", "screenshots", "output directory for screenshots")
	outJSON := flag.String("out-json", "output.json", "output JSON file path")
	fastSeek := flag.Bool("fast-seek", false, "fast but less accurate seeking (may capture wrong frame)")
	flag.BoolVar(fastSeek, "fs", false, "alias for -fast-seek")
	flag.Parse()

	if *video == "" {
		fmt.Fprintln(os.Stderr, "error: -video is required")
		flag.Usage()
		os.Exit(1)
	}

	if *srtPath == "" {
		*srtPath = findSRT(*video)
		if *srtPath == "" {
			fmt.Fprintln(os.Stderr, "error: no -srt given and no .srt file found next to video")
			os.Exit(1)
		}
	}

	if *format != "jpeg" && *format != "png" {
		fmt.Fprintln(os.Stderr, "error: -format must be jpeg or png")
		os.Exit(1)
	}

	return config{
		video:    *video,
		srt:      *srtPath,
		start:    *start,
		end:      *end,
		delay:    *delay,
		format:   *format,
		ops:      *ops,
		outDir:   *outDir,
		outJSON:  *outJSON,
		fastSeek: *fastSeek,
	}
}

func opsMidpoint(start, end time.Duration) time.Duration {
	durSec := int64((end - start) / time.Second)
	return start + time.Duration(durSec/2)*time.Second
}

func buildQueue(entries []Entry, cfg config) []work {
	var queue []work
	for _, e := range entries {
		if e.End <= cfg.start {
			continue
		}

		if cfg.end > 0 && e.Start >= cfg.end {
			continue
		}

		var timestamps []time.Duration
		if cfg.ops {
			timestamps = []time.Duration{opsMidpoint(e.Start, e.End)}
		} else {
			for t := e.Start; t < e.End; t += cfg.delay {
				timestamps = append(timestamps, t)
			}
		}

		queue = append(queue, work{e, timestamps})
	}

	return queue
}

func main() {
	cfg := parseFlags()

	entries, err := ParseSRT(cfg.srt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing SRT: %v\n", err)
		os.Exit(1)
	}

	// 0755 is rwxr-xr-x
	if err := os.MkdirAll(cfg.outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating output directory: %v\n", err)
		os.Exit(1)
	}

	ext := "jpg"
	if cfg.format == "png" {
		ext = "png"
	}

	queue := buildQueue(entries, cfg)

	total := 0
	for _, w := range queue {
		total += len(w.timestamps)
	}

	done := 0
	printProgress(done, total)

	jsonFile, err := os.Create(cfg.outJSON)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating JSON file: %v\n", err)
		os.Exit(1)
	}
	defer jsonFile.Close()

	header, _ := json.Marshal(map[string]string{"video": cfg.video, "subtitles": cfg.srt})
	// write opening: {"video":"...","subtitles":"...","entries":[
	fmt.Fprintf(jsonFile, "%s\n", appendEntriesKey(header))

	first := true
	entryCount := 0
	for _, w := range queue {
		shots := []string{}

		for _, t := range w.timestamps {
			filename := fmt.Sprintf("%04d_%s.%s", w.entry.Index, formatTimestamp(t), ext)
			outPath := filepath.Join(cfg.outDir, filename)

			if err := TakeScreenshot(cfg.video, t, outPath, cfg.format, cfg.fastSeek); err != nil {
				fmt.Fprintf(os.Stderr, "\nwarning: screenshot at %s failed: %v\n", formatSRTTime(t), err)
			} else {
				shots = append(shots, outPath)
			}

			done++
			printProgress(done, total)
		}

		entry := screenshotEntry{
			Index:       w.entry.Index,
			Start:       formatSRTTime(w.entry.Start),
			End:         formatSRTTime(w.entry.End),
			Text:        w.entry.Text,
			Screenshots: shots,
		}

		data, _ := json.MarshalIndent(entry, "    ", "  ")
		if !first {
			fmt.Fprint(jsonFile, ",\n")
		}

		fmt.Fprintf(jsonFile, "    %s", data)
		first = false
		entryCount++
	}
	fmt.Fprintln(jsonFile, "\n  ]\n}")
	fmt.Println()

	fmt.Printf("done: %d entries written to %s\n", entryCount, cfg.outJSON)
}

func splitDuration(d time.Duration) (h, m, s, ms int) {
	h = int(d.Hours())
	m = int(d.Minutes()) % 60
	s = int(d.Seconds()) % 60
	ms = int(d.Milliseconds()) % 1000
	return
}

func formatTimestamp(d time.Duration) string {
	h, m, s, ms := splitDuration(d)
	return fmt.Sprintf("%02d-%02d-%02d-%03d", h, m, s, ms)
}

func formatSRTTime(d time.Duration) string {
	h, m, s, ms := splitDuration(d)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}

// progress bar with an arrow ====>
func printProgress(done, total int) {
	const width = 40
	var filled int

	if total > 0 {
		filled = done * width / total
	}

	bar := strings.Repeat("=", filled)
	if filled < width {
		bar += ">"
	}

	bar += strings.Repeat(" ", width-len(bar))

	pct := 0
	if total > 0 {
		pct = done * 100 / total
	}

	fmt.Printf("\r[%s] %d/%d (%d%%)", bar, done, total, pct)
}

func findSRT(video string) string {
	dir := filepath.Dir(video)
	matches, err := filepath.Glob(filepath.Join(dir, "*.srt"))

	if err != nil || len(matches) == 0 {
		return ""
	}

	// reuse the same scanner in case there are multiple .srt files
	scanner := bufio.NewScanner(os.Stdin)
	// if the folder contains an .srt file, ask user if they want to use it
	// consider checking subdirectories for .srt
	for _, match := range matches {
		name := filepath.Base(match)
		fmt.Printf("no -srt given, but %s found. proceed? [y/N] ", name)
		scanner.Scan()
		if strings.ToLower(strings.TrimSpace(scanner.Text())) == "y" {
			return match
		}
	}

	return ""
}
