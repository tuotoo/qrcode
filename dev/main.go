package main

import (
	"git.spiritframe.com/tuotoo/utils"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"strconv"
    "github.com/tuotoo/qrcode"
)

var logger = log.New(os.Stdout, "\r\n", log.Ldate|log.Ltime|log.Llongfile)

func main() {
	fi, err := os.Open("qrcode10.png")
	if !check(err) {
		return
	}
	defer fi.Close()
	img, err := png.Decode(fi)
	if !check(err) {
		return
	}
	size := img.Bounds()
	pic := image.NewGray(size)
	draw.Draw(pic, size, img, img.Bounds().Min, draw.Src)

	width := size.Dx()
	height := size.Dy()
	zft := make([]int, 256) //用于保存每个像素的数量，注意这里用了int类型，在某些图像上可能会溢出。
	var idx int
	for i := 0; i < width; i++ {
		for j := 0; j < height; j++ {
			idx = i*height + j
			zft[pic.Pix[idx]]++ //image对像有一个Pix属性，它是一个slice，里面保存的是所有像素的数据。
		}
	}
	var fz uint8 = 128 //uint8(GetOSTUThreshold(zft))
	var m = map[qrcode.Pos]bool{}
	matrix := new(qrcode.Matrix)
	for y := 0; y < height; y++ {
		line := []bool{}
		for x := 0; x < width; x++ {
			if pic.Pix[y*width+x] < fz {
				m[qrcode.Pos{ x, y}] = true
				line = append(line, true)
			} else {
				line = append(line, false)
			}
		}
		matrix.Points = append(matrix.Points, line)
	}
	qrcode.ExportMatrix(size, matrix, "matrix")

	groups := [][]qrcode.Pos{}
	for pos, _ := range m {
		delete(m, pos)
		groups = append(groups, qrcode.SplitGroup(m, pos))
	}
	//计算分组
	c := 0
	for _, group := range groups {
		c += len(group)
	}
	// 判断圈圈
	kong := [][]qrcode.Pos{}
	// 判断实心
	bukong := [][]qrcode.Pos{}
	for _, group := range groups {
		if len(group) == 0 {
			continue
		}
		var groupmap = map[qrcode.Pos]bool{}
		for _, pos := range group {
			groupmap[pos] = true
		}
		minx, maxx, miny, maxy := qrcode.Rectangle(group)
		if qrcode.Kong(groupmap, minx, maxx, miny, maxy) {
			kong = append(kong, group)
		} else {
			bukong = append(bukong, group)
		}
	}
	qrcode.ExportGroups(size, groups, "groups")
	positionDetectionPatterns := [][][]qrcode.Pos{}
	for _, bukonggroup := range bukong {
		for _, konggroup := range kong {
			if qrcode.IsPositionDetectionPattern(bukonggroup, konggroup) {
				positionDetectionPatterns = append(positionDetectionPatterns, [][]qrcode.Pos{bukonggroup, konggroup})
			}
		}
	}
	for i, pattern := range positionDetectionPatterns {
		qrcode.ExportGroups(size, pattern, "positionDetectionPattern"+strconv.FormatInt(int64(i), 10))
	}
	linewidth := qrcode.LineWidth(positionDetectionPatterns)
	pdp := qrcode.NewPositionDetectionPattern(positionDetectionPatterns)
	topstart := &qrcode.Pos{X: pdp.Topleft.Center.X + (int(3.5*linewidth) + 1), Y: pdp.Topleft.Center.Y + int(3*linewidth)}
	topend := &qrcode.Pos{X: pdp.Right.Center.X - (int(3.5*linewidth) + 1), Y: pdp.Right.Center.Y + int(3*linewidth)}
	topTimePattens := qrcode.Line(topstart, topend, matrix)
	topcl := qrcode.Centerlist(topTimePattens, topstart.X)

	leftstart := &qrcode.Pos{X: pdp.Topleft.Center.X + int(3*linewidth), Y: pdp.Topleft.Center.Y + (int(3.5*linewidth) + 1)}
	leftend := &qrcode.Pos{X: pdp.Bottom.Center.X + int(3*linewidth), Y: pdp.Bottom.Center.Y - (int(3.5*linewidth) + 1)}
	leftTimePattens := qrcode.Line(leftstart, leftend, matrix)
	leftcl := qrcode.Centerlist(leftTimePattens, leftstart.Y)

	qrtopcl := []int{}
	for i := -3; i <= 3; i++ {
		qrtopcl = append(qrtopcl, pdp.Topleft.Center.X+int(float64(i)*linewidth))
	}
	qrtopcl = append(qrtopcl, topcl...)
	for i := -3; i <= 3; i++ {
		qrtopcl = append(qrtopcl, pdp.Right.Center.X+int(float64(i)*linewidth))
	}

	qrleftcl := []int{}
	for i := -3; i <= 3; i++ {
		qrleftcl = append(qrleftcl, pdp.Topleft.Center.Y+int(float64(i)*linewidth))
	}
	qrleftcl = append(qrleftcl, leftcl...)
	for i := -3; i <= 3; i++ {
		qrleftcl = append(qrleftcl, pdp.Bottom.Center.Y+int(float64(i)*linewidth))
	}

	qrmatrix := new(qrcode.Matrix)
	for _, y := range qrleftcl {
		line := []bool{}
		for _, x := range qrtopcl {
			line = append(line, matrix.At(x, y))
		}
		qrmatrix.Points = append(qrmatrix.Points, line)
	}
	qrcode.ExportMatrix(image.Rect(0, 0, len(qrtopcl), len(qrleftcl)), qrmatrix, "bitmatrix")
	qrErrorCorrectionLevel, qrMask := qrmatrix.FormatInfo()
	logger.Println("qrErrorCorrectionLevel, qrMask", qrErrorCorrectionLevel, qrMask)
	maskfunc := qrcode.MaskFunc(qrMask)
	unmaskmatrix := new(qrcode.Matrix)
	for y, line := range qrmatrix.Points {
		l := []bool{}
		for x, value := range line {
			l = append(l, maskfunc(x, y) != value)
		}
		unmaskmatrix.Points = append(unmaskmatrix.Points, l)
	}

	//logger.Println(qrmatrix.Points[0])
	//logger.Println(unmaskmatrix.Points[0])
	qrcode.ExportMatrix(image.Rect(0, 0, len(qrtopcl), len(qrleftcl)), unmaskmatrix, "unmaskmatrix")
	dataarea := unmaskmatrix.DataArea()
	qrcode.ExportMatrix(image.Rect(0, 0, len(qrtopcl), len(qrleftcl)), dataarea, "mask")
	//logger.Println(len(qrcode.GetData(unmaskmatrix,dataarea)),qrcode.GetData(unmaskmatrix,dataarea))
	//logger.Println(len(Bool2Byte(qrcode.GetData(unmaskmatrix,dataarea))),Bool2Byte(qrcode.GetData(unmaskmatrix,dataarea)))
	//logger.Println(StringByte(Bool2Byte(qrcode.GetData(unmaskmatrix,dataarea))))
	logger.Println(qrcode.StringBool(qrcode.GetData(unmaskmatrix, dataarea)))
	datacode, errorcode := qrcode.ParseBlock(qrmatrix, qrcode.GetData(unmaskmatrix, dataarea))
	logger.Println(qrcode.StringBool(datacode), qrcode.StringBool(errorcode))
	bt := qrcode.Bits2Bytes(datacode, unmaskmatrix.Version())
	logger.Println(bt)
	logger.Println(string(bt))
}


func check(err error) bool {
	return utils.Check(err)
}
