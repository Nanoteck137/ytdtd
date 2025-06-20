package cli

import (
	"encoding/json"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/gosimple/slug"
	"github.com/nanoteck137/ytdtd/utils"
	"github.com/spf13/cobra"
)

type Info struct {
	Title       string   `json:"title"`
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
	exe := "yt-dlp"
	override := os.Getenv("YT_DLP_OVERRIDE")

	if override != "" {
		exe = override
	}

	cmd := exec.Command(
		exe,
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

func downloadVideo(cwd, url string) error {
	return download(cwd, url, "01. %(title)s.%(ext)s")
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

	return nil
}

var downloadVideoCmd = &cobra.Command{
	Use:  "video <YT_URL>",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		out, _ := cmd.Flags().GetString("output")
		ytUrl := args[0]

		dname, err := os.MkdirTemp("", "ytdtd")
		if err != nil {
			log.Fatal(err)
		}
		defer os.RemoveAll(dname)

		err = downloadVideo(dname, ytUrl)
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
		name := track.Info.Track
		if name == "" {
			name = track.Info.Title
		}

		err = createAlbum(name, tracks, dname, out)
		if err != nil {
			log.Fatal(err)
		}
	},
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
		name := track.Info.Track
		if name == "" {
			ext := path.Ext(track.Filename)
			name = strings.TrimSuffix(track.Filename, ext)
		}

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
	downloadVideoCmd.Flags().StringP("output", "o", ".", "Output Directory")
	downloadSingleCmd.Flags().StringP("output", "o", ".", "Output Directory")
	downloadAlbumCmd.Flags().StringP("output", "o", ".", "Output Directory")

	downloadCmd.AddCommand(downloadVideoCmd)
	downloadCmd.AddCommand(downloadSingleCmd)
	downloadCmd.AddCommand(downloadAlbumCmd)

	rootCmd.AddCommand(downloadCmd)
}
