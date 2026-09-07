package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	mediadomain "github.com/savisaluwadana/DevRelOS/internal/domain/media"
	"github.com/savisaluwadana/DevRelOS/internal/storage"
)

func processAvailableMedia(ctx context.Context, store *storage.Store) {
	for {
		processed, err := processNextMedia(ctx, store)
		if err != nil {
			log.Printf("media queue error: %v", err)
			return
		}
		if !processed {
			return
		}
	}
}

func processNextMedia(ctx context.Context, store *storage.Store) (bool, error) {
	job, err := store.ClaimNextMediaJob(ctx)
	if err != nil {
		return false, err
	}
	if job == nil {
		return false, nil
	}
	if job.JobType != "render" {
		job.Status = "failed"
		job.Error = "unsupported media job type in base worker"
		return true, store.FinishMediaJob(ctx, *job, "")
	}

	asset, err := store.GetMediaAsset(ctx, job.ProjectID, job.MediaAssetID)
	if err != nil {
		return finishMediaFailure(ctx, store, job, fmt.Errorf("load media asset: %w", err))
	}
	clips, err := store.ListMediaClips(ctx, job.ProjectID, job.MediaAssetID)
	if err != nil {
		return finishMediaFailure(ctx, store, job, fmt.Errorf("load clip: %w", err))
	}
	var clip *mediadomain.Clip
	for i := range clips {
		if clips[i].ID == job.ClipID {
			clip = &clips[i]
			break
		}
	}
	if clip == nil {
		return finishMediaFailure(ctx, store, job, fmt.Errorf("clip %s not found", job.ClipID))
	}
	if asset.MediaType != "video" {
		return finishMediaFailure(ctx, store, job, fmt.Errorf("base renderer currently supports video assets only"))
	}
	if strings.TrimSpace(asset.SourcePath) == "" {
		return finishMediaFailure(ctx, store, job, fmt.Errorf("rendering requires a local sourcePath"))
	}

	mediaRoot := workerEnvOr("DEVRELOS_MEDIA_ROOT", "/data/media")
	outputRoot := workerEnvOr("DEVRELOS_MEDIA_OUTPUT_ROOT", filepath.Join(mediaRoot, "outputs"))
	source, err := confinedPath(mediaRoot, asset.SourcePath)
	if err != nil {
		return finishMediaFailure(ctx, store, job, err)
	}
	if _, err := os.Stat(source); err != nil {
		return finishMediaFailure(ctx, store, job, fmt.Errorf("source media unavailable: %w", err))
	}
	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		return finishMediaFailure(ctx, store, job, fmt.Errorf("create output directory: %w", err))
	}
	output := filepath.Join(outputRoot, clip.ID+".mp4")
	output, err = confinedPath(outputRoot, output)
	if err != nil {
		return finishMediaFailure(ctx, store, job, err)
	}

	filter, err := videoFilter(clip.AspectRatio)
	if err != nil {
		return finishMediaFailure(ctx, store, job, err)
	}
	startSeconds := float64(clip.StartMS) / 1000
	durationSeconds := float64(clip.EndMS-clip.StartMS) / 1000
	if durationSeconds <= 0 || durationSeconds > 900 {
		return finishMediaFailure(ctx, store, job, fmt.Errorf("clip duration must be between 0 and 900 seconds"))
	}

	_ = store.UpdateMediaClipStatus(ctx, job.ProjectID, clip.ID, "rendering")
	renderCtx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()
	ffmpeg := workerEnvOr("DEVRELOS_FFMPEG_BIN", "ffmpeg")
	cmd := exec.CommandContext(renderCtx, ffmpeg,
		"-hide_banner", "-loglevel", "error", "-y",
		"-ss", strconv.FormatFloat(startSeconds, 'f', 3, 64),
		"-i", source,
		"-t", strconv.FormatFloat(durationSeconds, 'f', 3, 64),
		"-vf", filter,
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "23",
		"-c:a", "aac", "-movflags", "+faststart", output,
	)
	combined, runErr := cmd.CombinedOutput()
	if runErr != nil {
		message := strings.TrimSpace(string(combined))
		if len(message) > 1200 {
			message = message[len(message)-1200:]
		}
		return finishMediaFailure(ctx, store, job, fmt.Errorf("ffmpeg failed: %v: %s", runErr, message))
	}
	job.Status = "succeeded"
	job.Error = ""
	if err := store.FinishMediaJob(ctx, *job, output); err != nil {
		return true, err
	}
	log.Printf("media render succeeded job=%s clip=%s output=%s", job.ID, clip.ID, output)
	return true, nil
}

func finishMediaFailure(ctx context.Context, store *storage.Store, job *mediadomain.Job, cause error) (bool, error) {
	job.Status = "failed"
	job.Error = cause.Error()
	if err := store.FinishMediaJob(ctx, *job, ""); err != nil {
		return true, fmt.Errorf("media job failed: %v; recording failure: %w", cause, err)
	}
	log.Printf("media job failed id=%s error=%v", job.ID, cause)
	return true, nil
}

func confinedPath(root, candidate string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	path := candidate
	if !filepath.IsAbs(path) {
		path = filepath.Join(rootAbs, path)
	}
	pathAbs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("media path escapes configured root")
	}
	return pathAbs, nil
}

func videoFilter(aspect string) (string, error) {
	switch aspect {
	case "9:16":
		return "scale=1080:1920:force_original_aspect_ratio=increase,crop=1080:1920", nil
	case "1:1":
		return "scale=1080:1080:force_original_aspect_ratio=increase,crop=1080:1080", nil
	case "16:9":
		return "scale=1920:1080:force_original_aspect_ratio=increase,crop=1920:1080", nil
	default:
		return "", fmt.Errorf("unsupported aspect ratio %q", aspect)
	}
}

func workerEnvOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
