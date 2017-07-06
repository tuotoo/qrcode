package main

import (
	"image/png"
	"os"
	"git.spiritframe.com/tuotoo/utils"
	"image"
	"image/draw"
	"fmt"
	"image/color"
	"strconv"
	"math"
	"github.com/astaxie/beego"
)

type PositionDetectionPatterns struct {
	topleft *PosGroup
	right   *PosGroup
	bottom  *PosGroup
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
	Points [][]bool
}

func (m *Matrix)At(x, y int) bool {
	t := 0
	f := 0
	for i := -1; i < 2; i++ {
		for j := -1; j < 2; j++ {
			if m.Points[y + i][x + j] {
				t += 1
			} else {
				f += 1
			}
		}
	}
	if t > f {
		return true
	}
	return false
}

func (m *Matrix)FormatInfo()(ErrorCorrectionLevel,Mask int) {
	fi1 := []Pos{
		{0,8},{1,8},{2,8},{3,8},{4,8},{5,8},{7,8},{8,8},
		{8,7},{8,5},{8,4},{8,3},{8,2},{8,1},{8,0},
	}
	maskedfidata := m.GetBin(fi1)
	unmaskfidata := maskedfidata ^ 0x5412
	if bch(unmaskfidata) == 0 {
		ErrorCorrectionLevel= unmaskfidata >> 13
		Mask = unmaskfidata >> 10 & 7

		fmt.Printf("FormatInfo1: ErrorCorrectionLevel %b; Mask  %b\n",ErrorCorrectionLevel,Mask)
		return
	}
	length := len(m.Points)
	fi2 := []Pos{
		{8,length-1},{8,length-2},{8,length-3},{8,length-4},{8,length-5},{8,length-6},{8,length-7},
		{length-8,8},{length-7,8},{length-6,8},{length-5,8},{length-4,8},{length-3,8},{length-2,8},{length-1,8},
	}
	maskedfidata = m.GetBin(fi2)
	unmaskfidata = maskedfidata ^ 0x5412
	if bch(unmaskfidata) == 0 {
		ErrorCorrectionLevel= unmaskfidata >> 13
		Mask = unmaskfidata >> 10 & 7

		fmt.Printf("FormatInfo2: ErrorCorrectionLevel %b; Mask  %b\n",ErrorCorrectionLevel,Mask)
		return
	}
	panic("not found errorcorrectionlevel and mask")
}

func (m *Matrix)GetBin(poss []Pos) int {
	var fidata int
	for _, pos := range (poss) {
		if m.Points[pos.Y][pos.X] {
			fidata = fidata << 1 + 1
		} else {
			fidata = fidata << 1
		}
	}
	return fidata
}

func (m *Matrix)Version()int{
	width := len(m.Points)
	return (width -21)/4+1
}

func bch(org int)int{
	var g int = 0x537
	for i := 4; i > -1; i-- {
		if org & (1 << (uint(i+10))) > 0 {
			org ^= g << uint(i)
		}
	}
	return org
}

