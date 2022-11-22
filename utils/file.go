package utils

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
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
	bytes, err = ioutil.ReadFile(file)
	data = string(bytes)
	return
}

func DownloadFile(ctx context.Context, filepath string, url string) error {
	var err error
	var resp *http.Response
	span := sentry.StartSpan(ctx, "file.download")
	span.Description = fmt.Sprintf("path: %s\nurl: %s", filepath, url)
	defer func() {
		tracing.FinishSpan(span, err)
	}()

	resp, err = http.Get(url)
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
