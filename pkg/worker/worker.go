package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"recorder/pkg/config"
)

const (
	retryDelay = 5 * time.Second
	stopTimeout = 10 * time.Second
)

func Run(ctx context.Context, cam config.CamCfg) {
	dur, _ := time.ParseDuration(cam.Duration)
	slack, _ := time.ParseDuration(cam.Slack)
	args := ffmpegArgs(cam)

	var wg sync.WaitGroup
	defer wg.Wait()

	base := time.Now()
	for segment := 0; ; segment++ {
		start := base.Add(time.Duration(segment) * dur)

		if delay := time.Until(start); delay > 0 {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return
			}
		}

		log.Printf("[%s] scheduling segment #%d at %s", cam.Name, segment, start.Format(time.RFC3339))

		wg.Add(1)
		go func() {
			defer wg.Done()
			runSegment(ctx, cam, segment, start, start.Add(dur+slack), args)
		}()
	}
}

func ffmpegArgs(cam config.CamCfg) []string {
	var args []string
	global := append([]string{}, cam.Args.Global...)
	in := append([]string{}, cam.Args.Input...)
	if cam.AutoArgsEnabled() {
		if !config.HasFlag(global, "-loglevel") && !config.HasFlag(global, "-v") {
			global = append(global, "-loglevel", "error")
		}
		if !config.HasFlag(global, "-stats") && !config.HasFlag(global, "-nostats") {
			global = append(global, "-nostats")
		}
		if cam.IsRTSP() && !config.HasFlag(in, "-rtsp_transport") {
			in = append(in, "-rtsp_transport", "udp")
		}
	}

	args = append(args, global...)
	args = append(args, in...)
	args = append(args, "-i", cam.URL)

	out := append([]string{}, cam.Args.Output...)
	if cam.AutoArgsEnabled() {
		if !config.HasCodecFlag(out) && !config.HasReencodeOnlyFlags(out) {
			out = append(out, "-c", "copy")
		}
	}
	return append(args, out...)
}

func runSegment(parentCtx context.Context, cam config.CamCfg, segment int, start, end time.Time, args []string) {
	segCtx, cancel := context.WithDeadline(parentCtx, end)
	defer cancel()

	fileTime := start
	for {
		outPath, err := outputPath(cam, fileTime)
		if err == nil {
			log.Printf("[%s] segment #%d: launching ffmpeg → %s", cam.Name, segment, outPath)
			err = runFFmpeg(segCtx, slices.Concat(args, []string{outPath}))
		}
		if segCtx.Err() != nil {
			log.Printf("[%s] segment #%d ended at %s", cam.Name, segment, time.Now().Format(time.RFC3339))
			return
		}
		log.Printf("[%s] segment #%d: ffmpeg exited early (%v); retrying in %s", cam.Name, segment, err, retryDelay)

		select {
		case <-time.After(retryDelay):
		case <-segCtx.Done():
			return
		}
		fileTime = time.Now()
	}
}

func outputPath(cam config.CamCfg, t time.Time) (string, error) {
	dir := filepath.Join(cam.Name, t.Format("20060102"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}
	return filepath.Join(dir, fmt.Sprintf("%s-%s%s", cam.Name, t.Format("20060102-150405"), cam.Suffix)), nil
}

func runFFmpeg(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	cmd.Cancel = func() error {
		_, err := stdin.Write([]byte("q"))
		return err
	}
	cmd.WaitDelay = stopTimeout
	return cmd.Run()
}
