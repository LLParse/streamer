package streamer

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/sirupsen/logrus"
)

// IProcess is an interface around the FFMPEG process
type IProcess interface {
	Spawn(path, URI string) *exec.Cmd
}

// ProcessLoggingOpts describes options for process logging
type ProcessLoggingOpts struct {
	Enabled    bool   // Option to set logging for transcoding processes
	Directory  string // Directory for the logs
	MaxSize    int    // Maximum size of kept logging files in megabytes
	MaxBackups int    // Maximum number of old log files to retain
	MaxAge     int    // Maximum number of days to retain an old log file.
	Compress   bool   // Indicates if the log rotation should compress the log files
}

// Process is the main type for creating new processes
type Process struct {
	keepFiles   bool
	audio       bool
	hlsTime     int
	hlsListSize int
}

// Type check
var _ IProcess = (*Process)(nil)

// NewProcess creates a new process able to spawn transcoding FFMPEG processes
func NewProcess(
	keepFiles bool,
	audio bool,
	hlsTime int,
	hlsListSize int,
) *Process {
	return &Process{keepFiles, audio, hlsTime, hlsListSize}
}

// getHLSFlags are for getting the flags based on the config context
func (p Process) getHLSFlags() string {
	if p.keepFiles {
		return "split_by_time+append_list"
	}
	return "delete_segments+split_by_time+append_list"
}

// Spawn creates a new FFMPEG cmd
func (p Process) Spawn(path, URI string) *exec.Cmd {
	os.MkdirAll(path, os.ModePerm)
	processCommands := []string{
		"-y",
		"-fflags",
		"flush_packets+discardcorrupt+nobuffer",
		"-rtsp_transport",
		"tcp",
		"-i",
		URI,
		"-f",
		"lavfi",
		"-i",
		"anullsrc=channel_layout=stereo:sample_rate=44100",
		"-fps_mode",
		"cfr",
		"-fpsmax",
		"15",
		"-copyts",
		"-vcodec",
		"copy",
		"-movflags",
		"frag_keyframe+empty_moov",
	}
	if !p.audio {
		processCommands = append(processCommands, "-an")
	}
	processCommands = append(processCommands,
		"-hls_flags",
		p.getHLSFlags(),
		"-f",
		"hls",
		"-segment_list_flags",
		"live",
		"-hls_time",
		strconv.Itoa(p.hlsTime),
		"-hls_list_size",
		strconv.Itoa(p.hlsListSize),
		"-hls_segment_filename",
		fmt.Sprintf("%s/%%d.ts", path),
		fmt.Sprintf("%s/index.m3u8", path),
	)
	cmd := exec.Command("ffmpeg", processCommands...)
	logrus.Debugf("%+v command | Process", cmd)
	return cmd
}
