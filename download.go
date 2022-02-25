package main

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
)

func GetMetadata(url string) (*VideoMeta, error) {
	cmd := exec.Command("yt-dlp", url, "-s", "--print-json")
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

func DownloadVideo(src VideoSource, waitC <-chan struct{}, doneC chan<- error) io.Reader {
	cmdParams := []string{src.Url, "-o", "-", "--no-part", "--max-filesize", "50M", "-f", "[filesize<50M]"}
	if src.Type == MusicType {
		cmdParams = append(cmdParams, "-f", "ba", "--audio-format", "mp3")
	}

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

func DownloadVideoRoutine(src VideoSource) (io.Reader, chan<- struct{}, <-chan error) {
	doneC := make(chan error)
	waitC := make(chan struct{})
	return DownloadVideo(src, waitC, doneC), waitC, doneC
}
