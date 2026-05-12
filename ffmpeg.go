package main

import (
	"fmt"
	"os/exec"
	"time"
)

func TakeScreenshot(video string, ts time.Duration, outPath string, format string) error {
	args := []string{
		"-ss", fmt.Sprintf("%.3f", ts.Seconds()),
		"-i", video,
		"-vframes", "1",
	}

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
