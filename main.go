package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"slices"
	"syscall"
	"time"

	ffmpeg "github.com/energet666/goffmpeg"
	votcli "github.com/energet666/govotcli"
	goytdlp "github.com/energet666/goytdlp"
)

type Playlist struct {
	PlaylistURL string `json:"playlistURL"`
	Translate   bool   `json:"translate"`
}

const (
	configPath     = ".private/config.json"
	downloadedPath = ".private/downloaded.json"
	ytDlpConfPath  = ".private/yt-dlp.conf"
)

func main() {
	// Настраиваем красивый текстовый логгер
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	if err := checkDependencies(); err != nil {
		slog.Error("Dependencies check failed", "error", err)
		os.Exit(1)
	}

	created, err := createConfigsIfNotExist()
	if err != nil {
		slog.Error("Critical error creating configs", "error", err)
		os.Exit(1)
	}
	if created {
		slog.Info("First run: configuration files created in .private/ directory")
		slog.Info("Please edit .private/config.json and .private/yt-dlp.conf, then restart the service")
		os.Exit(0)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("Service started", "interval", "1 minute")

	for {
		if err := runScanCycle(ctx); err != nil {
			slog.Error("Error in scan cycle", "error", err)
		}

		select {
		case <-time.After(time.Minute):
		case <-ctx.Done():
			slog.Info("Shutting down gracefully...")
			return
		}
	}
}

func runScanCycle(ctx context.Context) error {
	var playlists []Playlist
	if err := loadJSON(configPath, &playlists); err != nil {
		return err
	}

	var downloadedVideos []string
	if err := loadJSON(downloadedPath, &downloadedVideos); err != nil {
		return err
	}

	yt := goytdlp.NewYtDlp(ytDlpConfPath)

	for _, playlist := range playlists {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if playlist.PlaylistURL == "" {
			slog.Warn("Skipping empty PlaylistURL")
			continue
		}

		videos, err := yt.ScanPlaylist(playlist.PlaylistURL)
		if err != nil {
			slog.Error("Error scanning playlist", "url", playlist.PlaylistURL, "error", err)
			continue
		}

		for _, video := range videos {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			if slices.Contains(downloadedVideos, video.Id) {
				continue
			}

			slog.Info("Processing new video",
				"title", video.Title,
				"id", video.Id,
				"translate", playlist.Translate)

			if err := handleVideo(yt, video, playlist.Translate); err != nil {
				slog.Error("Error processing video", "id", video.Id, "error", err)
				continue
			}

			downloadedVideos = append(downloadedVideos, video.Id)
			if err := saveJSON(downloadedPath, downloadedVideos); err != nil {
				slog.Error("Error saving downloaded list", "error", err)
			}
		}
	}

	return nil
}

func handleVideo(yt *goytdlp.YtDlp, video goytdlp.Video, translate bool) error {
	err := yt.Download(video.Url)
	if err != nil {
		return err
	}

	if !translate {
		return nil
	}

	filename, err := yt.GetFilename(video.Url)
	if err != nil {
		return err
	}

	audioFile := filename + ".mp3"
	translatedVideo := filename + ".[VOT-CLI].mp4"

	slog.Info("Downloading translation", "filename", filename)
	err = votcli.Download(video.Url, ".", audioFile)
	if err != nil {
		return err
	}

	if _, err := os.Stat(audioFile); err == nil {
		slog.Info("Muxing audio with ffmpeg", "output", translatedVideo)
		err = ffmpeg.AddAudio(filename, audioFile, translatedVideo)
		if err != nil {
			return err
		}

		slog.Info("Cleaning up temporary files", "files", []string{audioFile, filename})
		os.Remove(audioFile)
		os.Remove(filename)
	} else {
		return fmt.Errorf("audio file not found: %s", audioFile)
	}

	return nil
}

func loadJSON(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func saveJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func checkDependencies() error {
	deps := []string{"yt-dlp", "ffmpeg", "vot-cli"}
	for _, dep := range deps {
		_, err := exec.LookPath(dep)
		if err != nil {
			return fmt.Errorf("required dependency not found: %s (please install it)", dep)
		}
	}
	return nil
}
