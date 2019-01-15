package qrcode

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDecode(t *testing.T) {
	err := filepath.Walk("example", func(path string, info os.FileInfo, err error) error {
		fileExt := filepath.Ext(path)
		if fileExt == ".jpg" || fileExt == ".png" {
			defer func() {
				if err := recover(); err != nil {
					t.Log("recover", path, err)
				}
			}()
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					t.Log("file close error", err)
				}
			}()
			startAt := time.Now()
			qr, err := Decode(f)
			if err != nil {
				t.Log("decode err", path, err)
				return nil
			}
			t.Log(path, qr.Content)
			t.Log(time.Since(startAt))
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}
