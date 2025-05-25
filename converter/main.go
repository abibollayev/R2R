package main

import (
	"log/slog"
	"os"
	"os/exec"
	"regexp"

	"github.com/hpcloud/tail"
)

const (
	rtmpLogFile = "/app/logs/rtmp-server.log"
)

var (
	reCreated    = regexp.MustCompile(`\[path (live/[^]]+)] created`)
	reDestroyed  = regexp.MustCompile(`\[path (live/[^]]+)] destroyed`)
	rePublishing = regexp.MustCompile(`is publishing to path '([^']+)', (\d+) tracks? \(([^)]+)\)`)
)

func main() {
	slog.Info("Check dependencies...")
	if err := checkFFmpeg(); err != nil {
		slog.Error("ffmpeg: is not installed or not found")
		os.Exit(1)
	}
	slog.Info("ffmpeg: is installed and available")

	t, err := tail.TailFile(rtmpLogFile, tail.Config{
		Follow:    true,
		ReOpen:    true,
		Location:  &tail.SeekInfo{Offset: 0, Whence: 2},
		MustExist: true,
	})
	if err != nil {
		slog.Error("Failed to tail file: ", slog.Any("err", err))
		os.Exit(1)
	}

	slog.Info("Start R2R/converter")

	manager := NewStreamManager()

	for line := range t.Lines {
		manager.HandleLine(line.Text)
	}
}

func checkFFmpeg() error {
	cmd := exec.Command("ffmpeg", "-version")
	err := cmd.Run()
	return err
}
