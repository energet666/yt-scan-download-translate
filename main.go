package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	goytdlp "github.com/energet666/goytdlp"
	"github.com/energet666/yt-scan-download-translate/ffmpeg"
	"github.com/energet666/yt-scan-download-translate/votclilive"
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
		shouldStart, err := setupInteractiveConfig()
		if err != nil {
			slog.Error("Interactive setup failed", "error", err)
			os.Exit(1)
		}
		if !shouldStart {
			slog.Info("Setup complete. You can start the service later.")
			os.Exit(0)
		}
		slog.Info("Setup complete. Starting service...")
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

	dir := filepath.Dir(filename)
	base := filepath.Base(filename)
	audioFile := base + ".mp3"
	audioPath := filepath.Join(dir, audioFile)
	translatedVideo := filename + ".[VOT-CLI-LIVE].mp4"

	slog.Info("Downloading translation", "filename", filename)
	err = votclilive.Download(video.Url, dir, audioFile)
	if err != nil {
		return err
	}

	if _, err := os.Stat(audioPath); err == nil {
		slog.Info("Muxing audio with ffmpeg", "output", translatedVideo)
		err = ffmpeg.AddAudio(filename, audioPath, translatedVideo)
		if err != nil {
			return err
		}

		slog.Info("Cleaning up temporary files", "files", []string{audioPath, filename})
		os.Remove(audioPath)
		os.Remove(filename)
	} else {
		return fmt.Errorf("audio file not found: %s", audioPath)
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
	deps := []string{"yt-dlp", "ffmpeg", "vot-cli-live"}
	for _, dep := range deps {
		_, err := exec.LookPath(dep)
		if err != nil {
			return fmt.Errorf("required dependency not found: %s (please install it)", dep)
		}
	}
	return nil
}

func setupInteractiveConfig() (bool, error) {
	scanner := bufio.NewScanner(os.Stdin)
	var playlists []Playlist
	var downloadedVideos []string

	// Загружаем текущий список (он может быть пуст, но файл уже создан из шаблона)
	_ = loadJSON(downloadedPath, &downloadedVideos)

	fmt.Println("")
	fmt.Println("--- Interactive Setup ---")

	yt := goytdlp.NewYtDlp(ytDlpConfPath)

	for {
		fmt.Print("Enter YouTube playlist URL: ")
		if !scanner.Scan() {
			break
		}
		url := strings.TrimSpace(scanner.Text())
		if url == "" {
			fmt.Println("URL cannot be empty, skipping...")
			continue
		}

		fmt.Print("Enable automatic translation (y/n)? [y]: ")
		translate := true
		if scanner.Scan() {
			input := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if input == "n" || input == "no" {
				translate = false
			}
		}

		fmt.Print("Download existing videos in this playlist (y/n)? [y]: ")
		downloadExisting := true
		if scanner.Scan() {
			input := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if input == "n" || input == "no" {
				downloadExisting = false
			}
		}

		if !downloadExisting {
			fmt.Println("Skipping existing videos (scanning playlist)...")
			videos, err := yt.ScanPlaylist(url)
			if err != nil {
				fmt.Printf("Warning: could not scan playlist to skip videos: %v\n", err)
			} else {
				for _, v := range videos {
					if !slices.Contains(downloadedVideos, v.Id) {
						downloadedVideos = append(downloadedVideos, v.Id)
					}
				}
				fmt.Printf("Marked %d videos as already downloaded.\n", len(videos))
			}
		}

		playlists = append(playlists, Playlist{
			PlaylistURL: url,
			Translate:   translate,
		})

		fmt.Print("Add another playlist (y/n)? [n]: ")
		if scanner.Scan() {
			input := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if input != "y" && input != "yes" {
				break
			}
		} else {
			break
		}
	}

	if len(playlists) == 0 {
		return false, fmt.Errorf("no playlists configured")
	}

	// Сохраняем и плейлисты, и список пропущенных видео
	if err := saveJSON(configPath, playlists); err != nil {
		return false, fmt.Errorf("saving config: %w", err)
	}
	if err := saveJSON(downloadedPath, downloadedVideos); err != nil {
		return false, fmt.Errorf("saving downloaded list: %w", err)
	}

	fmt.Printf("Configuration saved with %d playlist(s)\n", len(playlists))

	fmt.Print("Launch the service now (y/n)? [y]: ")
	shouldStart := true
	if scanner.Scan() {
		input := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if input == "n" || input == "no" {
			shouldStart = false
		}
	}

	fmt.Println("-------------------------")
	fmt.Println("")
	return shouldStart, nil
}
