package qrcode

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/maruel/rs"
)

type PositionDetectionPatterns struct {
	TopLeft *PosGroup
	Right   *PosGroup
	Bottom  *PosGroup
}

type PosGroup struct {
	Group    []Pos
	GroupMap map[Pos]bool
	Min      Pos
	Max      Pos
	Center   Pos
	Hollow   bool
}

type Matrix struct {
	OrgImage  image.Image
	OrgSize   image.Rectangle
	OrgPoints [][]bool
	Points    [][]bool
	Size      image.Rectangle
	Data      []bool
	Content   string

	// 二值化产生的灰度图与阈值，供后续仿射采样阶段使用。
	grayPix   []uint8
	threshold uint8
}

func (mx *Matrix) AtOrgPoints(x, y int) bool {
	if y >= 0 && y < len(mx.OrgPoints) {
		if x >= 0 && x < len(mx.OrgPoints[y]) {
			return mx.OrgPoints[y][x]
		}
	}
	return false
}

type FormatInfo struct {
	ErrorCorrectionLevel, Mask int
}

func (mx *Matrix) FormatInfo() (*FormatInfo, error) {
	fi1 := []Pos{
		{0, 8}, {1, 8}, {2, 8}, {3, 8},
		{4, 8}, {5, 8}, {7, 8},
		{8, 8}, {8, 7}, {8, 5}, {8, 4},
		{8, 3}, {8, 2}, {8, 1}, {8, 0},
	}
	maskedFileData := mx.GetBin(fi1)
	unmaskFileData := maskedFileData ^ 0x5412
	if bch(unmaskFileData) == 0 {
		return &FormatInfo{
			ErrorCorrectionLevel: unmaskFileData >> 13,
			Mask:                 unmaskFileData >> 10 & 7,
		}, nil
	}
	length := len(mx.Points)
	fi2 := []Pos{
		{8, length - 1}, {8, length - 2}, {8, length - 3}, {8, length - 4},
		{8, length - 5}, {8, length - 6}, {8, length - 7},
		{length - 8, 8}, {length - 7, 8}, {length - 6, 8}, {length - 5, 8},
		{length - 4, 8}, {length - 3, 8}, {length - 2, 8}, {length - 1, 8},
	}
	maskedFileData = mx.GetBin(fi2)
	unmaskFileData = maskedFileData ^ 0x5412
	if bch(unmaskFileData) == 0 {
		return &FormatInfo{
			ErrorCorrectionLevel: unmaskFileData >> 13,
			Mask:                 unmaskFileData >> 10 & 7,
		}, nil
	}
	return nil, errors.New("not found error correction level and mask")
}

func (mx *Matrix) AtPoints(x, y int) bool {
	if y >= 0 && y < len(mx.Points) {
		if x >= 0 && x < len(mx.Points[y]) {
			return mx.Points[y][x]
		}
	}
	return false
}

func (mx *Matrix) GetBin(poss []Pos) int {
	var fileData int
	for _, pos := range poss {
		if mx.AtPoints(pos.X, pos.Y) {
			fileData = fileData<<1 + 1
		} else {
			fileData = fileData << 1
		}
	}
	return fileData
}

func (mx *Matrix) Version() int {
	width := len(mx.Points)
	return (width-21)/4 + 1
}

type Pos struct {
	X int
	Y int
}

func bch(org int) int {
	var g = 0x537
	for i := 4; i > -1; i-- {
		if org&(1<<(uint(i+10))) > 0 {
			org ^= g << uint(i)
		}
	}
	return org
}

