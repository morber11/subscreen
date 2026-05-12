package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
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
	offset   time.Duration
	format   string
	ops      bool
	outDir   string
	outJSON  string
	fastSeek bool
	forceYes bool
	trySync  float64
	autoSync bool
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
	offset := flag.Duration("offset", 0, "shift subtitle timestamps (e.g. 2s, -500ms)")
	flag.DurationVar(offset, "o", 0, "alias for -offset")
	fastSeek := flag.Bool("fast-seek", false, "fast but less accurate seeking (may capture wrong frame)")
	flag.BoolVar(fastSeek, "fs", false, "alias for -fast-seek")
	forceYes := flag.Bool("y", false, "yes to all prompts")
	trySync := flag.Float64("try-sync", 1.0, "rate multiplier for linear drift correction (e.g. 0.9986)")
	autoSync := flag.Bool("ts", false, "auto-detect rate from video and SRT duration")
	flag.Parse()

	if *video == "" {
		fmt.Fprintln(os.Stderr, "error: -video is required")
		flag.Usage()
		os.Exit(1)
	}

	if *srtPath == "" {
		*srtPath = findSRT(*video, *forceYes)
		if *srtPath == "" {
			fmt.Fprintln(os.Stderr, "error: no -srt given and no .srt file found next to video")
			os.Exit(1)
		}
	}

	if *format != "jpeg" && *format != "png" {
		fmt.Fprintln(os.Stderr, "error: -format must be jpeg or png")
		os.Exit(1)
	}

	if *trySync <= 0 {
		fmt.Fprintln(os.Stderr, "error: -try-sync must be greater than 0")
		os.Exit(1)
	}

	return config{
		video:    *video,
		srt:      *srtPath,
		start:    *start,
		end:      *end,
		delay:    *delay,
		offset:   *offset,
		format:   *format,
		ops:      *ops,
		outDir:   *outDir,
		outJSON:  *outJSON,
		fastSeek: *fastSeek,
		forceYes: *forceYes,
		trySync:  *trySync,
		autoSync: *autoSync,
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
	// if autoSync is enabled, calculate trySync based on the ratio of video duration to subtitle duration
	if cfg.autoSync {
		if dur, err := getVideoDuration(cfg.video); err == nil {
			// rate = actual video length / where the SRT thinks the video ends;
			// corrects linear drift from framerate mismatches (e.g. 23.976fps SRT on a 25fps encode)
			cfg.trySync = float64(dur) / float64(entries[len(entries)-1].End)
		} else {
			fmt.Fprintf(os.Stderr, "warning: -ts could not read video duration: %v\n", err)
		}
	}
	// if subtitles are out of sync, we can apply an offset
	if cfg.offset != 0 || cfg.trySync != 1.0 {
		for i := range entries {
			entries[i].Start = max(0, time.Duration(float64(entries[i].Start)*cfg.trySync)+cfg.offset)
			entries[i].End = max(0, time.Duration(float64(entries[i].End)*cfg.trySync)+cfg.offset)
		}
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

	// for CTRL+C handling
	var stopping int32
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, os.Interrupt)

	go func() {
		<-sigCh
		atomic.StoreInt32(&stopping, 1)
		fmt.Fprintln(os.Stderr, "\nwinding down, finishing current screenshot... (ctrl+c again to force quit)")
		<-sigCh
		fmt.Fprintln(os.Stderr, "\nforce quit")
		os.Exit(1)
	}()

	done := 0
	var recentDurations [5]time.Duration
	recentCount := 0
	printProgress(done, total, 0)

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
		if atomic.LoadInt32(&stopping) == 1 {
			break
		}
		shots := []string{}

		for _, t := range w.timestamps {
			if atomic.LoadInt32(&stopping) == 1 {
				break
			}
			filename := fmt.Sprintf("%d-%s.%s", w.entry.Index, formatTimestamp(t), ext)
			outPath := filepath.Join(cfg.outDir, filename)

			start := time.Now()
			if err := TakeScreenshot(cfg.video, t, outPath, cfg.format, cfg.fastSeek); err != nil {
				fmt.Fprintf(os.Stderr, "\nwarning: screenshot at %s failed: %v\n", formatSRTTime(t), err)
			} else {
				shots = append(shots, outPath)
			}

			recentDurations[recentCount%5] = time.Since(start)
			recentCount++
			window := recentCount

			if window > 5 {
				window = 5
			}
			var sum time.Duration
			for k := 0; k < window; k++ {
				sum += recentDurations[k]
			}

			avg := sum / time.Duration(window)
			eta := avg * time.Duration(total-done-1)

			done++
			printProgress(done, total, eta)
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
func printProgress(done, total int, eta time.Duration) {
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

	if eta > 0 {
		fmt.Printf("\r[%s] %d/%d (%d%%) ETA %s", bar, done, total, pct, fmtETA(eta))
	} else {
		fmt.Printf("\r[%s] %d/%d (%d%%)", bar, done, total, pct)
	}
}

func fmtETA(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	}

	if m > 0 {
		return fmt.Sprintf("%dm%02ds", m, s)
	}

	return fmt.Sprintf("%ds", s)
}

func findSRT(video string, forceYes bool) string {
	dir := filepath.Dir(video)
	matches, err := filepath.Glob(filepath.Join(dir, "*.srt"))

	if err != nil || len(matches) == 0 {
		return ""
	}

	if forceYes {
		return matches[0]
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
