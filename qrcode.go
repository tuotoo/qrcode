package qrcode

import (
	"fmt"
	"github.com/maruel/rs"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"io"
	"log"
	"math"
	"os"
	reflect "reflect"
	"runtime"
	"strconv"
	"time"
)

var logger = log.New(os.Stdout, "\r\n", log.Ldate|log.Ltime|log.Llongfile)

var Debug = false

type PositionDetectionPatterns struct {
	Topleft *PosGroup
	Right   *PosGroup
	Bottom  *PosGroup
}

type PosGroup struct {
	Group    []Pos
	GroupMap map[Pos]bool
	Min      Pos
	Max      Pos
	Center   Pos
	Kong     bool
}

type Matrix struct {
	OrgImage  image.Image
	OrgSize   image.Rectangle
	OrgPoints [][]bool
	Points    [][]bool
	Size      image.Rectangle
	Data      []bool
	Content   string
}

func (m *Matrix) At(x, y int) bool {
	return m.OrgPoints[y][x]
}

func (m *Matrix) FormatInfo() (ErrorCorrectionLevel, Mask int) {
	fi1 := []Pos{
		{0, 8}, {1, 8}, {2, 8}, {3, 8}, {4, 8}, {5, 8}, {7, 8}, {8, 8},
		{8, 7}, {8, 5}, {8, 4}, {8, 3}, {8, 2}, {8, 1}, {8, 0},
	}
	maskedfidata := m.GetBin(fi1)
	unmaskfidata := maskedfidata ^ 0x5412
	if bch(unmaskfidata) == 0 {
		ErrorCorrectionLevel = unmaskfidata >> 13
		Mask = unmaskfidata >> 10 & 7
		return
	}
	length := len(m.Points)
	fi2 := []Pos{
		{8, length - 1}, {8, length - 2}, {8, length - 3}, {8, length - 4}, {8, length - 5}, {8, length - 6}, {8, length - 7},
		{length - 8, 8}, {length - 7, 8}, {length - 6, 8}, {length - 5, 8}, {length - 4, 8}, {length - 3, 8}, {length - 2, 8}, {length - 1, 8},
	}
	maskedfidata = m.GetBin(fi2)
	unmaskfidata = maskedfidata ^ 0x5412
	if bch(unmaskfidata) == 0 {
		ErrorCorrectionLevel = unmaskfidata >> 13
		Mask = unmaskfidata >> 10 & 7
		return
	}
	panic("not found errorcorrectionlevel and mask")
}

func (m *Matrix) GetBin(poss []Pos) int {
	var fidata int
	for _, pos := range poss {
		if m.Points[pos.Y][pos.X] {
			fidata = fidata<<1 + 1
		} else {
			fidata = fidata << 1
		}
	}
	return fidata
}

func (m *Matrix) Version() int {
	width := len(m.Points)
	return (width-21)/4 + 1
}

type Pos struct {
	X int
	Y int
}

func bch(org int) int {
	var g int = 0x537
	for i := 4; i > -1; i-- {
		if org&(1<<(uint(i+10))) > 0 {
			org ^= g << uint(i)
		}
	}
	return org
}

