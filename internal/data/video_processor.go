package data

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultImageExtension = "png"
	DefaultImageMIMEType  = "image/png"
	ignoreError           = "broken pipe"
)

type VideoProcessor struct {
	cmd    *exec.Cmd
	stdOut io.ReadCloser
	stdErr io.ReadCloser
}

func NewVideoProcessor(ctx context.Context) (*VideoProcessor, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg")

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
		stdErr: stdErr,
		stdOut: stdOut,
	}, nil
}

func (vp *VideoProcessor) GetThumbnailGenerator(data []byte) (*ThumbnailGenerator, error) {
	tmp, err := os.CreateTemp(os.Getenv("TMP_VOLUME"), uuid.NewString())
	if err != nil {
		return nil, err
	}

	tg := &ThumbnailGenerator{
		vp:  vp,
		tmp: tmp,
	}

	_, err = tmp.Write(data)
	if err != nil {
		return nil, err
	}

	tg.vp.cmd.Args = append(tg.vp.cmd.Args,
		"-i", tmp.Name(),
		"-ss", "00:00:00",
		"-frames:v", "1",
		"-hide_banner",
		//"-report",
		fmt.Sprintf("pipe:1.%s", DefaultImageExtension))

	return tg, nil
}

type ThumbnailGenerator struct {
	vp *VideoProcessor

	tmp *os.File
}

func (tg *ThumbnailGenerator) GetThumbnail() (*Image, error) {
	thumnailImageData, err := io.ReadAll(tg.vp.stdOut)
	if err != nil {
		return nil, err
	}

	return &Image{
		Data:      thumnailImageData,
		Extension: DefaultImageExtension,
		MimeType:  DefaultImageMIMEType,
	}, nil
}

func (tg *ThumbnailGenerator) GetMetadata() (string, error) {
	metadataBytes, err := io.ReadAll(tg.vp.stdErr)
	if err != nil {
		return "", err
	}

	return string(metadataBytes), nil
}

func (tg *ThumbnailGenerator) Start() error {
	return tg.vp.cmd.Start()
}

func (tg *ThumbnailGenerator) Wait() error {
	return tg.vp.cmd.Wait()
}

func (tg *ThumbnailGenerator) Close() error {
	err := os.Remove(tg.tmp.Name())
	if err != nil {
		return err
	}

	return nil
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
