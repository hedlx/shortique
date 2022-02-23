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

func DownloadVideo(url string, waitC <-chan struct{}, doneC chan<- error) io.Reader {
	cmd := exec.Command("yt-dlp", url, "-o", "-", "--no-part", "--max-filesize", "50M", "-f", "[filesize<50M]")
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

func DownloadVideoRoutine(url string) (io.Reader, chan<- struct{}, <-chan error) {
	doneC := make(chan error)
	waitC := make(chan struct{})
	return DownloadVideo(url, waitC, doneC), waitC, doneC
}
