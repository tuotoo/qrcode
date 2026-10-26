package qrcode

import (
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/text/encoding/japanese"
)

func TestBinarization(t *testing.T) {
	gray := image.NewGray(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			if x < 50 {
				gray.Pix[y*100+x] = 20
			} else {
				gray.Pix[y*100+x] = 220
			}
		}
	}
	mx := &Matrix{}
	threshold := mx.Binarization(gray)
	if threshold <= 20 || threshold >= 220 {
		t.Errorf("大津法阈值 %d 应位于两级灰度 20 与 220 之间", threshold)
	}
}

func bits(s string) []bool {
	var b []bool
	for _, c := range s {
		b = append(b, c == '1')
	}
	return b
}

func TestNumericDecoder(t *testing.T) {
	d := &NumericDecoder{countIndicator: 10}

	// "123456" = 123(10 bit) + 456(10 bit)，count=6
	data := append(bits("0000000110"), bits("0001111011")...)
	data = append(data, bits("0111001000")...)
	out, err := d.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "123456" {
		t.Errorf("expect 123456, got %q", string(out))
	}

	// "12345" = 123(10 bit) + 45(7 bit)，count=5（奇数，最后 2 位 7 bit）
	d2 := &NumericDecoder{countIndicator: 10}
	data2 := append(bits("0000000101"), bits("0001111011")...)
	data2 = append(data2, bits("0101101")...)
	out2, err := d2.Decode(data2)
	if err != nil {
		t.Fatal(err)
	}
	if string(out2) != "12345" {
		t.Errorf("expect 12345, got %q", string(out2))
	}
}

func TestKanjiDecoder(t *testing.T) {
	// 用 japanese 编码器把 UTF-8 汉字转成 Shift JIS，再按 QR Kanji 规则构造 13 bit。
	sjis, err := japanese.ShiftJIS.NewEncoder().Bytes([]byte("点"))
	if err != nil {
		t.Fatal(err)
	}
	code := int(sjis[0])<<8 | int(sjis[1])
	var diff int
	if code >= 0x8140 && code <= 0x9FFC {
		diff = code - 0x8140
	} else {
		diff = code - 0xC140
	}
	// ISO 18004 7.4.6: 13 bit 值 = (diff 高字节)*0xC0 + diff 低字节。
	v := (diff>>8)*0xC0 + (diff & 0xFF)

	// 构造 data：count=1 (8 bit) + 13 bit 值。
	d := &KanjiDecoder{countIndicator: 8}
	var data []bool
	for i := 7; i >= 0; i-- {
		data = append(data, (1>>uint(i))&1 == 1)
	}
	for i := 12; i >= 0; i-- {
		data = append(data, (v>>uint(i))&1 == 1)
	}

	out, err := d.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "点" {
		t.Errorf("expect 点, got %q", string(out))
	}
}

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
		{in: "qrcode14.jpeg", out: "AEL-10007-78402-01XXB45EBF1163C414B24AFD062B008024605AA3AB554463147C78A4B0ECA23B1DA80"},
		{in: "qrcode15.jpeg", out: "AEL-10007-78379-02XX524DBEEF63C414A830F3062A0047E2404ECEAF6E8C1DCCF9E0ED2484355C22EF0"},
		{in: "qrcode16.jpeg", out: "AEL-10007-78379-02XX524DBEEF63C414A830F3062A0047E2404ECEAF6E8C1DCCF9E0ED2484355C22EF0"},
		{in: "qrcode17.jpeg", out: "AEL-10007-78379-02XX524DBEEF63C414A830F3062A0047E2404ECEAF6E8C1DCCF9E0ED2484355C22EF0"},
		{in: "qrcode18.jpeg", out: "AEL-10007-78379-02XX524DBEEF63C414A830F3062A0047E2404ECEAF6E8C1DCCF9E0ED2484355C22EF0"},
		{in: "qrcode19.jpeg", out: "AEL-10007-78379-02XX524DBEEF63C414A830F3062A0047E2404ECEAF6E8C1DCCF9E0ED2484355C22EF0"},
		{in: "qrcode20.png", out: "1234567890"},
		{in: "qrcode21.png", out: "点"},
	}
	for _, tt := range tests {
		f, err := os.Open(filepath.Join("example", tt.in))
		if err != nil {
			t.Fatal(tt.in, err)
		}
		startAt := time.Now()
		qr, err := Decode(f)
		if err != nil {
			t.Fatal(tt.in, err)
		}
		if qr.Content != tt.out {
			t.Errorf("expect: %s, got: %s", tt.out, qr.Content)
		}
		t.Log(tt.in, time.Since(startAt))
	}
}
