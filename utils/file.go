package utils

import (
	"context"
	"io"
	"net/http"
	"os"

	"github.com/gempages/go-helper/tracing"
	"github.com/getsentry/sentry-go"
)

func CloseFile(f *os.File) {
	err := f.Close()
	if err != nil {
		panic(err)
	}
}

func ReadFile(file string) (data string, err error) {
	var bytes []byte
	bytes, err = os.ReadFile(file)
	data = string(bytes)
	return
}

func DownloadFile(ctx context.Context, filepath string, url string) error {
	var err error

	span := sentry.StartSpan(ctx, "shopify.download_file")
	span.Description = url
	defer func() {
		tracing.FinishSpan(span, err)
	}()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer CloseFile(out)

	_, err = io.Copy(out, resp.Body)
	return err
}
