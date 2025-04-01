package data

import (
	"context"
	"io"
	"os"
	"os/exec"

	"github.com/google/uuid"
)

type AudioProcessor struct {
	cmd    *exec.Cmd
	stdOut io.ReadCloser
	stdErr io.ReadCloser

	tmp *os.File
}

func NewAudioProcessor(ctx context.Context, data []byte) (*AudioProcessor, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg")

	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	tmp, err := os.CreateTemp(os.Getenv("TMP_VOLUME"), uuid.NewString())
	if err != nil {
		return nil, err
	}

	_, err = tmp.Write(data)
	if err != nil {
		return nil, err
	}

	cmd.Args = append(cmd.Args,
		"-i", tmp.Name(),
		"-hide_banner",
		"pipe:1.mp3",
	)

	return &AudioProcessor{
		cmd:    cmd,
		stdErr: stdErr,
		stdOut: stdOut,
		tmp:    tmp,
	}, nil
}

func (ap *AudioProcessor) GetMetadata() (string, error) {
	metadataBytes, err := io.ReadAll(ap.stdErr)
	if err != nil {
		return "", err
	}

	return string(metadataBytes), nil
}

func (ap *AudioProcessor) Start() error {
	return ap.cmd.Start()
}

func (ap *AudioProcessor) Wait() error {
	return ap.cmd.Wait()
}

func (ap *AudioProcessor) Close() error {
	err := os.Remove(ap.tmp.Name())
	if err != nil {
		return err
	}

	return nil
}
