package data

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

const (
	DefaultImageExtension = "png"
	DefaultImageMIMEType  = "image/png"
	ignoreError           = "broken pipe"
)

type VideoProcessor struct {
	cmd    *exec.Cmd
	stdIn  io.WriteCloser
	stdOut io.ReadCloser
	stdErr io.ReadCloser
}

func NewVideoProcessor() (*VideoProcessor, error) {
	cmd := exec.Command("ffmpeg",
		"-i", "pipe:",
		"-ss", "00:00:00",
		"-frames:v", "1",
		"-hide_banner",
		//"-report",
		fmt.Sprintf("pipe:1.%s", DefaultImageExtension),
	)
	stdIn, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	return &VideoProcessor{
		cmd:    cmd,
		stdIn:  stdIn,
		stdErr: stdErr,
		stdOut: stdOut,
	}, nil
}

func (vp *VideoProcessor) ProcessVideo(data []byte) error {
	_, err := vp.stdIn.Write(data)
	if err != nil {
		if !strings.Contains(err.Error(), ignoreError) {
			return err
		}
	}

	return nil
}

func (vp *VideoProcessor) GetThumbnail() (*Image, error) {
	thumnailImageData, err := io.ReadAll(vp.stdOut)
	if err != nil {
		return nil, err
	}

	return &Image{
		Data:      thumnailImageData,
		Extension: DefaultImageExtension,
		MimeType:  DefaultImageMIMEType,
	}, err
}

func (vp *VideoProcessor) GetMetadata() (string, error) {
	metadataBytes, err := io.ReadAll(vp.stdErr)
	if err != nil {
		return "", err
	}

	return string(metadataBytes), nil
}

func (vp *VideoProcessor) Start() error {
	return vp.cmd.Start()
}

func (vp *VideoProcessor) Wait() error {
	return vp.cmd.Wait()
}

func ExtractDurationFromMetadata(metadata string) (float32, error) {
	for scanner := bufio.NewScanner(strings.NewReader(metadata)); scanner.Scan(); {
		if strings.Contains(scanner.Text(), "Duration") {
			line, ok := strings.CutPrefix(scanner.Text(), "  Duration: ")
			if !ok {
				return 0, fmt.Errorf("metadata parsing error")
			}
			lines := strings.Split(line, ",")
			if len(lines) == 0 {
				return 0, fmt.Errorf("metadata parsing error")
			}

			t, err := time.Parse("15:04:05.9", lines[0])
			if err != nil {
				return 0, fmt.Errorf("metadata parsing error")
			}

			defaultTime, _ := time.Parse("2006-01-02 15:04:05", "0000-01-01 00:00:00")

			return float32(t.Sub(defaultTime).Seconds()), nil
		}
	}

	return 0, nil
}
