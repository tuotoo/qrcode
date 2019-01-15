package qrcode

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		in, out string
	}{
		{in: "qrcode.jpg", out: "http://www.imdb.com/title/tt2948356/"},
		{in: "qrcode.png", out: "http://weixin.qq.com/r/2fKmvj-EkmLtrXvd96fL"},
		{in: "qrcode1.png", out: "http://weixin.qq.com/r/2fKmvj-EkmLtrXvd96fL"},
		{in: "qrcode4.png", out: "http://www.example.org"},
		{in: "qrcode5.png", out: "a"},
		{in: "qrcode6.png", out: "abcdefg"},
		{in: "qrcode7.png", out: "abcdefg"},
		{in: "qrcode8.png", out: "中文"},
		{in: "qrcode9.png", out: "abcdefg"},
		{in: "qrcode10.png", out: "abcdefghijklmnopqrstuvwxyz"},
		{in: "qrcode11.png", out: `PProf是一个CPU分析器( cpu profiler)， 它是gperftools工具的一个组件， 由Google工程师为分析多线程的程序所开发。
Go标准库中的pprof package通过HTTP的方式为pprof工具提供数据。
(译者注：不止这个包，runtime/pprof还可以为控制`},
		{in: "qrcode13.png", out: `PProf是一个CPU分析器( cpu profiler)， 它是gperftools工具的一个组件， 由Google工程师为分析多线程的程序所开发。
Go标准库中的pprof package通过HTTP的方式为pprof工具提供数据。
(译者注：不止这个包，runtime/pprof还可以为控制`},
	}
	for _, tt := range tests {
		f, err := os.Open(filepath.Join("example", tt.in))
		if err != nil {
			t.Fatal(err)
		}
		startAt := time.Now()
		qr, err := Decode(f)
		if err != nil {
			t.Fatal(err)
		}
		if qr.Content != tt.out {
			t.Errorf("expect: %s, got: %s", tt.out, qr.Content)
		}
		t.Log(time.Since(startAt))
	}
}