func (m *Matrix) DataArea() *Matrix {
	da := new(Matrix)
	width := len(m.Points)
	maxpos := width - 1
	for _, line := range m.Points {
		l := []bool{}
		for range line {
			l = append(l, true)
		}
		da.Points = append(da.Points, l)
	}
	// Position Detection Pattern是定位图案，用于标记二维码的矩形大小。
	// 这三个定位图案有白边叫Separators for Postion Detection Patterns。之所以三个而不是四个意思就是三个就可以标识一个矩形了。
	for y := 0; y < 9; y++ {
		for x := 0; x < 9; x++ {
			da.Points[y][x] = false //左上
		}
	}
	for y := 0; y < 9; y++ {
		for x := 0; x < 8; x++ {
			da.Points[y][maxpos-x] = false //右上
		}
	}
	for y := 0; y < 8; y++ {
		for x := 0; x < 9; x++ {
			da.Points[maxpos-y][x] = false //左下
		}
	}
	// Timing Patterns也是用于定位的。原因是二维码有40种尺寸，尺寸过大了后需要有根标准线，不然扫描的时候可能会扫歪了。
	for i := 0; i < width; i++ {
		da.Points[6][i] = false
		da.Points[i][6] = false
	}
	//Alignment Patterns 只有Version 2以上（包括Version2）的二维码需要这个东东，同样是为了定位用的。
	version := da.Version()
	Alignments := AlignmentPatternCenter[version]
	for _, AlignmentX := range Alignments {
		for _, AlignmentY := range Alignments {
			if (AlignmentX == 6 && AlignmentY == 6) || (maxpos-AlignmentX == 6 && AlignmentY == 6) || (AlignmentX == 6 && maxpos-AlignmentY == 6) {
				continue
			}
			for y := AlignmentY - 2; y <= AlignmentY+2; y++ {
				for x := AlignmentX - 2; x <= AlignmentX+2; x++ {
					da.Points[y][x] = false
				}
			}
		}
	}
	//Version Information 在 >= Version 7以上，需要预留两块3 x 6的区域存放一些版本信息。
	if version >= 7 {
		for i := maxpos - 10; i < maxpos-7; i++ {
			for j := 0; j < 6; j++ {
				da.Points[i][j] = false
				da.Points[j][i] = false
			}
		}
	}
	return da
}

