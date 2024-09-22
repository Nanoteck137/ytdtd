package cli

import (
	"encoding/json"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gosimple/slug"
	"github.com/nanoteck137/ytdtd/utils"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
)

type TrackFile struct {
	Lossless string `toml:"lossless"`
	Lossy    string `toml:"lossy"`
}

type TrackMetadata struct {
	Num       int       `toml:"num"`
	Name      string    `toml:"name"`
	Duration  int       `toml:"duration"`
	Artist    string    `toml:"artist"`
	Year      int       `toml:"year"`
	Tags      []string  `toml:"tags"`
	Genres    []string  `toml:"genres"`
	Featuring []string  `toml:"featuring"`
	File      TrackFile `toml:"file,inline"`
}

type AlbumMetadata struct {
	Album    string          `toml:"album"`
	Artist   string          `toml:"artist"`
	CoverArt string          `toml:"coverart"`
	Tracks   []TrackMetadata `toml:"tracks"`
}

type Info struct {
	Track       string   `json:"track"`
	Album       string   `json:"album"`
	Artists     []string `json:"artists"`
	ReleaseYear int      `json:"release_year"`
	UploadDate  string   `json:"upload_date"`
}

type Track struct {
	Filename string
	Info     Info
}

func GetTracks(dir string) ([]Track, error) {
	var tracks []Track
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		name := d.Name()
		ext := path.Ext(name)
		if ext == ".opus" {
			n := strings.TrimSuffix(name, ext)

			d, err := os.ReadFile(path.Join(dir, n+".info.json"))
			if err != nil {
				return err
			}

			var info Info
			err = json.Unmarshal(d, &info)
			if err != nil {
				return err
			}

			tracks = append(tracks, Track{
				Filename: path.Join(dir, name),
				Info:     info,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return tracks, nil
}

var downloadCmd = &cobra.Command{
	Use: "download",
}

func download(cwd, url, outputTemplate string) error {
	cmd := exec.Command(
		"yt-dlp",
		"-x", "--audio-format", "opus",
		"--embed-metadata", "--embed-thumbnail",
		"--write-info-json",
		"--cookies-from-browser", "firefox",
		"-o", outputTemplate,
		url,
	)
	cmd.Dir = cwd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func downloadSingle(cwd, url string) error {
	return download(cwd, url, "01. %(track)s.%(ext)s")
}

func downloadAlbum(cwd, url string) error {
	return download(cwd, url, "%(playlist_index)s. %(track)s.%(ext)s")
}

func createCoverImage(track, sourceDir, destDir string) error {
	coverOutPath := path.Join(sourceDir, "cover.png")
	cmd := exec.Command("ffmpeg", "-y", "-i", track, "-frames:v", "1", coverOutPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return err
	}

	cmd = exec.Command("magick", coverOutPath, "-gravity", "Center", "-extent", "1:1", path.Join(destDir, "cover.png"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func createAlbum(albumName string, tracks []Track, srcDir, outputDir string) error {
	track := tracks[0]

	s := slug.Make(albumName)
	dest := path.Join(outputDir, s)

	err := os.Mkdir(dest, 0755)
	if err != nil {
		return err
	}

	err = createCoverImage(track.Filename, srcDir, dest)
	if err != nil {
		return err
	}

	for _, track := range tracks {
		d := path.Join(dest, path.Base(track.Filename))
		_, err := utils.CopyFile(track.Filename, d)
		if err != nil {
			return err
		}
	}

	album := AlbumMetadata{
		Album:    albumName,
		Artist:   track.Info.Artists[0],
		CoverArt: "cover.png",
		Tracks:   []TrackMetadata{},
	}

	for _, track := range tracks {
		year := track.Info.ReleaseYear
		if year == 0 {
			year, _ = strconv.Atoi(track.Info.UploadDate[:4])
		}

		album.Tracks = append(album.Tracks, TrackMetadata{
			Num:       utils.ExtractNumber(path.Base(track.Filename)),
			Name:      track.Info.Track,
			Artist:    track.Info.Artists[0],
			Year:      year,
			Tags:      []string{},
			Genres:    []string{},
			Featuring: track.Info.Artists[1:],
			File: TrackFile{
				Lossy: path.Base(track.Filename),
			},
		})
	}

	d, err := toml.Marshal(album)
	if err != nil {
		return err
	}

	err = os.WriteFile(path.Join(dest, "album.toml"), d, 0644)
	if err != nil {
		return err
	}

	return nil
}

var downloadSingleCmd = &cobra.Command{
	Use:  "single <YT_URL>",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		out, _ := cmd.Flags().GetString("output")
		ytUrl := args[0]

		dname, err := os.MkdirTemp("", "ytdtd")
		if err != nil {
			log.Fatal(err)
		}
		defer os.RemoveAll(dname)

		err = downloadSingle(dname, ytUrl)
		if err != nil {
			log.Fatal(err)
		}

		tracks, err := GetTracks(dname)
		if err != nil {
			log.Fatal(err)
		}

		if len(tracks) <= 0 {
			return
		}

		track := tracks[0]
		err = createAlbum(track.Info.Track, tracks, dname, out)
		if err != nil {
			log.Fatal(err)
		}
	},
}

var downloadAlbumCmd = &cobra.Command{
	Use:  "album <YT_URL>",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		out, _ := cmd.Flags().GetString("output")
		ytUrl := args[0]

		dname, err := os.MkdirTemp("", "ytdtd")
		if err != nil {
			log.Fatal(err)
		}
		defer os.RemoveAll(dname)

		err = downloadAlbum(dname, ytUrl)
		if err != nil {
			log.Fatal(err)
		}

		tracks, err := GetTracks(dname)
		if err != nil {
			log.Fatal(err)
		}

		if len(tracks) <= 0 {
			return
		}

		track := tracks[0]
		err = createAlbum(track.Info.Album, tracks, dname, out)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	downloadSingleCmd.Flags().StringP("output", "o", ".", "Output Directory")
	downloadAlbumCmd.Flags().StringP("output", "o", ".", "Output Directory")

	downloadCmd.AddCommand(downloadSingleCmd)
	downloadCmd.AddCommand(downloadAlbumCmd)

	rootCmd.AddCommand(downloadCmd)
}
