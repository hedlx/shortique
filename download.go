package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

func GetMetadata(url string, origin string) (*VideoMeta, error) {
	if origin != "youtube" {
		return &VideoMeta{}, nil
	}

	L.Info("Getting metadata", zap.String("url", url))
	params := []string{url, "-s", "--print-json"}

	cmd := exec.Command("yt-dlp", params...)
	var output bytes.Buffer
	cmd.Stdout = &output

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return ParseMeta(output.Bytes())
}

type VideoSource struct {
	Url  string
	Type int
}

type DownloadInput struct {
	Origin string
	Src    VideoSource
}

type downloader = func(in DownloadInput, waitC <-chan struct{}, doneC chan<- error) io.Reader

var downloadStrategy = map[string]downloader{
	"youtube":   downloadYT,
	"instagram": downloadInsta,
}

func DownloadVideo(in DownloadInput, waitC <-chan struct{}, doneC chan<- error) io.Reader {
	download := downloadStrategy[in.Origin]
	if download == nil {
		doneC <- fmt.Errorf("unknown origin: %s", in.Origin)
		return bytes.NewReader(nil)
	}

	return download(in, waitC, doneC)
}

func downloadInsta(in DownloadInput, waitC <-chan struct{}, doneC chan<- error) io.Reader {
	parts := strings.Split(in.Src.Url, "/")
	if len(parts) == 0 {
		doneC <- fmt.Errorf("invalid URL: %s", in.Src.Url)
		return bytes.NewReader(nil)
	}
	id := parts[len(parts)-1]

	resp, err := http.Get(fmt.Sprintf("https://www.ddinstagram.com/videos/%s/1", id))
	if err != nil {
		doneC <- err
		return bytes.NewReader(nil)
	}

	if resp.StatusCode > 299 || resp.StatusCode < 200 {
		resp.Body.Close()
		doneC <- fmt.Errorf("bad response: %d", resp.StatusCode)
		return bytes.NewReader(nil)
	}

	go func() {
		defer resp.Body.Close()
		<-waitC
		doneC <- nil
	}()

	return resp.Body
}

func downloadYT(in DownloadInput, waitC <-chan struct{}, doneC chan<- error) io.Reader {
	L.Info("Starting download video")
	cmdParams := []string{in.Src.Url, "-o", "-", "--no-part"}
	cmdParams = append(cmdParams, "--max-filesize", "50M", "-f", "[filesize_approx<50M]")

	if in.Src.Type == MusicType {
		cmdParams = append(cmdParams, "-f", "ba", "--audio-format", "mp3")
	}

	L.Debug("Executing command", zap.String("cmd", strings.Join(append([]string{"yt-dlp"}, cmdParams...), " ")))
	cmd := exec.Command("yt-dlp", cmdParams...)
	out, err := cmd.StdoutPipe()
	if err != nil {
		doneC <- err
		return nil
	}

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	go func() {
		if err := cmd.Start(); err != nil {
			doneC <- err
			return
		}

		<-waitC
		if err := cmd.Wait(); err != nil {
			doneC <- fmt.Errorf(errBuf.String())
			return
		}

		doneC <- nil
	}()

	return out
}

func DownloadVideoRoutine(src VideoSource, origin string) (io.Reader, chan<- struct{}, <-chan error) {
	doneC := make(chan error, 1)
	waitC := make(chan struct{})
	return DownloadVideo(DownloadInput{Src: src, Origin: origin}, waitC, doneC), waitC, doneC
}
