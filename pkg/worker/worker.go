package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"recorder/pkg/config"
)

func Run(ctx context.Context, cam config.CamCfg) {
	dur, _ := time.ParseDuration(cam.Duration)
	slack, _ := time.ParseDuration(cam.Slack)

	base := time.Now()
	segment := 0

	for {
		start := base.Add(time.Duration(segment) * dur)

		if delay := time.Until(start); delay > 0 {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return
			}
		}

		dateDir := filepath.Join(cam.Name, start.Format("20060102"))
		if err := os.MkdirAll(dateDir, 0o755); err != nil {
			log.Printf("[%s] mkdir error: %v", cam.Name, err)
		}
		fname := fmt.Sprintf("%s-%s%s", cam.Name, start.Format("20060102-150405"), cam.Suffix)
		outPath := filepath.Join(dateDir, fname)

		log.Printf("[%s] scheduling segment #%d at %s → %s",
			cam.Name, segment,
			start.Format(time.RFC3339), outPath,
		)

		go runSegment(ctx, cam, segment, start, outPath, dur, slack)

		segment++
	}
}

func runSegment(parentCtx context.Context, cam config.CamCfg, segment int, start time.Time, outPath string, dur, slack time.Duration) {
	end := start.Add(dur + slack)
	segCtx, cancel := context.WithDeadline(parentCtx, end)
	defer cancel()

	var args []string
	in := append([]string{}, cam.Args.Input...)
	if cam.AutoArgsEnabled() {
		if cam.IsRTSP() && !config.HasFlag(in, "-rtsp_transport") {
			in = append(in, "-rtsp_transport", "udp")
		}
	}

	args = append(args, cam.Args.Global...)
	args = append(args, in...)
	args = append(args, "-i", cam.URL)

	out := append([]string{}, cam.Args.Output...)
	if cam.AutoArgsEnabled() {
		if !config.HasCodecFlag(out) && !config.HasReencodeOnlyFlags(out) {
			out = append(out, "-c", "copy")
		}
	}
	args = append(args, out...)
	args = append(args, outPath)

	for {
		select {
		case <-segCtx.Done():
			log.Printf("[%s] segment #%d ended at %s", cam.Name, segment, time.Now().Format(time.RFC3339))
			return
		default:
		}

		cmd := exec.CommandContext(segCtx, "ffmpeg",
		  args...
		)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr

		log.Printf("[%s] segment #%d: launching ffmpeg", cam.Name, segment)
		err := cmd.Run()
		if segCtx.Err() == context.DeadlineExceeded {
			return
		}
		log.Printf("[%s] segment #%d: ffmpeg error: %v; retrying in 5s", cam.Name, segment, err)

		select {
		case <-time.After(5 * time.Second):
		case <-segCtx.Done():
			return
		}
	}
}