func main() {
	fi, err := os.Open("qrcode3.png")
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
	zft := make([]int, 256)//用于保存每个像素的数量，注意这里用了int类型，在某些图像上可能会溢出。
	var idx int
	for i := 0; i < width; i++ {
		for j := 0; j < height; j++ {
			idx = i * height + j
			zft[pic.Pix[idx]]++    //image对像有一个Pix属性，它是一个slice，里面保存的是所有像素的数据。
		}
	}
	fz := uint8(GetOSTUThreshold(zft))
	var m = map[Pos]bool{}
	matrix := new(Matrix)
	for y := 0; y < height; y++ {
		line := []bool{}
		for x := 0; x < width; x++ {
			if pic.Pix[y * width + x] < fz {
				m[Pos{X:x, Y:y}] = true
				line = append(line, true)
			} else {
				line = append(line, false)
			}
		}
		matrix.Points = append(matrix.Points, line)
	}
	exportmatrix(size, matrix, "matrix")

	groups := [][]Pos{}
	for pos, _ := range (m) {
		delete(m, pos)
		groups = append(groups, SplitGroup(m, pos))
	}
	//计算分组
	c := 0
	for _, group := range (groups) {
		c += len(group)
	}
	// 判断圈圈
	kong := [][]Pos{}
	// 判断实心
	bukong := [][]Pos{}
	for _, group := range (groups) {
		if len(group) == 0 {
			continue
		}
		var groupmap = map[Pos]bool{}
		for _, pos := range (group) {
			groupmap[pos] = true
		}
		minx, maxx, miny, maxy := Rectangle(group)
		if Kong(groupmap, minx, maxx, miny, maxy) {
			kong = append(kong, group)
		} else {
			bukong = append(bukong, group)
		}
	}
	fmt.Println("groups", len(groups))
	fmt.Println("kong", len(kong))
	fmt.Println("bukong", len(bukong))
	//exporteverygroup(size,kong,"kong")
	//exporteverygroup(size,bukong,"bukong")
	exportgroups(size, groups, "groups")
	positionDetectionPatterns := [][][]Pos{}
	for _, bukonggroup := range (bukong) {
		for _, konggroup := range (kong) {
			if IsPositionDetectionPattern(bukonggroup, konggroup) {
				positionDetectionPatterns = append(positionDetectionPatterns, [][]Pos{bukonggroup, konggroup})
			}
		}
	}
	for i, pattern := range (positionDetectionPatterns) {
		exportgroups(size, pattern, "positionDetectionPattern" + strconv.FormatInt(int64(i), 10))
	}
	linewidth := LineWidth(positionDetectionPatterns)
	fmt.Println(linewidth)
	pdp := NewPositionDetectionPattern(positionDetectionPatterns)

	fmt.Println("pdp.topleft.Center", pdp.topleft.Center)

	fmt.Println("pdp.bottom.Center", pdp.bottom.Center)

	fmt.Println("pdp.right.Center", pdp.right.Center)
	topstart := &Pos{X:pdp.topleft.Center.X + (int(3.5 * linewidth) + 1), Y:pdp.topleft.Center.Y + int(3 * linewidth)}
	topend := &Pos{X:pdp.right.Center.X - (int(3.5 * linewidth) + 1), Y:pdp.right.Center.Y + int(3 * linewidth)}
	fmt.Println(topstart, topend)
	topTimePattens := Line(topstart, topend, matrix)
	fmt.Println(topTimePattens)
	topcl := centerlist(topTimePattens, topstart.X)
	fmt.Println("topcl", topcl, len(topcl))

	leftstart := &Pos{X:pdp.topleft.Center.X + int(3 * linewidth), Y:pdp.topleft.Center.Y + (int(3.5 * linewidth) + 1)}
	leftend := &Pos{X:pdp.bottom.Center.X + int(3 * linewidth), Y:pdp.bottom.Center.Y - (int(3.5 * linewidth) + 1)}
	fmt.Println(leftstart, leftend)
	leftTimePattens := Line(leftstart, leftend, matrix)
	fmt.Println(leftTimePattens)
	leftcl := centerlist(leftTimePattens, leftstart.Y)
	fmt.Println("leftcl", leftcl, len(leftcl))

	qrtopcl := []int{}
	for i := -3; i <= 3; i++ {
		qrtopcl = append(qrtopcl, pdp.topleft.Center.X + int(float64(i) * linewidth))
	}
	qrtopcl = append(qrtopcl, topcl...)
	for i := -3; i <= 3; i++ {
		qrtopcl = append(qrtopcl, pdp.right.Center.X + int(float64(i) * linewidth))
	}

	qrleftcl := []int{}
	for i := -3; i <= 3; i++ {
		qrleftcl = append(qrleftcl, pdp.topleft.Center.Y + int(float64(i) * linewidth))
	}
	qrleftcl = append(qrleftcl, leftcl...)
	for i := -3; i <= 3; i++ {
		qrleftcl = append(qrleftcl, pdp.bottom.Center.Y + int(float64(i) * linewidth))
	}

	fmt.Println("qrtopcl", qrtopcl, len(qrtopcl))
	fmt.Println("qrleftcl", qrleftcl, len(qrleftcl))

	qrmatrix := new(Matrix)
	for _, y := range (qrleftcl) {
		line := []bool{}
		for _, x := range (qrtopcl) {
			line = append(line, matrix.At(x, y))
		}
		qrmatrix.Points = append(qrmatrix.Points, line)
	}
	exportmatrix(image.Rect(0, 0, len(qrtopcl), len(qrleftcl)), qrmatrix, "bitmatrix")
	qrErrorCorrectionLevel,qrMask := qrmatrix.FormatInfo()
	beego.Debug(qrErrorCorrectionLevel,qrMask)
	maskfunc := MaskFunc(qrMask)
	unmaskmatrix := new(Matrix)
	for y,line := range(qrmatrix.Points){
		l := []bool{}
		for x,value := range(line){
			l = append(l,maskfunc(x,y)!=value)
		}
		unmaskmatrix.Points = append(unmaskmatrix.Points,l)
	}

	fmt.Println(qrmatrix.Points[0])
	fmt.Println(unmaskmatrix.Points[0])
	exportmatrix(image.Rect(0,0,len(qrtopcl), len(qrleftcl)),unmaskmatrix,"unmaskmatrix")
	dataarea := unmaskmatrix.DataArea()
	exportmatrix(image.Rect(0,0,len(qrtopcl), len(qrleftcl)),dataarea,"dataarea")

func ParseData(data []bool,format int){

}

func GetData(unmaskmatrix,dataarea *Matrix)[]bool{
	width := len(unmaskmatrix.Points)
	data := []bool{}
	maxpos := width -1
	for t:=maxpos;t>0;{
		for y:=maxpos;y>=0;y--{
			for x:=t;x>=t-1;x--{
				if dataarea.Points[y][x]{
					data = append(data,unmaskmatrix.Points[y][x])
				}
			}
		}
		for y:=0;y>=maxpos;y--{
			for x:=t;x>=t-1;x--{
				if dataarea.Points[y][x]{
					data = append(data,unmaskmatrix.Points[y][x])
				}
			}
		}
		t = t-2
		if t == 6{
			t=t-1
		}
	}
	return data
}

func Line(start, end *Pos, matrix *Matrix) (line []bool) {
	if math.Abs(float64(start.X - end.X)) > math.Abs(float64(start.Y - end.Y)) {
		length := (end.X - start.X )
		if length > 0 {
			for i := 0; i <= length; i++ {
				k := float64(end.Y - start.Y) / float64(length)
				x := start.X + i
				y := start.Y + int(k * float64(i))
				//fmt.Println(x,y,matrix.Points[y][x])
				line = append(line, matrix.Points[y][x])
			}
		} else {
			for i := 0; i >= length; i-- {
				k := float64(end.Y - start.Y) / float64(length)
				x := start.X + i
				y := start.Y + int(k * float64(i))
				//fmt.Println(x,y,matrix.Points[y][x])
				line = append(line, matrix.Points[y][x])
			}
		}
	} else {
		length := (end.Y - start.Y)
		if length > 0 {
			for i := 0; i <= length; i++ {
				k := float64(end.X - start.X) / float64(length)
				y := start.Y + i
				x := start.X + int(k * float64(i))
				//fmt.Println(x,y,matrix.Points[y][x])
				line = append(line, matrix.Points[y][x])
			}
		} else {
			for i := 0; i >= length; i-- {
				k := float64(end.X - start.X) / float64(length)
				y := start.Y + i
				x := start.X + int(k * float64(i))
				//fmt.Println(x,y,matrix.Points[y][x])
				line = append(line, matrix.Points[y][x])
			}
		}
	}
	return
}

// 标线
func centerlist(line []bool, offset int) (li []int) {
	submap := map[int]int{}
	value := line[0]
	sublength := 0
	for _, b := range (line) {
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
	maxcountsublength := 0
	var meansublength int
	for k, v := range (submap) {
		if v > maxcountsublength {
			maxcountsublength = v
			meansublength = k
		}
	}
	fmt.Println("meansublength", meansublength)
	start := false
	curvalue := false
	curgroup := []int{}
	for i, v := range (line) {
		if v == false {
			start = true
		}
		if !start {
			continue
		}
		if v != curvalue {
			if len(curgroup) > meansublength / 2 && len(curgroup) < meansublength + meansublength / 2 {
				curvalue = v
				mean := 0
				for _, index := range (curgroup) {
					mean += index
				}
				li = append(li, mean / len(curgroup) + offset)
				curgroup = []int{}
			} else {
				curgroup = append(curgroup, i)
			}
		} else {
			curgroup = append(curgroup, i)
		}
	}
	if len(curgroup) > meansublength / 2 && len(curgroup) < meansublength + meansublength / 2 {
		mean := 0
		for _, index := range (curgroup) {
			mean += index
		}
		li = append(li, mean / len(curgroup) + offset)
	}
	fmt.Println(offset, li)
	return li
	// todo
}

func LineWidth(positionDetectionPatterns [][][]Pos) float64 {
	sumwidth := 0
	for _, positionDetectionPattern := range (positionDetectionPatterns) {
		for _, group := range (positionDetectionPattern) {
			minx, maxx, miny, maxy := Rectangle(group)
			sumwidth += maxx - minx + 1
			sumwidth += maxy - miny + 1
			fmt.Println(maxx - minx, maxy - miny)
		}
	}
	fmt.Println(sumwidth, 60, float64(sumwidth) / 60)
	return float64(sumwidth) / 60
}

func IsPositionDetectionPattern(bukonggroup, konggroup []Pos) bool {
	buminx, bumaxx, buminy, bumaxy := Rectangle(bukonggroup)
	minx, maxx, miny, maxy := Rectangle(konggroup)
	if !(buminx > minx && bumaxx > minx &&
		buminx < maxx && bumaxx < maxx &&
		buminy > miny && bumaxy > miny &&
		buminy < maxy && bumaxy < maxy) {
		return false
	}
	kongcenter := centerpoint(konggroup)
	if !(kongcenter.X > buminx && kongcenter.X < bumaxx &&
		kongcenter.Y > buminy && kongcenter.Y < bumaxy) {
		return false
	}
	return true
}

func Rectangle(group []Pos) (minx, maxx, miny, maxy int) {
	minx, maxx, miny, maxy = group[0].X, group[0].X, group[0].Y, group[0].Y

	for _, pos := range (group) {
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

func centerpoint(group []Pos) Pos {
	sumx, sumy := 0, 0
	for _, pos := range (group) {
		sumx += pos.X
		sumy += pos.Y
	}
	meanx := sumx / len(group)
	meany := sumy / len(group)
	return Pos{X:meanx, Y:meany}
}

func exporteverygroup(size image.Rectangle, kong [][]Pos, filename string) {
	for i, group := range (kong) {
		exportgroup(size, group, filename + strconv.FormatInt(int64(i), 10))
	}
}

func exportgroups(size image.Rectangle, kong [][]Pos, filename string) {
	result := image.NewGray(size)
	for _, group := range (kong) {
		for _, pos := range (group) {
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

func exportgroup(size image.Rectangle, group []Pos, filename string) {
	result := image.NewGray(size)
	for _, pos := range (group) {
		result.Set(pos.X, pos.Y, color.White)
	}
	firesult, err := os.Create(filename + ".png")
	if !check(err) {
		panic(err)
	}
	defer firesult.Close()
	png.Encode(firesult, result)
}

func exportmatrix(size image.Rectangle, matrix *Matrix, filename string) {
	result := image.NewGray(size)
	for y, line := range (matrix.Points) {
		for x, value := range (line) {
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

func Kong(groupmap map[Pos]bool, minx, maxx, miny, maxy int) bool {
	count := 0
	for x := minx; x <= maxx; x++ {
		dian := false
		last := false
		for y := miny; y <= maxy; y++ {
			if _, ok := groupmap[Pos{X:x, Y:y}]; ok {
				if !last {
					if dian {
						if count > 2 {
							return true
						}
					} else {
						dian = true
					}
				}
				last = true
			} else {
				last = false
				if dian {
					count++
				}
			}
		}
	}
	return false
}

func SplitGroup(poss map[Pos]bool, pos Pos) []Pos {
	around := []Pos{}
	for y := -1; y < 2; y++ {
		for x := -1; x < 2; x++ {
			here := Pos{X:pos.X + x, Y:pos.Y + y}
			_, ok := poss[here]
			if ok {
				delete(poss, here)
				around = append(around, here)
			}
		}
	}
	for _, v := range (around) {
		for _, p := range (SplitGroup(poss, v)) {
			around = append(around, p)
		}
	}
	return around
}

type Pos struct {
	X int
	Y int
}

func GetOSTUThreshold(HistGram []int) int {
	var Y, Amount int
	var PixelBack, PixelFore, PixelIntegralBack, PixelIntegralFore, PixelIntegral int
	var OmegaBack, OmegaFore, MicroBack, MicroFore, SigmaB, Sigma float64 // 类间方差;
	var MinValue, MaxValue int
	var Threshold int = 0
	for MinValue = 0; MinValue < 256 && HistGram[MinValue] == 0; MinValue++ {
	}
	for MaxValue = 255; MaxValue > MinValue && HistGram[MinValue] == 0; MaxValue-- {
	}
	if MaxValue == MinValue {
		return MaxValue // 图像中只有一个颜色
	}
	if MinValue + 1 == MaxValue {
		return MinValue // 图像中只有二个颜色
	}
	for Y = MinValue; Y <= MaxValue; Y++ {
		Amount += HistGram[Y] //  像素总数
	}
	PixelIntegral = 0
	for Y = MinValue; Y <= MaxValue; Y++ {
		PixelIntegral += HistGram[Y] * Y
	}
	SigmaB = -1
	for Y = MinValue; Y < MaxValue; Y++ {
		PixelBack = PixelBack + HistGram[Y]
		PixelFore = Amount - PixelBack
		OmegaBack = float64(PixelBack) / float64(Amount)
		OmegaFore = float64(PixelFore) / float64(Amount)
		PixelIntegralBack += HistGram[Y] * Y
		PixelIntegralFore = PixelIntegral - PixelIntegralBack
		MicroBack = float64(PixelIntegralBack) / float64(PixelBack)
		MicroFore = float64(PixelIntegralFore) / float64(PixelFore)
		Sigma = OmegaBack * OmegaFore * (MicroBack - MicroFore) * (MicroBack - MicroFore)
		if Sigma > SigmaB {
			SigmaB = Sigma
			Threshold = Y
		}
	}
	return Threshold
}

func PossToGroup(group []Pos) *PosGroup {
	posgroup := new(PosGroup)
	posgroup.Group = group
	posgroup.Center = centerpoint(group)
	var mapgroup = map[Pos]bool{}
	for _, pos := range (group) {
		mapgroup[pos] = true
	}
	posgroup.GroupMap = mapgroup
	minx, maxx, miny, maxy := Rectangle(group)
	posgroup.Kong = Kong(mapgroup, minx, maxx, miny, maxy)
	posgroup.Min = Pos{X:minx, Y:miny}
	posgroup.Max = Pos{X:maxx, Y:maxy}
	return posgroup
}

func PosslistToGroup(groups [][]Pos) *PosGroup {
	newgroup := []Pos{}
	for _, group := range (groups) {
		newgroup = append(newgroup, group...)
	}
	return PossToGroup(newgroup)
}

type K struct {
	FirstPosGroup *PosGroup
	LastPosGroup  *PosGroup
	K             float64
}

func NewPositionDetectionPattern(pdps [][][]Pos) *PositionDetectionPatterns {
	if len(pdps) < 3 {
		panic("缺少pdp")
	}
	pdpgroups := []*PosGroup{}
	for _, pdp := range (pdps) {
		pdpgroups = append(pdpgroups, PosslistToGroup(pdp))
	}
	ks := []*K{}
	for i, firstpdpgroup := range (pdpgroups) {
		for j, lastpdpgroup := range (pdpgroups) {
			if i == j {
				continue
			}
			k := &K{FirstPosGroup:firstpdpgroup, LastPosGroup:lastpdpgroup}
			Radian(k)
			ks = append(ks, k)
		}
	}
	fmt.Println("len(ks)", len(ks))
	var Offset float64 = 360
	var KF, KL *K
	for i, kf := range (ks) {
		for j, kl := range (ks) {
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
	fmt.Println(Offset)
	fmt.Println(KF.FirstPosGroup.Center, KF.LastPosGroup.Center, KF.K)
	fmt.Println(KL.FirstPosGroup.Center, KL.LastPosGroup.Center, KL.K)
	positionDetectionPatterns := new(PositionDetectionPatterns)
	positionDetectionPatterns.topleft = KF.FirstPosGroup
	positionDetectionPatterns.bottom = KL.LastPosGroup
	positionDetectionPatterns.right = KF.LastPosGroup
	return positionDetectionPatterns
}

func Radian(k *K) {
	x, y := k.LastPosGroup.Center.X - k.FirstPosGroup.Center.X, k.LastPosGroup.Center.Y - k.FirstPosGroup.Center.Y
	k.K = math.Atan2(float64(y), float64(x))
}

func IsVertical(kf, kl *K) (offset float64) {
	dk := kl.K - kf.K
	offset = math.Abs(dk - math.Pi / 2)
	return
}

func check(err error) bool {
	return utils.Check(err)
}

func MaskFunc(code int)func(x,y int)(bool){
	fmt.Println(code)
	switch code{
	case 0://000
		return func(x,y int)(bool){
			return (x+y)%2 == 0
		}
	case 1://001
		return func(x,y int)(bool){
			return y%2 == 0
		}
	case 2://010
		return func(x,y int)(bool){
			return x%3 == 0
		}
	case 3://011
		return func(x,y int)(bool){
			return (x+y)%3 == 0
		}
	case 4:// 100
		return func(x,y int)(bool){
			return (y/2+x/3)%2 == 0
		}
	case 5:// 101
		return func(x,y int)(bool){
			return (x*y)%2+(x*y)%3 == 0
		}
	case 6:// 110
		return func(x,y int)(bool){
			return ((x*y)%2+(x*y)%3)%2 == 0
		}
	case 7:// 111
		return func(x,y int)(bool){
			return ((x+y)%2+(x*y)%3)%2 == 0
		}
	}
	return func(x,y int)(bool){
		return false
	}
}

func (m *Matrix)DataArea()*Matrix{
	da := new(Matrix)
	width := len(m.Points)
	maxpos := width-1
	for _,line:=range(m.Points){
		l := []bool{}
		for range(line){
			l = append(l,true)
		}
		da.Points = append(da.Points,l)
	}
	//定位标记
	for y:=0;y<9;y++{
		for x:=0;x<9;x++{
			da.Points[y][x]=false       //左上
		}
	}
	for y:=0;y<9;y++{
		for x:=0;x<8;x++{
			da.Points[y][maxpos-x]=false //右上
		}
	}
	for y:=0;y<8;y++{
		for x:=0;x<9;x++{
			da.Points[maxpos-y][x]=false //左下
		}
	}
	for i :=0; i <width; i++{
		da.Points[6][i]=false
		da.Points[i][6]=false
	}
	var Alignments = []Pos{}
	version :=  da.Version()
	switch {
	case version >0 && version <7:
		Alignments = []Pos{{maxpos-6,maxpos-6}}
	case version >=7:
		middle := maxpos/2
		Alignments = []Pos{{maxpos-6,maxpos-6},
			{maxpos-6,middle},
			{middle,maxpos-6},
			{middle,middle},
			{6,middle},
			{middle,6},
		}
	}
	for _,Alignment :=range(Alignments){
		for y:=Alignment.Y-1;y<=Alignment.Y+1;y++{
			for x:=Alignment.X-1;x<=Alignment.X+1;x++{
				da.Points[y][x] = false
			}
		}
	}
	if version >= 7{
		for i:=maxpos-8;i<maxpos-11;i++{
			for j:=0;j<6;j++{
				da.Points[i][j]=false
				da.Points[j][i]=false
			}
		}
	}
	return da
}