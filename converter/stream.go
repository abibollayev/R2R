package main

import (
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
)

type StreamInfo struct {
	Path       string
	VideoCodec string
	AudioCodec string
	HasAudio   bool
}

type StreamManager struct {
	active   map[string]*StreamInfo
	procs    map[string]*exec.Cmd
	procsMux *sync.Mutex
}

func NewStreamManager() *StreamManager {
	return &StreamManager{
		active:   make(map[string]*StreamInfo),
		procs:    make(map[string]*exec.Cmd),
		procsMux: &sync.Mutex{},
	}
}

func (sm *StreamManager) HandleLine(text string) {
	switch {
	case reCreated.MatchString(text):
		sm.handleCreated(text)
	case reDestroyed.MatchString(text):
		sm.handleDestroyed(text)
	case rePublishing.MatchString(text):
		sm.handlePublishing(text)
	}
}

func (sm *StreamManager) handleCreated(text string) {
	matches := reCreated.FindStringSubmatch(text)
	if matches == nil {
		return
	}
	path := matches[1]
	sm.active[path] = &StreamInfo{Path: path}
	slog.Info("[STREAMING] OPENED", slog.String("path", path))
}

func (sm *StreamManager) handleDestroyed(text string) {
	matches := reDestroyed.FindStringSubmatch(text)
	if matches == nil {
		return
	}
	path := matches[1]
	if _, ok := sm.active[path]; ok {
		slog.Info("[STREAMING] CLOSED", slog.String("path", path))
		sm.stopConvert(path)
		delete(sm.active, path)
	} else {
		slog.Info("[STREAMING] CLOSED", slog.String("path", path))
	}
}

func (sm *StreamManager) handlePublishing(text string) {
	matches := rePublishing.FindStringSubmatch(text)
	if matches == nil {
		return
	}
	path := matches[1]
	trackCount := matches[2]
	codecList := strings.Split(matches[3], ", ")

	var videoCodec, audioCodec string
	if len(codecList) > 0 {
		videoCodec = strings.TrimSpace(codecList[0])
	}
	if len(codecList) > 1 {
		audioCodec = strings.TrimSpace(codecList[1])
	}

	stream, ok := sm.active[path]
	if !ok {
		return
	}

	stream.VideoCodec = videoCodec
	stream.AudioCodec = audioCodec
	stream.HasAudio = trackCount == "2"

	slog.Info("[STREAMING] DETECTED", slog.String("path", path), slog.String("codec", videoCodec), slog.Bool("audio", stream.HasAudio))

	sm.procsMux.Lock()
	_, exists := sm.procs[path]
	sm.procsMux.Unlock()

	if !exists {
		sm.startConvert(path)
	}
}

func (sm *StreamManager) startConvert(path string) {
	input := fmt.Sprintf("rtmp://rtmp-server:1935/%s", path)
	output := fmt.Sprintf("rtsp://rtsp-server:8554/%s", path)

	cmd := exec.Command("ffmpeg", "-i", input, "-c", "copy", "-f", "rtsp", output)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		slog.Error("Failed to start ffmpeg", slog.String("path", path), slog.Any("error", err))
		return
	}

	sm.procsMux.Lock()
	sm.procs[path] = cmd
	sm.procsMux.Unlock()

	slog.Info("[CONVERTING] STARTED", slog.String("path", path))
}

func (sm *StreamManager) stopConvert(path string) {
	sm.procsMux.Lock()
	defer sm.procsMux.Unlock()

	if cmd, ok := sm.procs[path]; ok {
		err := cmd.Process.Kill()
		if err != nil {
			slog.Error("Failed to kill ffmpeg", slog.String("path", path), slog.Any("error", err))
		}

		_, waitErr := cmd.Process.Wait()
		if waitErr != nil {
			slog.Warn("Error while waiting ffmpeg", slog.String("path", path), slog.Any("error", waitErr))
		}

		slog.Info("[CONVERTING] STOPPED", slog.String("path", path))
		delete(sm.procs, path)
	}
}
