package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"slices"
	"time"

	ffmpeg "github.com/energet666/goffmpeg"
	votcli "github.com/energet666/govotcli"
	goytdlp "github.com/energet666/goytdlp"
)

type Playlist struct {
	PlaylistURL string `json:"playlistURL"`
	Translate   bool   `json:"translate"`
}

func main() {
	err := createConfigsIfNotExist()
	if err != nil {
		panic(err)
	}

	playlists := new([]Playlist)
	playlistsJSON, err := os.ReadFile(".private/config.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(playlistsJSON, playlists)
	if err != nil {
		panic(err)
	}

	downloadedVideos := new([]string)
	downloadedJSON, err := os.ReadFile(".private/downloaded.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(downloadedJSON, downloadedVideos)
	if err != nil {
		panic(err)
	}

	yt := goytdlp.NewYtDlp(".private/yt-dlp.conf")

	for {
		downloadedFileNeedUpdate := false

		for _, playlist := range *playlists {
			if playlist.PlaylistURL == "" {
				log.Println("PlaylistURL is empty")
				continue
			}

			videos, err := yt.ScanPlaylist(playlist.PlaylistURL)
			if err != nil {
				log.Println("error scanning playlist:", err)
				time.Sleep(time.Second * 10)
				continue
			}

			for _, video := range videos {
				if slices.Contains(*downloadedVideos, video.Id) {
					continue
				}
				fmt.Println("New video:", video.Title)
				fmt.Println("\t", video.Url)
				fmt.Println("\t", video.Id)

				err := yt.Download(video.Url)
				if err != nil {
					log.Println("downloading video error:", err)
					continue
				}

				if playlist.Translate {
					filename, err := yt.GetFilename(video.Url)
					err = votcli.Download(video.Url, ".", filename+".mp3")
					if err != nil {
						log.Println("[vot-cli]", err)
						continue
					}

					if _, err := os.Stat(filename + ".mp3"); err == nil {
						log.Println(filename+".mp3", "is exist")
						err = ffmpeg.AddAudio(filename, filename+".mp3", filename+".[VOT-CLI].mp4")
						if err != nil {
							log.Println("[ffmpeg]", err)
							continue
						}
					} else {
						log.Println(filename+".mp3", "is not exist")
					}
				}

				*downloadedVideos = append(*downloadedVideos, video.Id)
				downloadedFileNeedUpdate = true
			}
		}

		if downloadedFileNeedUpdate {
			data, err := json.Marshal(downloadedVideos)
			if err != nil {
				log.Println("marshaling downloaded list error:", err)
			}
			err = os.WriteFile(".private/downloaded.json", data, 0644)
			if err != nil {
				log.Println("writing downloaded list to file error:", err)
			}
		}

		time.Sleep(time.Second * 60)
	}
}
