# QR Code Decoder by Golang

This Project is Developing.

* [lktoken] 相对原项目增加了`DecodeFromImg(img Image)`接口，方便截图中使用

# Plan

1. 动态二值化:
2. 提升图片扫描的速度 OK
3. 修复标线取值 OK
4. 容错码纠正数据 OK
5. 数据编码方式
<br/>Numbert
<br/>alphanumeric OK
<br/>8-bit byte OK
<br/>Kanji
6. 识别各角度倾斜的二维码

# Example

    fi, err := os.Open("qrcode.png")
    if err != nil{
        logger.Println(err.Error())
        return
    }
    defer fi.Close()
    qrmatrix, err := qrcode.Decode(fi)
    if err != nil{
        logger.Println(err.Error())
        return
    }
    logger.Println(qrmatrix.Content)

    fi, err := os.Open("qrcode.png")
    if err != nil{
        logger.Println(err.Error())
        return
    }
    defer fi.Close()
	img, _, _ := image.Decode(file)
	qrmatrix, _ := qrcode.DecodeFromImg(img)
    if err != nil{
        logger.Println(err.Error())
        return
    }
    logger.Println(qrmatrix.Content)