func (mx *Matrix) DataArea() *Matrix {
	da := new(Matrix)
	width := len(mx.Points)
	maxPos := width - 1
	for _, line := range mx.Points {
		var l []bool
		for range line {
			l = append(l, true)
		}
		da.Points = append(da.Points, l)
	}
	// Position Detection Pattern是定位图案，用于标记二维码的矩形大小。
	// 这三个定位图案有白边叫Separators for Position Detection Patterns。之所以三个而不是四个意思就是三个就可以标识一个矩形了。
	for y := 0; y < 9; y++ {
		for x := 0; x < 9; x++ {
			if y < len(mx.Points) && x < len(mx.Points[y]) {
				da.Points[y][x] = false // 左上
			}
		}
	}
	for y := 0; y < 9; y++ {
		for x := 0; x < 8; x++ {
			if y < len(mx.Points) && maxPos-x < len(mx.Points[y]) {
				da.Points[y][maxPos-x] = false // 右上
			}
		}
	}
	for y := 0; y < 8; y++ {
		for x := 0; x < 9; x++ {
			if maxPos-y < len(mx.Points) && x < len(mx.Points[y]) {
				da.Points[maxPos-y][x] = false // 左下
			}
		}
	}
	// Timing Patterns也是用于定位的。原因是二维码有40种尺寸，尺寸过大了后需要有根标准线，不然扫描的时候可能会扫歪了。
	for i := 0; i < width; i++ {
		if 6 < len(mx.Points) && i < len(mx.Points[6]) {
			da.Points[6][i] = false
		}
		if i < len(mx.Points) && 6 < len(mx.Points[i]) {
			da.Points[i][6] = false
		}
	}
	// Alignment Patterns 只有Version 2以上（包括Version2）的二维码需要这个东东，同样是为了定位用的。
	version := da.Version()
	Alignments := AlignmentPatternCenter[version]
	for _, AlignmentX := range Alignments {
		for _, AlignmentY := range Alignments {
			if (AlignmentX == 6 && AlignmentY == 6) || (maxPos-AlignmentX == 6 && AlignmentY == 6) || (AlignmentX == 6 && maxPos-AlignmentY == 6) {
				continue
			}
			for y := AlignmentY - 2; y <= AlignmentY+2; y++ {
				for x := AlignmentX - 2; x <= AlignmentX+2; x++ {
					if y < len(mx.Points) && x < len(mx.Points[y]) {
						da.Points[y][x] = false
					}
				}
			}
		}
	}
	// Version Information 在 >= Version 7以上，需要预留两块3 x 6的区域存放一些版本信息。
	if version >= 7 {
		for i := maxPos - 10; i < maxPos-7; i++ {
			for j := 0; j < 6; j++ {
				if i < len(mx.Points) && j < len(mx.Points[i]) {
					da.Points[i][j] = false
				}
				if j < len(mx.Points) && i < len(mx.Points[j]) {
					da.Points[j][i] = false
				}
			}
		}
	}
	return da
}

func NewPositionDetectionPattern(PDPs [][]*PosGroup) (*PositionDetectionPatterns, error) {
	if len(PDPs) < 3 {
		return nil, errors.New("lost Position Detection Pattern")
	}
	var pdpGroups []*PosGroup
	for _, pdp := range PDPs {
		pdpGroups = append(pdpGroups, PossListToGroup(pdp))
	}
	var ks []*K
	for i, firstPDPGroup := range pdpGroups {
		for j, lastPDPGroup := range pdpGroups {
			if i == j {
				continue
			}
			k := &K{FirstPosGroup: firstPDPGroup, LastPosGroup: lastPDPGroup}
			Radian(k)
			ks = append(ks, k)
		}
	}
	var Offset float64 = 360
	var KF, KL *K
	for i, kf := range ks {
		for j, kl := range ks {
			if i == j {
				continue
			}
			if kf.FirstPosGroup != kl.FirstPosGroup {
				continue
			}
			offset := IsVertical(kf, kl)
			if offset < Offset {
				Offset = offset
				KF = kf
				KL = kl
			}
		}
	}
	positionDetectionPatterns := new(PositionDetectionPatterns)
	positionDetectionPatterns.TopLeft = KF.FirstPosGroup
	positionDetectionPatterns.Bottom = KL.LastPosGroup
	positionDetectionPatterns.Right = KF.LastPosGroup
	return positionDetectionPatterns, nil
}

func PossListToGroup(groups []*PosGroup) *PosGroup {
	var newGroup []Pos
	for _, group := range groups {
		newGroup = append(newGroup, group.Group...)
	}
	return PossToGroup(newGroup)
}

