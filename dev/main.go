package main

import (
	"git.spiritframe.com/tuotoo/utils"
	"github.com/tuotoo/qrcode"
	"log"
	"os"
)

var logger = log.New(os.Stdout, "\r\n", log.Ldate|log.Ltime|log.Llongfile)

func main() {
	fi, err := os.Open("qrcode10.png")
	if !check(err) {
		return
	}
	defer fi.Close()
    qrmatrix,err := qrcode.Decode(fi)
    check(err)
	logger.Println(qrmatrix.Content)
}

func check(err error) bool {
	return utils.Check(err)
}
