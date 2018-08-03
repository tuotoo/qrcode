package qrcode

import (
	"testing"
	"github.com/lazywei/go-opencv/opencv"
	"runtime"
	"path"
	"git.spiritframe.com/tuotoo/utils"
)

func TestDetection(t *testing.T) {
	red := opencv.NewScalar(0, 0, 255, 0)
	green := opencv.NewScalar(0, 255, 0, 0)

	_, currentfile, _, _ := runtime.Caller(0)
	image := opencv.LoadImage(path.Join(path.Dir(currentfile), "qrcode_test.jpg"))
	defer image.Release()
	gray := opencv.CreateImage(image.Width(), image.Height(), opencv.IPL_DEPTH_8U, 1)
	defer gray.Release()
	edge := opencv.CreateImage(image.Width(), image.Height(), opencv.IPL_DEPTH_8U, 1)
	defer edge.Release()
	opencv.CvtColor(image, gray, opencv.CV_BGR2GRAY)
	// findContours 是寻找轮廓的函数，函数定义如下：
	// image：资源图片，8 bit 单通道，一般需要将普通的 BGR 图片通过 cvtColor 函数转换。
	// mode：边缘检测的模式，包括：
	//  CV_RETR_EXTERNAL：只检索最大的外部轮廓（extreme outer），没有层级关系，只取根节点的轮廓。
	//  CV_RETR_LIST：检索所有轮廓，但是没有 Parent 和 Child 的层级关系，所有轮廓都是同级的。
	//  CV_RETR_CCOMP：检索所有轮廓，并且按照二级结构组织：外轮廓和内轮廓。以前面的大图为例，0、1、2、3、4、5 都属于第0层，2a 和 3a 都属于第1层。
	//  CV_RETR_TREE：检索所有轮廓，并且按照嵌套关系组织层级。以前面的大图为例，0、1、2 属于第0层，2a 属于第1层，3 属于第2层，3a 属于第3层，4、5 属于第4层。
	// method：边缘近似的方法，包括：
	//  CV_CHAIN_APPROX_NONE：严格存储所有边缘点，即：序列中任意两个点的距离均为1。
	//  CV_CHAIN_APPROX_SIMPLE：压缩边缘，通过顶点绘制轮廓。
	opencv.Canny(gray,edge,float64(100), float64(200), 3)
	seq := edge.FindContours(opencv.CV_RETR_TREE, opencv.CV_CHAIN_APPROX_NONE, opencv.Point{0, 0})
	utils.Debug(seq.Total())
	var v,h int
	for vseq:=seq;vseq!=nil;vseq=vseq.VNext(){
		v +=1
		for hseq:=vseq;hseq!=nil;hseq=hseq.HNext(){
			h+=1
			utils.Debug(v,h,hseq.Total())
		}
	}
	// 	drawContours 是绘制边缘的函数，可以传入 findContours 函数返回的轮廓结果，在目标图像上绘制轮廓。函数定义如下：
	//
	// Python: cv2.drawContours(image, contours, contourIdx, color) → image
	//
	// 	其中：
	//
	// 	image：目标图像，直接修改目标的像素点，实现绘制。
	// 	contours：需要绘制的边缘数组。
	// 	contourIdx：需要绘制的边缘索引，如果全部绘制则为 -1。
	// 	color：绘制的颜色，为 BGR 格式的 Scalar 。
	// 	thickness：可选，绘制的密度，即描绘轮廓时所用的画笔粗细。
	// lineType: 可选，连线类型，分为以下几种：
	// 	LINE_4：4-connected line，只有相邻的点可以连接成线，一个点有四个相邻的坑位。
	// 	LINE_8：8-connected line，相邻的点或者斜对角相邻的点可以连接成线，一个点有四个相邻的坑位和四个斜对角相邻的坑位，所以一共有8个坑位。
	// 	LINE_AA：antialiased line，抗锯齿连线。
	// 	hierarchy：可选，如果需要绘制某些层级的轮廓时作为层级关系传入。
	// 	maxLevel：可选，需要绘制的层级中的最大级别。如果为1，则只绘制最外层轮廓，如果为2，绘制最外层和第二层轮廓，以此类推。
	opencv.DrawContours(image, seq, green, red, 8, 1, 8, opencv.Point{0, 0})

	opencv.SaveImage("detection.png", image, nil)
}