type K struct {
	FirstPosGroup *PosGroup
	LastPosGroup  *PosGroup
	K             float64
}

func Radian(k *K) {
	x, y := k.LastPosGroup.Center.X-k.FirstPosGroup.Center.X, k.LastPosGroup.Center.Y-k.FirstPosGroup.Center.Y
	k.K = math.Atan2(float64(y), float64(x))
}

func IsVertical(kf, kl *K) (offset float64) {
	dk := kl.K - kf.K
	offset = math.Abs(dk - math.Pi/2)
	return
}

func PossToGroup(group []Pos) *PosGroup {
	posGroup := new(PosGroup)
	posGroup.Group = group
	posGroup.Center = CenterPoint(group)
	posGroup.GroupMap = make(map[Pos]bool)
	for _, pos := range group {
		posGroup.GroupMap[pos] = true
	}
	minX, maxX, minY, maxY := Rectangle(group)
	posGroup.Min = Pos{X: minX, Y: minY}
	posGroup.Max = Pos{X: maxX, Y: maxY}
	posGroup.Hollow = Hollow(posGroup)
	return posGroup
}

func Rectangle(group []Pos) (minX, maxX, minY, maxY int) {
	minX, maxX, minY, maxY = group[0].X, group[0].X, group[0].Y, group[0].Y
	for _, pos := range group {
		if pos.X < minX {
			minX = pos.X
		}
		if pos.X > maxX {
			maxX = pos.X
		}
		if pos.Y < minY {
			minY = pos.Y
		}
		if pos.Y > maxY {
			maxY = pos.Y
		}
	}
	return
}

func CenterPoint(group []Pos) Pos {
	sumX, sumY := 0, 0
	for _, pos := range group {
		sumX += pos.X
		sumY += pos.Y
	}
	meanX := sumX / len(group)
	meanY := sumY / len(group)
	return Pos{X: meanX, Y: meanY}
}

func MaskFunc(code int) func(x, y int) bool {
	switch code {
	case 0: // 000
		return func(x, y int) bool {
			return (x+y)%2 == 0
		}
	case 1: // 001
		return func(x, y int) bool {
			return y%2 == 0
		}
	case 2: // 010
		return func(x, y int) bool {
			return x%3 == 0
		}
	case 3: // 011
		return func(x, y int) bool {
			return (x+y)%3 == 0
		}
	case 4: // 100
		return func(x, y int) bool {
			return (y/2+x/3)%2 == 0
		}
	case 5: // 101
		return func(x, y int) bool {
			return (x*y)%2+(x*y)%3 == 0
		}
	case 6: // 110
		return func(x, y int) bool {
			return ((x*y)%2+(x*y)%3)%2 == 0
		}
	case 7: // 111
		return func(x, y int) bool {
			return ((x+y)%2+(x*y)%3)%2 == 0
		}
	}
	return func(x, y int) bool {
		return false
	}
}

func SplitGroup(poss *[][]bool, centerX, centerY int, around *[]Pos) {
	maxy := len(*poss) - 1
	for y := -1; y < 2; y++ {
		for x := -1; x < 2; x++ {
			hereY := centerY + y
			if hereY < 0 || hereY > maxy {
				continue
			}
			hereX := centerX + x
			maxX := len((*poss)[hereY]) - 1
			if hereX < 0 || hereX > maxX {
				continue
			}
			v := (*poss)[hereY][hereX]
			if v {
				(*poss)[hereY][hereX] = false
				*around = append(*around, Pos{hereX, hereY})
			}
		}
	}
}

func Hollow(group *PosGroup) bool {
	count := len(group.GroupMap)
	for y := group.Min.Y; y <= group.Max.Y; y++ {
		min := -1
		max := -1
		for x := group.Min.X; x <= group.Max.X; x++ {
			if group.GroupMap[Pos{x, y}] {
				if min < 0 {
					min = x
				}
				max = x
			}
		}
		count = count - (max - min + 1)
	}
	return count != 0
}

