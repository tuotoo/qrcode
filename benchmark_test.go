package qrcode

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkDecode 对每个示例二维码做端到端解码基准。文件预先读入内存以排除
// 磁盘 IO，聚焦图像解码 + 二维码解码的完整流程（含 PNG/JPEG 解压与定位采样）。
func BenchmarkDecode(b *testing.B) {
	for _, tc := range decodeCases {
		b.Run(tc.in, func(b *testing.B) {
			data, err := os.ReadFile(filepath.Join("example", tc.in))
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				qr, err := Decode(bytes.NewReader(data))
				if err != nil {
					b.Fatalf("%s: %v", tc.in, err)
				}
				if qr.Content != tc.out {
					b.Fatalf("%s: expect %q got %q", tc.in, tc.out, qr.Content)
				}
			}
		})
	}
}
