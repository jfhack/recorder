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
		fname := fmt.Sprintf("%s-%s.mkv", cam.Name, start.Format("20060102-150405"))
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

	for {
		select {
		case <-segCtx.Done():
			log.Printf("[%s] segment #%d ended at %s", cam.Name, segment, time.Now().Format(time.RFC3339))
			return
		default:
		}

		cmd := exec.CommandContext(segCtx, "ffmpeg",
			"-rtsp_transport", cam.Transport,
			"-i", cam.URL,
			"-c", "copy",
			outPath,
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