func NewPositionDetectionPattern(pdps [][]*PosGroup) *PositionDetectionPatterns {
	if len(pdps) < 3 {
		panic("lost PositionDetectionPattern")
	}
	pdpgroups := []*PosGroup{}
	for _, pdp := range pdps {
		pdpgroups = append(pdpgroups, PosslistToGroup(pdp))
	}
	ks := []*K{}
	for i, firstpdpgroup := range pdpgroups {
		for j, lastpdpgroup := range pdpgroups {
			if i == j {
				continue
			}
			k := &K{FirstPosGroup: firstpdpgroup, LastPosGroup: lastpdpgroup}
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
	positionDetectionPatterns.Topleft = KF.FirstPosGroup
	positionDetectionPatterns.Bottom = KL.LastPosGroup
	positionDetectionPatterns.Right = KF.LastPosGroup
	return positionDetectionPatterns
}

func PosslistToGroup(groups []*PosGroup) *PosGroup {
	newgroup := []Pos{}
	for _, group := range groups {
		newgroup = append(newgroup, group.Group...)
	}
	return PossToGroup(newgroup)
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
	posgroup := new(PosGroup)
	posgroup.Group = group
	posgroup.Center = CenterPoint(group)
	posgroup.GroupMap = make(map[Pos]bool)
	for _, pos := range group {
		posgroup.GroupMap[pos] = true
	}
	minx, maxx, miny, maxy := Rectangle(group)
	posgroup.Min = Pos{X: minx, Y: miny}
	posgroup.Max = Pos{X: maxx, Y: maxy}
	posgroup.Kong = Kong(posgroup)
	return posgroup
}

func check(err error, backLevel ...int) (ok bool) {
	level := 1
	if len(backLevel) != 0 {
		level = backLevel[0]
	}
	if err != nil {
		_, filename, line, _ := runtime.Caller(level)
		logger.Println(filename, line, err)
		ok = false
	} else {
		ok = true
	}
	return
}

func Rectangle(group []Pos) (minx, maxx, miny, maxy int) {
	minx, maxx, miny, maxy = group[0].X, group[0].X, group[0].Y, group[0].Y
	for _, pos := range group {
		if pos.X < minx {
			minx = pos.X
		}
		if pos.X > maxx {
			maxx = pos.X
		}
		if pos.Y < miny {
			miny = pos.Y
		}
		if pos.Y > maxy {
			maxy = pos.Y
		}
	}
	return
}

func CenterPoint(group []Pos) Pos {
	sumx, sumy := 0, 0
	for _, pos := range group {
		sumx += pos.X
		sumy += pos.Y
	}
	meanx := sumx / len(group)
	meany := sumy / len(group)
	return Pos{X: meanx, Y: meany}
}

func MaskFunc(code int) func(x, y int) bool {
	switch code {
	case 0: //000
		return func(x, y int) bool {
			return (x+y)%2 == 0
		}
	case 1: //001
		return func(x, y int) bool {
			return y%2 == 0
		}
	case 2: //010
		return func(x, y int) bool {
			return x%3 == 0
		}
	case 3: //011
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

func SplitGroup(poss *[][]bool, centerx, centery int, around *[]Pos) {
	maxy := len(*poss) - 1
	for y := -1; y < 2; y++ {
		for x := -1; x < 2; x++ {
			herey := centery + y
			if herey < 0 || herey > maxy {
				continue
			}
			herex := centerx + x
			maxx := len((*poss)[herey]) - 1
			if herex < 0 || herex > maxx {
				continue
			}
			v := (*poss)[herey][herex]
			if v {
				(*poss)[herey][herex] = false
				*around = append(*around, Pos{herex, herey})
			}
		}
	}
}

func Kong(group *PosGroup) bool {
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

func ParseBlock(m *Matrix, data []bool) []bool {
	version := m.Version()
	level, _ := m.FormatInfo()
	var qrcodeversion = QRcodeVersion{}
	for _, qrcodeVersion := range Versions {
		if qrcodeVersion.Level == RecoveryLevel(level) && qrcodeVersion.Version == version {
			qrcodeversion = qrcodeVersion
		}
	}

	dataBlocks := [][]bool{}
	for _, block := range qrcodeversion.Block {
		for i := 0; i < block.NumBlocks; i++ {
			dataBlocks = append(dataBlocks, []bool{})
		}
	}
	for {
		leftlength := len(data)
		no := 0
		for _, block := range qrcodeversion.Block {
			for i := 0; i < block.NumBlocks; i++ {
				if len(dataBlocks[no]) < block.NumDataCodewords*8 {
					dataBlocks[no] = append(dataBlocks[no], data[0:8]...)
					data = data[8:]
				}
				no += 1
			}
		}
		if leftlength == len(data) {
			break
		}
	}

	errorBlocks := [][]bool{}
	for _, block := range qrcodeversion.Block {
		for i := 0; i < block.NumBlocks; i++ {
			errorBlocks = append(errorBlocks, []bool{})
		}
	}
	for {
		leftlength := len(data)
		no := 0
		for _, block := range qrcodeversion.Block {
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
		if leftlength == len(data) {
			break
		}
	}

	result := []byte{}
	for i, _ := range dataBlocks {
		blockbyte := QRReconstruct(Bool2Byte(dataBlocks[i]), Bool2Byte(errorBlocks[i]))
		result = append(result, blockbyte[:len(Bool2Byte(dataBlocks[i]))]...)
	}
	return Byte2Bool(result)
}

func Byte2Bool(bl []byte) []bool {
	result := []bool{}
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

func LineWidth(positionDetectionPatterns [][]*PosGroup) float64 {
	sumwidth := 0
	for _, positionDetectionPattern := range positionDetectionPatterns {
		for _, group := range positionDetectionPattern {
			sumwidth += group.Max.X - group.Min.X + 1
			sumwidth += group.Max.Y - group.Min.Y + 1
		}
	}
	return float64(sumwidth) / 60
}

func IsPositionDetectionPattern(bukonggroup, konggroup *PosGroup) bool {
	buminx, bumaxx, buminy, bumaxy := bukonggroup.Min.X, bukonggroup.Max.X, bukonggroup.Min.Y, bukonggroup.Max.Y
	minx, maxx, miny, maxy := konggroup.Min.X, konggroup.Max.X, konggroup.Min.Y, konggroup.Max.Y
	if !(buminx > minx && bumaxx > minx &&
		buminx < maxx && bumaxx < maxx &&
		buminy > miny && bumaxy > miny &&
		buminy < maxy && bumaxy < maxy) {
		return false
	}
	kongcenter := konggroup.Center
	if !(kongcenter.X > buminx && kongcenter.X < bumaxx &&
		kongcenter.Y > buminy && kongcenter.Y < bumaxy) {
		return false
	}
	return true
}

func GetData(unmaskmatrix, dataarea *Matrix) []bool {
	width := len(unmaskmatrix.Points)
	data := []bool{}
	maxpos := width - 1
	for t := maxpos; t > 0; {
		for y := maxpos; y >= 0; y-- {
			for x := t; x >= t-1; x-- {
				if dataarea.Points[y][x] {
					data = append(data, unmaskmatrix.Points[y][x])
				}
			}
		}
		t = t - 2
		if t == 6 {
			t = t - 1
		}
		for y := 0; y <= maxpos; y++ {
			for x := t; x >= t-1; x-- {
				if dataarea.Points[y][x] {
					data = append(data, unmaskmatrix.Points[y][x])
				}
			}
		}
		t = t - 2
	}
	return data
}

func Bits2Bytes(datacode []bool, version int) []byte {
	format := Bit2Int(datacode[0:4])
	offset := GetDataEncoder(version).CharCountBits(format)
	length := Bit2Int(datacode[4 : 4+offset])
	datacode = datacode[4+offset : length*8+4+offset]
	result := []byte{}
	for i := 0; i < length*8; {
		result = append(result, Bit2Byte(datacode[i:i+8]))
		i += 8
	}
	return result
}

func StringBool(datacode []bool) string {
	return StringByte(Bool2Byte(datacode))
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

func Bool2Byte(datacode []bool) []byte {
	result := []byte{}
	for i := 0; i < len(datacode); {
		result = append(result, Bit2Byte(datacode[i:i+8]))
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

func Line(start, end *Pos, matrix *Matrix) (line []bool) {
	if math.Abs(float64(start.X-end.X)) > math.Abs(float64(start.Y-end.Y)) {
		length := (end.X - start.X)
		if length > 0 {
			for i := 0; i <= length; i++ {
				k := float64(end.Y-start.Y) / float64(length)
				x := start.X + i
				y := start.Y + int(k*float64(i))
				line = append(line, matrix.OrgPoints[y][x])
			}
		} else {
			for i := 0; i >= length; i-- {
				k := float64(end.Y-start.Y) / float64(length)
				x := start.X + i
				y := start.Y + int(k*float64(i))
				line = append(line, matrix.OrgPoints[y][x])
			}
		}
	} else {
		length := (end.Y - start.Y)
		if length > 0 {
			for i := 0; i <= length; i++ {
				k := float64(end.X-start.X) / float64(length)
				y := start.Y + i
				x := start.X + int(k*float64(i))
				line = append(line, matrix.OrgPoints[y][x])
			}
		} else {
			for i := 0; i >= length; i-- {
				k := float64(end.X-start.X) / float64(length)
				y := start.Y + i
				x := start.X + int(k*float64(i))
				line = append(line, matrix.OrgPoints[y][x])
			}
		}
	}
	return
}

// 标线
func (m *Matrix) Centerlist(line []bool, offset int) (li []int) {
	submap := map[int]int{} //
	value := line[0]
	sublength := 0
	for _, b := range line {
		if b == value {
			sublength += 1
		} else {
			_, ok := submap[sublength]
			if ok {
				submap[sublength] += 1
			} else {
				submap[sublength] = 1
			}
			value = b
			sublength = 1
		}
	}
	var maxcountsublength float64
	var meansublength float64
	for k, v := range submap {
		if float64(v) > maxcountsublength {
			maxcountsublength = float64(v)
			meansublength = float64(k)
		}
	}
	value = !line[0]
	for index, b := range line {
		if b != value {
			li = append(li, index+offset+int(meansublength/2))
			value = b
		}
	}
	return li
	// todo: jiaodu
}

func ExportEveryGroup(size image.Rectangle, kong [][]Pos, filename string) {
	if !Debug {
		return
	}
	for i, group := range kong {
		ExportGroup(size, group, filename+strconv.FormatInt(int64(i), 10))
	}
}

func ExportGroups(size image.Rectangle, kong []*PosGroup, filename string) {
	if !Debug {
		return
	}
	result := image.NewGray(size)
	for _, group := range kong {
		for _, pos := range group.Group {
			result.Set(pos.X, pos.Y, color.White)
		}
	}
	firesult, err := os.Create(filename + ".png")
	if !check(err) {
		panic(err)
	}
	defer firesult.Close()
	png.Encode(firesult, result)
}

func ExportGroup(size image.Rectangle, group []Pos, filename string) {
	if !Debug {
		return
	}
	result := image.NewGray(size)
	for _, pos := range group {
		result.Set(pos.X, pos.Y, color.White)
	}
	firesult, err := os.Create(filename + ".png")
	if !check(err) {
		panic(err)
	}
	defer firesult.Close()
	png.Encode(firesult, result)
}

func ExportMatrix(size image.Rectangle, points [][]bool, filename string) {
	if !Debug {
		return
	}
	result := image.NewGray(size)
	for y, line := range points {
		for x, value := range line {
			var c color.Color
			if value {
				c = color.Black
			} else {
				c = color.White
			}
			result.Set(x, y, c)
		}
	}
	firesult, err := os.Create(filename + ".png")
	if !check(err) {
		panic(err)
	}
	defer firesult.Close()
	png.Encode(firesult, result)
}

func (matrix *Matrix) Binarizat() uint8 {
	return 128
}

func (matrix *Matrix) SplitGroups() [][]Pos {
	m := Copy(matrix.OrgPoints).([][]bool)
	groups := [][]Pos{}
	for y, line := range m {
		for x, v := range line {
			if !v {
				continue
			}
			newgroup := []Pos{}
			newgroup = append(newgroup, Pos{x, y})
			m[y][x] = false
			for i := 0; i < len(newgroup); i++ {
				v := newgroup[i]
				SplitGroup(&m, v.X, v.Y, &newgroup)
			}
			groups = append(groups, newgroup)
		}
	}
	return groups
}

func (matrix *Matrix) ReadImage() {
	matrix.OrgSize = matrix.OrgImage.Bounds()
	width := matrix.OrgSize.Dx()
	height := matrix.OrgSize.Dy()
	pic := image.NewGray(matrix.OrgSize)
	draw.Draw(pic, matrix.OrgSize, matrix.OrgImage, matrix.OrgImage.Bounds().Min, draw.Src)
	var fz uint8 = matrix.Binarizat() //uint8(GetOSTUThreshold(zft))
	for y := 0; y < height; y++ {
		line := []bool{}
		for x := 0; x < width; x++ {
			if pic.Pix[y*width+x] < fz {
				line = append(line, true)
			} else {
				line = append(line, false)
			}
		}
		matrix.OrgPoints = append(matrix.OrgPoints, line)
	}
	ExportMatrix(matrix.OrgSize, matrix.OrgPoints, "matrix")
}

func DecodeImg(img image.Image) (*Matrix, error) {
	matrix := new(Matrix)
	matrix.OrgImage = img
	matrix.ReadImage()

	groups := matrix.SplitGroups()
	// 判断圈圈
	kong := []*PosGroup{}
	// 判断实心
	bukong := []*PosGroup{}
	for _, group := range groups {
		if len(group) == 0 {
			continue
		}
		newgroup := PossToGroup(group)
		if newgroup.Kong {
			kong = append(kong, newgroup)
		} else {
			bukong = append(bukong, newgroup)
		}
	}
	ExportEveryGroup(matrix.OrgSize, groups, "groups/groups")
	positionDetectionPatterns := [][]*PosGroup{}
	for _, bukonggroup := range bukong {
		for _, konggroup := range kong {
			if IsPositionDetectionPattern(bukonggroup, konggroup) {
				positionDetectionPatterns = append(positionDetectionPatterns, []*PosGroup{bukonggroup, konggroup})
			}
		}
	}
	for i, pattern := range positionDetectionPatterns {
		ExportGroups(matrix.OrgSize, pattern, "positionDetectionPattern"+strconv.FormatInt(int64(i), 10))
	}
	linewidth := LineWidth(positionDetectionPatterns)
	if Debug {
		logger.Println("linewidth", linewidth)
	}
	pdp := NewPositionDetectionPattern(positionDetectionPatterns)

	//顶部标线
	topstart := &Pos{X: pdp.Topleft.Center.X + (int(3.5*linewidth) + 1), Y: pdp.Topleft.Center.Y + int(3*linewidth)}
	topend := &Pos{X: pdp.Right.Center.X - (int(3.5*linewidth) + 1), Y: pdp.Right.Center.Y + int(3*linewidth)}
	topTimePattens := Line(topstart, topend, matrix)
	if Debug {
		logger.Println("topTimePattens", topTimePattens)
	}
	topcl := matrix.Centerlist(topTimePattens, topstart.X)
	if Debug {
		logger.Println("topcl", topcl)
	}
	//左侧标线
	leftstart := &Pos{X: pdp.Topleft.Center.X + int(3*linewidth), Y: pdp.Topleft.Center.Y + (int(3.5*linewidth) + 1)}
	leftend := &Pos{X: pdp.Bottom.Center.X + int(3*linewidth), Y: pdp.Bottom.Center.Y - (int(3.5*linewidth) + 1)}
	leftTimePattens := Line(leftstart, leftend, matrix)
	if Debug {
		logger.Println("leftTimePattens", leftTimePattens)
	}
	leftcl := matrix.Centerlist(leftTimePattens, leftstart.Y)
	if Debug {
		logger.Println("leftcl", leftcl)
	}
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
	if Debug {
		logger.Println("qrtopcl", qrtopcl)
	}
	for _, y := range qrleftcl {
		line := []bool{}
		for _, x := range qrtopcl {
			line = append(line, matrix.At(x, y))
		}
		matrix.Points = append(matrix.Points, line)
	}
	matrix.Size = image.Rect(0, 0, len(matrix.Points), len(matrix.Points))
	return matrix, nil
}

func Decode(fi io.Reader) (*Matrix, error) {
	img, _, err := image.Decode(fi)
	if !check(err) {
		return nil, err
	}
	qrmatrix, err := DecodeImg(img)
	check(err)
	if Debug {
		logger.Println("qrmatrix.Size", qrmatrix.Size)
		logger.Println("qrmatrix.Points", len(qrmatrix.Points))
	}
	ExportMatrix(qrmatrix.Size, qrmatrix.Points, "bitmatrix")
	qrErrorCorrectionLevel, qrMask := qrmatrix.FormatInfo()
	if Debug {
		logger.Println("qrErrorCorrectionLevel, qrMask", qrErrorCorrectionLevel, qrMask)
	}
	maskfunc := MaskFunc(qrMask)
	unmaskmatrix := new(Matrix)
	for y, line := range qrmatrix.Points {
		l := []bool{}
		for x, value := range line {
			l = append(l, maskfunc(x, y) != value)
		}
		unmaskmatrix.Points = append(unmaskmatrix.Points, l)
	}
	if Debug {
		logger.Println("Version:", unmaskmatrix.Version())
	}
	ExportMatrix(qrmatrix.Size, unmaskmatrix.Points, "unmaskmatrix")
	dataarea := unmaskmatrix.DataArea()
	ExportMatrix(qrmatrix.Size, dataarea.Points, "mask")
	datacode := ParseBlock(qrmatrix, GetData(unmaskmatrix, dataarea))
	bt := Bits2Bytes(datacode, unmaskmatrix.Version())
	qrmatrix.Content = string(bt)
	return qrmatrix, nil
}

func QRReconstruct(data, ecc []byte) []byte {
	d := rs.NewDecoder(rs.QRCodeField256)
	orgdata := Copy(data).([]byte)
	orgecc := Copy(ecc).([]byte)
	nbErrors, err := d.Decode(data, ecc)
	if err != nil {
		if Debug {
			logger.Println("data: %s", data)
			logger.Println("ecc: %s", ecc)
			logger.Panicf("Got error: %s", err)
		}
	}
	if nbErrors != 0 && Debug {
		logger.Println("nbErrors", nbErrors)
		logger.Println("orgdata vs lastdata")
		logger.Println(StringBool(Byte2Bool(orgdata)))
		logger.Println(StringBool(Byte2Bool(data)))
		logger.Println("orgecc vs lastdata")
		logger.Println(StringBool(Byte2Bool(orgecc)))
		logger.Println(StringBool(Byte2Bool(ecc)))
	}
	return data
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
			// for settable fields.  I'm not sure why.  -mohae
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
