package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func TakeScreenshot(video string, ts time.Duration, outPath string, format string, fastSeek bool) error {
	var args []string

	// fast seek
	if fastSeek {
		args = []string{
			"-ss", fmt.Sprintf("%.3f", ts.Seconds()),
			"-i", video,
		}
	} else {
		// more accurate seek but much slower
		const preSeekBuf = 5 * time.Second
		preSeek := ts - preSeekBuf

		if preSeek < 0 {
			preSeek = 0
		}
		// second -ss is relative to the pre-seek position, not absolute
		relSeek := ts - preSeek
		args = []string{
			"-ss", fmt.Sprintf("%.3f", preSeek.Seconds()),
			"-i", video,
			"-ss", fmt.Sprintf("%.3f", relSeek.Seconds()),
		}
	}

	args = append(args, "-vframes", "1")

	// add more formats later
	if format != "png" {
		args = append(args, "-q:v", "2")
	}

	args = append(args, "-y", outPath)

	out, err := exec.Command("ffmpeg", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg: %w: %s", err, out)
	}

	return nil
}

func getVideoDuration(path string) (time.Duration, error) {
	out, err := exec.Command("ffprobe", "-v", "quiet", "-show_entries", "format=duration", "-of", "csv=p=0", path).Output()
	if err != nil {
		return 0, err
	}

	secs, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0, err
	}

	return time.Duration(secs * float64(time.Second)), nil
}
