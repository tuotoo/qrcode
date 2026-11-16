# qrcode

[![Go Reference](https://pkg.go.dev/badge/github.com/tuotoo/qrcode.svg)](https://pkg.go.dev/github.com/tuotoo/qrcode)
[![Go Version](https://img.shields.io/github/go-mod/go-version/tuotoo/qrcode)](go.mod)
[![License: GPL-3.0](https://img.shields.io/badge/license-GPL--3.0-blue)](LICENSE)

A pure-Go **QR code decoder**: hand it an image and it returns the decoded payload.
No CGO, no external binaries, no separate image pipeline — image loading comes from
the standard library, error correction from a small Reed–Solomon package.

English | [简体中文](#中文速览)

## Features

- **Real-world images** — screenshots, camera photos, screen captures with logos/glare, JPEG artifacts.
- **Every version and EC level** — QR versions 1–40, error-correction levels L/M/Q/H.
- **Every data mode** — numeric, alphanumeric, 8-bit byte, and Kanji (Shift JIS decoded to UTF-8).
- **Rotation** — arbitrary rotation handled by affine sampling anchored on the three finder patterns.
- **Perspective** — homography sampling from the alignment pattern (versions ≥ 2); version 1 falls back to affine.
- **Adaptive binarization** — Otsu threshold computed per image, so uneven lighting needs no tuning.
- **Error correction** — block deinterleaving plus Reed–Solomon correction.
- **Inverted and dot-style symbols** — light-on-dark and non-connected ("art") codes decode through a scanline 1:1:3:1:1 fallback.
- **Fast on large photos** — finders are detected on a downscaled grid while modules are sampled from full-resolution pixels.

## Installation

```bash
go get github.com/tuotoo/qrcode
```

Requires Go 1.27 or newer.

## Quick start

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/tuotoo/qrcode"
)

func main() {
	f, err := os.Open("qrcode1.png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	m, err := qrcode.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(m.Content)         // decoded payload
	fmt.Println(m.Version())       // symbol version, 1-40
}
```

`Decode` accepts anything readable and decodable by Go's `image` package with PNG and
JPEG registered (this package imports both for side effects).

If you already hold an `image.Image` — a decoded frame, a cropped region, a canvas —
skip the file handling:

```go
m, err := qrcode.DecodeImg(img)
if err != nil {
	log.Fatal(err)
}
fmt.Println(m.Content)
```

For other formats (GIF, WebP, BMP, …), decode the image yourself and call `DecodeImg`.

### Run the bundled example

```bash
cd example
go run .
```

The example reads `qrcode15.jpeg` from its own directory, so run it from `example/`.

## API

| Symbol | Description |
| --- | --- |
| `qrcode.Decode(r io.Reader) (*Matrix, error)` | Decode a QR code from an encoded image (PNG/JPEG). |
| `qrcode.DecodeImg(img image.Image) (*Matrix, error)` | Decode a QR code from an already-decoded image. |
| `(*Matrix).Content` | Decoded payload as a string. |
| `(*Matrix).Version()` | Symbol version (1–40), derived from the sampled grid. |
| `(*Matrix).FormatInfo()` | Error-correction level and mask pattern; level values follow ISO/IEC 18004 (0 = M, 1 = L, 2 = H, 3 = Q). |
| `(*Matrix).Points`, `(*Matrix).Size` | Sampled module grid and its dimensions, useful for debugging or re-rendering. |
| `qrcode.QRReconstruct(data, ecc []byte) ([]byte, error)` | Correct data codewords with the supplied ECC codewords. |

Lower-level building blocks (`NewPositionDetectionPattern`, `MaskFunc`, `ParseBlock`,
`Bits2Bytes`, …) are exported as well; they are mainly useful when experimenting with
the detection pipeline and are not covered by compatibility guarantees.

## How it works

1. **Binarize** — the image is converted to grayscale and thresholded with Otsu's method.
2. **Locate finders** — two complementary strategies:
   - *Connected components*: a 3×3 solid core inside a 7×7 hollow ring, with an area-ratio sanity check.
   - *Scanline*: run-length scans along rows and columns matching the 1:1:3:1:1 ratio, independent of connectedness and global polarity.
3. **Sample modules** — three finder centers define an affine frame (handles rotation);
   when an alignment pattern is found, four points define a homography (handles perspective).
4. **Read and correct** — format information is BCH-validated (both copies), the mask
   is removed, codewords are read in the standard zig-zag order, deinterleaved per
   block, and Reed–Solomon corrected.
5. **Decode content** — mode indicators select the numeric, alphanumeric, byte, or Kanji decoder.

For tough inputs the detector walks a fallback ladder and stops at the first success:
downscaled grid → downscaled + eroded → full resolution → full resolution + eroded →
scanline. Erosion only affects detection; sampled modules always come from grayscale
pixels, so it never damages the payload.

## Supported input and limitations

- **Decoder only** — this package does not generate QR codes.
- **One symbol per image** — multi-code scenes and structured-append sequences are not reassembled.
- **No ECI handling** — byte-mode payloads are returned verbatim. Bytes are converted to
  `string` as-is, so interpreting non-UTF-8 text (or binary data) is up to the caller.
- **Kanji mode** assumes Shift JIS, as specified by the standard.
- **Dot-style codes under strong perspective** — the scanline fallback samples with three
  finder patterns only (no alignment pattern), so very high-version art codes that are also
  heavily keystoned may fail. Regular solid-module codes use the full homography path.

## Performance

Detection runs on a downscaled binary grid when the long edge exceeds ~1000 px
(integer factor, capped at 4), then modules are sampled from the full-resolution
grayscale image. This keeps large photos fast without sacrificing accuracy.

End-to-end timings are dominated by image decoding (PNG inflate / JPEG), not by QR logic.
Two benchmarks separate the layers:

```bash
go test -run '^$' -bench 'BenchmarkDecodeImg' -benchmem  # library only, image pre-decoded
go test -run '^$' -bench 'BenchmarkDecode$' -benchmem    # full pipeline
```

## Testing

```bash
go test ./...
```

`example/` doubles as the regression corpus and covers:

- plain screenshots and high-contrast renders,
- rotated and perspective-distorted photos,
- a screen capture with logo noise and glare (`qrcode23.png`),
- an inverted, dot-style art code (`qrcode12.png`),
- all four data modes, including Chinese text and Kanji.

Expected content for each image lives in `decodeCases` in `qrcode_test.go`, which is
shared by the tests and `BenchmarkDecode`. To add a case: drop the image into
`example/`, add its expected payload to `decodeCases`, and run `go test ./...`.

## Reference documents

`doc/` contains the material this implementation was written against:

- `qr_standard.pdf` — ISO/IEC 18004, the QR code specification.
- `bch.pdf` — BCH coding, used by the format-information checks.
- `基于边缘增强的二维码区域快速检测定位算法.pdf` — edge-enhanced QR region detection.

## Contributing

Issues and pull requests are welcome. Before submitting:

```bash
gofmt -l .
go vet ./...
go test ./...
```

Keep new test images in `example/` with their expected payload in `decodeCases`, and
note in the PR which detection path your change affects.

## License

[GNU General Public License v3.0](LICENSE).

---

## 中文速览

纯 Go 实现的 **二维码解码库**，输入图片，输出内容。不依赖 CGO，也不需要外部可执行文件。

### 安装

```bash
go get github.com/tuotoo/qrcode
```

需要 Go 1.27 及以上版本。

### 用法

```go
f, err := os.Open("qrcode1.png")
if err != nil {
	log.Fatal(err)
}
defer f.Close()

m, err := qrcode.Decode(f)
if err != nil {
	log.Fatal(err)
}
fmt.Println(m.Content)   // 解码内容
fmt.Println(m.Version()) // 版本号 1-40
```

已经拿到 `image.Image` 时（视频帧、裁剪区域、画布）直接调用 `qrcode.DecodeImg(img)`；
GIF/WebP/BMP 等格式请自行解码后再走 `DecodeImg`。

### 能力

- 支持版本 1–40、纠错等级 L/M/Q/H；
- 支持数字、字母数字、8-bit 字节、汉字（Shift JIS 转 UTF-8）四种编码模式；
- 支持旋转（三点仿射采样）与透视畸变（对齐图案 + 单应矩阵）；
- 大津法自适应二值化，适应光照不均；
- Reed–Solomon 纠错；
- 支持反色、圆点式艺术二维码（扫描线 1:1:3:1:1 回退路径）；
- 大图先下采样定位、再全分辨率采样，兼顾速度与精度。

检测失败时会依次回退：下采样 → 下采样+腐蚀 → 全分辨率 → 全分辨率+腐蚀 → 扫描线法。

### 限制

- 只解码，不生成二维码；
- 一张图只处理一个二维码，不支持多码检测与 structured append 拼接；
- 不处理 ECI，字节模式内容按原始字节返回，非 UTF-8 文本需调用方自行转码；
- 汉字模式按标准假定为 Shift JIS；
- 扫描线回退只用三个定位符做仿射采样，高版本 + 强透视的艺术码可能失败（常规实心码走单应矩阵路径）。

### 测试

```bash
go test ./...
go test -run '^$' -bench 'BenchmarkDecodeImg' -benchmem   # 库自身耗时（预解码图像）
go test -run '^$' -bench 'BenchmarkDecode$' -benchmem     # 端到端耗时
```

`example/` 目录同时是回归测试集，预期内容见 `qrcode_test.go` 的 `decodeCases`。