func ParseBlock(m *Matrix, data []bool) ([]bool, error) {
	version := m.Version()
	info, err := m.FormatInfo()
	if err != nil {
		return nil, err
	}
	var qrCodeVersion = QRcodeVersion{}
	for _, qrCV := range Versions {
		if qrCV.Level == RecoveryLevel(info.ErrorCorrectionLevel) && qrCV.Version == version {
			qrCodeVersion = qrCV
		}
	}

	var dataBlocks [][]bool
	for _, block := range qrCodeVersion.Block {
		for i := 0; i < block.NumBlocks; i++ {
			dataBlocks = append(dataBlocks, []bool{})
		}
	}
	for {
		leftLength := len(data)
		no := 0
		for _, block := range qrCodeVersion.Block {
			for i := 0; i < block.NumBlocks; i++ {
				if len(dataBlocks[no]) < block.NumDataCodewords*8 {
					dataBlocks[no] = append(dataBlocks[no], data[0:8]...)
					data = data[8:]
				}
				no += 1
			}
		}
		if leftLength == len(data) {
			break
		}
	}

	var errorBlocks [][]bool
	for _, block := range qrCodeVersion.Block {
		for i := 0; i < block.NumBlocks; i++ {
			errorBlocks = append(errorBlocks, []bool{})
		}
	}
	for {
		leftLength := len(data)
		no := 0
		for _, block := range qrCodeVersion.Block {
			for i := 0; i < block.NumBlocks; i++ {
				if len(errorBlocks[no]) < (block.NumCodewords-block.NumDataCodewords)*8 {
					errorBlocks[no] = append(errorBlocks[no], data[:8]...)
					if len(data) > 8 {
						data = data[8:]
					}
				}
				no += 1
			}
		}
		if leftLength == len(data) {
			break
		}
	}

	var result []byte
	for i := range dataBlocks {
		blockByte, err := QRReconstruct(Bool2Byte(dataBlocks[i]), Bool2Byte(errorBlocks[i]))
		if err != nil {
			return nil, err
		}
		result = append(result, blockByte[:len(Bool2Byte(dataBlocks[i]))]...)
	}
	return Byte2Bool(result), nil
}

func Byte2Bool(bl []byte) []bool {
	var result []bool
	for _, b := range bl {
		temp := make([]bool, 8)
		for i := 0; i < 8; i++ {
			if (b>>uint(i))&1 == 1 {
				temp[7-i] = true
			} else {
				temp[7-i] = false
			}

		}
		result = append(result, temp...)
	}
	return result
}

func IsPositionDetectionPattern(solidGroup, hollowGroup *PosGroup) bool {
	solidMinX, solidMaxX, solidMinY, solidMaxY := solidGroup.Min.X, solidGroup.Max.X, solidGroup.Min.Y, solidGroup.Max.Y
	minX, maxX, minY, maxY := hollowGroup.Min.X, hollowGroup.Max.X, hollowGroup.Min.Y, hollowGroup.Max.Y
	if !(solidMinX > minX && solidMaxX > minX &&
		solidMinX < maxX && solidMaxX < maxX &&
		solidMinY > minY && solidMaxY > minY &&
		solidMinY < maxY && solidMaxY < maxY) {
		return false
	}
	hollowCenter := hollowGroup.Center
	if !(hollowCenter.X > solidMinX && hollowCenter.X < solidMaxX &&
		hollowCenter.Y > solidMinY && hollowCenter.Y < solidMaxY) {
		return false
	}
	return true
}

func GetData(unmaskMatrix, dataArea *Matrix) []bool {
	width := len(unmaskMatrix.Points)
	var data []bool
	maxPos := width - 1
	for t := maxPos; t > 0; {
		for y := maxPos; y >= 0; y-- {
			for x := t; x >= t-1; x-- {
				if dataArea.AtPoints(x, y) {
					data = append(data, unmaskMatrix.AtPoints(x, y))
				}
			}
		}
		t = t - 2
		if t == 6 {
			t = t - 1
		}
		for y := 0; y <= maxPos; y++ {
			for x := t; x >= t-1 && x >= 0; x-- {
				if x < len(unmaskMatrix.Points[y]) && dataArea.AtPoints(x, y) {
					data = append(data, unmaskMatrix.AtPoints(x, y))
				}
			}
		}
		t = t - 2
	}
	return data
}

