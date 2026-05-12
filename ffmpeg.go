package main

import (
	"fmt"
	"os/exec"
	"time"
)

func TakeScreenshot(video string, ts time.Duration, outPath string, format string, fastSeek bool) error {
	var args []string

	if fastSeek {
		args = []string{
			"-ss", fmt.Sprintf("%.3f", ts.Seconds()),
			"-i", video,
		}
	} else {
		const preSeekBuf = 5 * time.Second
		preSeek := ts - preSeekBuf

		if preSeek < 0 {
			preSeek = 0
		}

		args = []string{
			"-ss", fmt.Sprintf("%.3f", preSeek.Seconds()),
			"-i", video,
			"-ss", fmt.Sprintf("%.3f", ts.Seconds()),
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
