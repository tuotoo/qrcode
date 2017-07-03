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
)

func main() {
	fi,err := os.Open("qrcode1.png")
	if !check(err){
		return
	}
	defer fi.Close()
	img,err := png.Decode(fi)
	if !check(err){
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
			idx = i*height + j
			zft[pic.Pix[idx]]++    //image对像有一个Pix属性，它是一个slice，里面保存的是所有像素的数据。
		}
	}
	fmt.Println("GetOSTUThreshold")
	fz := uint8(GetOSTUThreshold(zft))
	fmt.Println(fz)

	fmt.Println("for i := 0; i < len(pic.Pix); i++ {")
	var m = map[Pos]bool{}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if pic.Pix[y*width+x] < fz {
				m[Pos{X:x,Y:y}]=true
			}
		}
	}
	groups := [][]Pos{}
	for pos,_ := range(m){
		delete(m,pos)
		groups = append(groups, SplitGroup(m,pos))
	}
	//计算分组
	c := 0
	for _,group := range(groups){
		fmt.Println(len(group))
		c += len(group)
	}
	fmt.Println(c)
	// 判断圈圈
	kong := [][]Pos{}
	// 判断实心
	bukong := [][]Pos{}
	for _,group := range(groups){
		if len(group)==0{
			continue
		}
		var groupmap = map[Pos]bool{}
		for _,pos := range(group){
			groupmap[pos]=true
		}
		minx,maxx,miny,maxy := Rectangle(group)
		if Kong(groupmap,minx,maxx,miny,maxy){
			kong = append(kong,group)
		}else{
			bukong = append(bukong,group)
		}
	}
	fmt.Println("groups",len(groups))
	fmt.Println("kong",len(kong))
	fmt.Println("bukong",len(bukong))
	fmt.Println(bukong)
	//exporteverygroup(size,kong,"kong")
	//exporteverygroup(size,bukong,"bukong")
	positionDetectionPatterns := [][][]Pos{}
	for _,bukonggroup := range(bukong){
		for _,konggroup := range(kong){
			if PositionDetectionPattern(bukonggroup,konggroup){
				positionDetectionPatterns = append(positionDetectionPatterns,[][]Pos{bukonggroup,konggroup})
			}
		}
	}
	for i,pattern := range(positionDetectionPatterns){
		exportgroups(size,pattern,"positionDetectionPattern"+strconv.FormatInt(int64(i),10))
	}
	linewidth := LineWidth(positionDetectionPatterns)
	fmt.Println(linewidth)
}

func LineWidth(positionDetectionPatterns [][][]Pos)int{
	sumwidth := 0
	for _,positionDetectionPattern := range(positionDetectionPatterns){
		for _,group := range(positionDetectionPattern){
			minx,maxx,miny,maxy := Rectangle(group)
			sumwidth += maxx - minx+1
			sumwidth += maxy - miny+1
			fmt.Println(maxx - minx,maxy - miny)
		}
	}
	fmt.Println(sumwidth,60,sumwidth/60)
	return sumwidth/60
}

func PositionDetectionPattern(bukonggroup,konggroup []Pos)bool{
	buminx,bumaxx,buminy,bumaxy := Rectangle(bukonggroup)
	minx,maxx,miny,maxy := Rectangle(konggroup)
	if !(buminx > minx && bumaxx >minx &&
	   buminx < maxx && bumaxx < maxx &&
	   buminy > miny && bumaxy >miny &&
	   buminy < maxy && bumaxy < maxy){
		return false
	}
	kongcenter := centerpoint(konggroup)
	if !(kongcenter.X > buminx && kongcenter.X < bumaxx &&
	   kongcenter.Y > buminy && kongcenter.Y < bumaxy){
		return false
	}
	return true
}

func Rectangle(group []Pos)(minx,maxx,miny,maxy int){
	minx,maxx,miny,maxy = group[0].X,group[0].X,group[0].Y,group[0].Y

	for _,pos := range(group){
		if pos.X < minx{
			minx = pos.X
		}
		if pos.X > maxx{
			maxx = pos.X
		}
		if pos.Y <miny{
			miny = pos.Y
		}
		if pos.Y > maxy{
			maxy= pos.Y
		}
	}
	return
}

func centerpoint(group []Pos)Pos{
	sumx,sumy := 0,0
	for _,pos := range(group){
		sumx +=pos.X
		sumy += pos.Y
	}
	meanx := sumx/len(group)
	meany := sumy/len(group)
	return Pos{X:meanx,Y:meany}
}

func exporteverygroup(size image.Rectangle,kong [][]Pos ,filename string){
	for i,group := range(kong){
		exportgroup(size,group,filename+strconv.FormatInt(int64(i),10))
	}
}

func exportgroups(size image.Rectangle,kong [][]Pos ,filename string){
	result := image.NewGray(size)
	for _,group := range(kong){
		for _,pos := range(group){
			result.Set(pos.X,pos.Y,color.White)
		}
	}
	firesult,err := os.Create(filename+".png")
	if !check(err){
		panic(err)
	}
	defer firesult.Close()
	png.Encode(firesult,result)
}

func exportgroup(size image.Rectangle,group []Pos,filename string){
	result := image.NewGray(size)
	for _,pos := range(group){
		result.Set(pos.X,pos.Y,color.White)
	}
	firesult,err := os.Create(filename+".png")
	if !check(err){
		panic(err)
	}
	defer firesult.Close()
	png.Encode(firesult,result)
}

func Kong(groupmap map[Pos]bool,minx,maxx,miny,maxy int)bool{
	count := 0
	for x:=minx;x<=maxx;x++{
		dian := false
		last := false
		for y:=miny;y<=maxy;y++{
			if _,ok :=groupmap[Pos{X:x,Y:y}];ok{
				if !last{
					if dian{
						if count >2{
							return true
						}
					}else{
						dian = true
					}
				}
				last = true
			}else{
				last = false
				if dian{
					count++
				}
			}
		}
	}
	return false
}

func SplitGroup(poss map[Pos]bool,pos Pos)[]Pos{
	around := []Pos{}
	for y:=-1;y<2;y++{
		for x:=-1;x<2;x++{
			here := Pos{X:pos.X+x,Y:pos.Y+y}
			_,ok := poss[here]
			if ok{
				delete(poss,here)
				around = append(around,here)
			}
		}
	}
	for _,v:=range(around){
		for _,p :=range(SplitGroup(poss,v)){
			around=append(around, p)
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
	if MinValue+1 == MaxValue {
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






func check(err error)bool{
	return utils.Check(err)
}