func Bits2Bytes(dataCode []bool, version int) ([]byte, error) {
	//前4bit为编码格式， 后四位为真正的数据
	mode := Bit2Int(dataCode[0:4])
	encoder, err := GetDataEncoder(version)
	if err != nil {
		return nil, err
	}
	err = encoder.SetCharModeCharDecoder(mode)
	if err != nil {
		return nil, err
	}

	modeCharDecoder := encoder.ModeCharDecoder

	return modeCharDecoder.Decode(dataCode[4:])
}




func StringBool(dataCode []bool) string {
	return StringByte(Bool2Byte(dataCode))
}

func StringByte(b []byte) string {
	var bitString string
	for i := 0; i < len(b)*8; i++ {
		if (i % 8) == 0 {
			bitString += " "
		}

		if (b[i/8] & (0x80 >> byte(i%8))) != 0 {
			bitString += "1"
		} else {
			bitString += "0"
		}
	}

	return fmt.Sprintf("numBits=%d, bits=%s", len(b)*8, bitString)
}

func Bool2Byte(dataCode []bool) []byte {
	var result []byte
	for i := 0; i < len(dataCode); {
		result = append(result, Bit2Byte(dataCode[i:i+8]))
		i += 8
	}
	return result
}
func Bit2Int(bits []bool) int {
	g := 0
	for _, i := range bits {
		g = g << 1
		if i {
			g += 1
		}
	}
	return g
}

func Bit2Byte(bits []bool) byte {
	var g uint8
	for _, i := range bits {
		g = g << 1
		if i {
			g += 1
		}
	}
	return byte(g)
}

func ExportGroups(size image.Rectangle, hollow []*PosGroup, filename string) error {
	result := image.NewGray(size)
	for _, group := range hollow {
		for _, pos := range group.Group {
			result.Set(pos.X, pos.Y, color.White)
		}
	}
	outImg, err := os.Create(filename + ".png")
	if err != nil {
		return err
	}
	defer outImg.Close()
	return png.Encode(outImg, result)
}

// Binarization 使用大津法（Otsu）自动计算全局阈值，避免固定阈值对光照变化敏感。
func (mx *Matrix) Binarization(pic *image.Gray) uint8 {
	var histogram [256]int
	for _, v := range pic.Pix {
		histogram[v]++
	}
	return otsuThreshold(histogram[:], len(pic.Pix))
}

// otsuThreshold 通过最大化类间方差选取阈值，使黑白两类像素的分离度最高。
func otsuThreshold(histogram []int, total int) uint8 {
	sum := 0
	for i, count := range histogram {
		sum += i * count
	}

	var (
		sumB      int
		wB        int
		maximum   float64
		threshold int
	)
	for i, count := range histogram {
		wB += count
		if wB == 0 {
			continue
		}
		wF := total - wB
		if wF == 0 {
			break
		}
		sumB += i * count
		mB := float64(sumB) / float64(wB)
		mF := float64(sum-sumB) / float64(wF)
		between := float64(wB) * float64(wF) * (mB - mF) * (mB - mF)
		if between >= maximum {
			threshold = i
			maximum = between
		}
	}
	return uint8(threshold)
}

func (mx *Matrix) SplitGroups() [][]Pos {
	m := Copy(mx.OrgPoints).([][]bool)
	var groups [][]Pos
	for y, line := range m {
		for x, v := range line {
			if !v {
				continue
			}
			var newGroup []Pos
			newGroup = append(newGroup, Pos{x, y})
			m[y][x] = false
			for i := 0; i < len(newGroup); i++ {
				v := newGroup[i]
				SplitGroup(&m, v.X, v.Y, &newGroup)
			}
			groups = append(groups, newGroup)
		}
	}
	return groups
}

func (mx *Matrix) ReadImage(batchPath string) {
	mx.OrgSize = mx.OrgImage.Bounds()
	width := mx.OrgSize.Dx()
	height := mx.OrgSize.Dy()
	pic := image.NewGray(mx.OrgSize)
	draw.Draw(pic, mx.OrgSize, mx.OrgImage, mx.OrgImage.Bounds().Min, draw.Src)
	fz := mx.Binarization(pic)
	mx.threshold = fz
	mx.grayPix = pic.Pix
	for y := 0; y < height; y++ {
		var line []bool
		for x := 0; x < width; x++ {
			if pic.Pix[y*width+x] < fz {
				line = append(line, true)
			} else {
				line = append(line, false)
			}
		}
		mx.OrgPoints = append(mx.OrgPoints, line)
	}
}

// distance 返回两点间的欧氏距离。
func distance(a, b Pos) float64 {
	dx := float64(a.X - b.X)
	dy := float64(a.Y - b.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

// clampInt 将 v 限制在 [lo, hi] 区间内。
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// estimateModuleSize 通过定位符像素面积估算单个模块的像素尺寸。
// 一个定位符由 7x7 外圈黑环(24 个模块)与 3x3 中心黑块(9 个模块)组成，共 33 个模块，
// 面积旋转无关，因此对任意角度的二维码均适用。
func estimateModuleSize(pdp *PositionDetectionPatterns) float64 {
	area := 0.0
	for _, g := range []*PosGroup{pdp.TopLeft, pdp.Right, pdp.Bottom} {
		area += float64(len(g.Group))
	}
	area /= 3
	return math.Sqrt(area / 33)
}

// nearestValidN 将估算的模块数对齐到合法的二维码边长（17 + 4*version）。
func nearestValidN(n int) int {
	if n < 17 {
		return 17
	}
	if n > 177 {
		return 177
	}
	return 17 + 4*int(math.Round(float64(n-17)/4))
}

// sampleGray 在灰度图上做双线性插值采样，坐标越界时取最近边界像素。
func (mx *Matrix) sampleGray(x, y float64) uint8 {
	width := mx.OrgSize.Dx()
	height := mx.OrgSize.Dy()

	// 先夹紧到图像范围，保证后续坐标与插值系数均有效。
	if x < 0 {
		x = 0
	} else if x > float64(width-1) {
		x = float64(width - 1)
	}
	if y < 0 {
		y = 0
	} else if y > float64(height-1) {
		y = float64(height - 1)
	}

	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	x1 := clampInt(x0+1, 0, width-1)
	y1 := clampInt(y0+1, 0, height-1)
	fx := x - float64(x0)
	fy := y - float64(y0)

	v00 := float64(mx.grayPix[y0*width+x0])
	v01 := float64(mx.grayPix[y0*width+x1])
	v10 := float64(mx.grayPix[y1*width+x0])
	v11 := float64(mx.grayPix[y1*width+x1])

	top := v00*(1-fx) + v01*fx
	bottom := v10*(1-fx) + v11*fx
	return uint8(top*(1-fy) + bottom*fy)
}

// candidateNs 依据定位符间距估算二维码边长，返回最可能的候选边长（按接近程度排序）。
// 面积法估算的模块尺寸存在几个百分点的误差，可能落在相邻版本之间，
// 因此多返回几个候选，交由 DecodeImg 逐个校验选出正确版本。
func candidateNs(pdp *PositionDetectionPatterns) []int {
	moduleSize := estimateModuleSize(pdp)
	if moduleSize <= 0 {
		return nil
	}
	tl := pdp.TopLeft.Center
	dRight := distance(tl, pdp.Right.Center)
	dBottom := distance(tl, pdp.Bottom.Center)
	avgDist := (dRight + dBottom) / 2

	nEst := avgDist/moduleSize + 7
	base := nearestValidN(int(math.Round(nEst)))

	var result []int
	for _, delta := range []int{0, -4, 4, -8, 8} {
		n := base + delta
		if n >= 17 && n <= 177 {
			result = append(result, n)
		}
	}
	return result
}

// sampleGrid 依据三个定位符中心建立仿射坐标系，按标准模块网格采样生成二值矩阵。
// 以 TopLeft 为原点、TopLeft->Right 为 X 轴、TopLeft->Bottom 为 Y 轴，
// 从而摆脱对图像 X/Y 轴对齐的依赖，支持任意角度旋转的二维码。
// n 为二维码边长（模块数），由 candidateNs 提供候选值。
func (mx *Matrix) sampleGrid(pdp *PositionDetectionPatterns, n int) error {
	tl := pdp.TopLeft.Center
	right := pdp.Right.Center
	bottom := pdp.Bottom.Center

	// 相邻定位符中心相距 N-7 个模块，据此反推精确的模块尺寸。
	dRight := distance(tl, right)
	dBottom := distance(tl, bottom)
	avgDist := (dRight + dBottom) / 2
	moduleSize := avgDist / float64(n-7)
	if moduleSize <= 0 {
		return errors.New("invalid module size")
	}

	// 以定位符中心连线方向作为仿射基向量。
	ux := float64(right.X-tl.X) / dRight
	uy := float64(right.Y-tl.Y) / dRight
	vx := float64(bottom.X-tl.X) / dBottom
	vy := float64(bottom.Y-tl.Y) / dBottom

	mx.Size = image.Rect(0, 0, n, n)
	mx.Points = make([][]bool, n)
	for row := 0; row < n; row++ {
		line := make([]bool, n)
		// 模块中心相对 TopLeft 中心的偏移（模块单位），TopLeft 中心位于 (3.5, 3.5)。
		dy := float64(row) - 3.0
		for col := 0; col < n; col++ {
			dx := float64(col) - 3.0
			x := float64(tl.X) + dx*moduleSize*ux + dy*moduleSize*vx
			y := float64(tl.Y) + dx*moduleSize*uy + dy*moduleSize*vy
			line[col] = mx.sampleGray(x, y) < mx.threshold
		}
		mx.Points[row] = line
	}
	return nil
}

func DecodeImg(img image.Image, batchPath string) (*Matrix, error) {
	matrix := new(Matrix)
	matrix.OrgImage = img

	matrix.ReadImage(batchPath)

	groups := matrix.SplitGroups()
	// 判断空心
	var hollow []*PosGroup
	// 判断实心
	var solid []*PosGroup
	for _, group := range groups {
		if len(group) == 0 {
			continue
		}
		newGroup := PossToGroup(group)
		if newGroup.Hollow {
			hollow = append(hollow, newGroup)
		} else {
			solid = append(solid, newGroup)
		}
	}
	var positionDetectionPatterns [][]*PosGroup
	for _, solidGroup := range solid {
		for _, hollowGroup := range hollow {
			if IsPositionDetectionPattern(solidGroup, hollowGroup) {
				positionDetectionPatterns = append(positionDetectionPatterns, []*PosGroup{solidGroup, hollowGroup})
			}
		}
	}
	for i, pattern := range positionDetectionPatterns {
		ExportGroups(matrix.OrgSize, pattern, filepath.Join(batchPath, "positionDetectionPattern"+strconv.Itoa(i)))
	}
	pdp, err := NewPositionDetectionPattern(positionDetectionPatterns)
	if err != nil {
		return nil, err
	}
	// 逐个候选边长采样，并以完整解码（含 RS 纠错）校验选出正确版本。
	for _, n := range candidateNs(pdp) {
		if err := matrix.sampleGrid(pdp, n); err != nil {
			continue
		}
		if err := decodeMatrix(matrix); err == nil {
			return matrix, nil
		}
	}
	return nil, errors.New("unable to decode QR code")
}

// decodeMatrix 对已采样的矩阵执行格式信息解析、去掩码、数据提取、RS 纠错与内容解码，
// 成功后把内容写入 mx.Content。
func decodeMatrix(mx *Matrix) error {
	info, err := mx.FormatInfo()
	if err != nil {
		return err
	}
	maskFunc := MaskFunc(info.Mask)
	unmaskMatrix := new(Matrix)
	for y, line := range mx.Points {
		var l []bool
		for x, value := range line {
			l = append(l, maskFunc(x, y) != value)
		}
		unmaskMatrix.Points = append(unmaskMatrix.Points, l)
	}
	dataArea := unmaskMatrix.DataArea()
	dataCode, err := ParseBlock(mx, GetData(unmaskMatrix, dataArea))
	if err != nil {
		return err
	}
	bt, err := Bits2Bytes(dataCode, unmaskMatrix.Version())
	if err != nil {
		return err
	}
	mx.Content = string(bt)
	return nil
}

// Decode 二维码识别函数
func Decode(fi io.Reader) (*Matrix, error) {
	img, _, err := image.Decode(fi)
	if err != nil {
		return nil, err
	}
	batchID := uuid.New().String()
	batchPath := filepath.Join(os.TempDir(), "tuotoo", "qrcode", batchID)
	return DecodeImg(img, batchPath)
}

func QRReconstruct(data, ecc []byte) ([]byte, error) {
	_, err := rs.NewDecoder(rs.QRCodeField256).Decode(data, ecc)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Copy creates a deep copy of whatever is passed to it and returns the copy
// in an interface{}.  The returned value will need to be asserted to the
// correct type.
func Copy(src interface{}) interface{} {
	if src == nil {
		return nil
	}

	// Make the interface a reflect.Value
	original := reflect.ValueOf(src)

	// Make a copy of the same type as the original.
	cpy := reflect.New(original.Type()).Elem()

	// Recursively copy the original.
	copyRecursive(original, cpy)

	// Return the copy as an interface.
	return cpy.Interface()
}

// copyRecursive does the actual copying of the interface. It currently has
// limited support for what it can handle. Add as needed.
func copyRecursive(original, cpy reflect.Value) {
	// handle according to original's Kind
	switch original.Kind() {
	case reflect.Ptr:
		// Get the actual value being pointed to.
		originalValue := original.Elem()

		// if  it isn't valid, return.
		if !originalValue.IsValid() {
			return
		}
		cpy.Set(reflect.New(originalValue.Type()))
		copyRecursive(originalValue, cpy.Elem())

	case reflect.Interface:
		// If this is a nil, don't do anything
		if original.IsNil() {
			return
		}
		// Get the value for the interface, not the pointer.
		originalValue := original.Elem()

		// Get the value by calling Elem().
		copyValue := reflect.New(originalValue.Type()).Elem()
		copyRecursive(originalValue, copyValue)
		cpy.Set(copyValue)

	case reflect.Struct:
		t, ok := original.Interface().(time.Time)
		if ok {
			cpy.Set(reflect.ValueOf(t))
			return
		}
		// Go through each field of the struct and copy it.
		for i := 0; i < original.NumField(); i++ {
			// The Type's StructField for a given field is checked to see if StructField.PkgPath
			// is set to determine if the field is exported or not because CanSet() returns false
			// for settable fields.  I'm not sure why.
			if original.Type().Field(i).PkgPath != "" {
				continue
			}
			copyRecursive(original.Field(i), cpy.Field(i))
		}

	case reflect.Slice:
		if original.IsNil() {
			return
		}
		// Make a new slice and copy each element.
		cpy.Set(reflect.MakeSlice(original.Type(), original.Len(), original.Cap()))
		for i := 0; i < original.Len(); i++ {
			copyRecursive(original.Index(i), cpy.Index(i))
		}

	case reflect.Map:
		if original.IsNil() {
			return
		}
		cpy.Set(reflect.MakeMap(original.Type()))
		for _, key := range original.MapKeys() {
			originalValue := original.MapIndex(key)
			copyValue := reflect.New(originalValue.Type()).Elem()
			copyRecursive(originalValue, copyValue)
			copyKey := Copy(key.Interface())
			cpy.SetMapIndex(reflect.ValueOf(copyKey), copyValue)
		}

	default:
		cpy.Set(original)
	}
}
