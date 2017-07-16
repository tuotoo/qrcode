package main

import (
	"git.spiritframe.com/tuotoo/utils"
	"github.com/tuotoo/qrcode"
	"log"
	"os"
	"runtime/pprof"
)

var logger = log.New(os.Stdout, "\r\n", log.Ldate|log.Ltime|log.Llongfile)

func main() {
	f, err := os.Create("cpu-profile.prof")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	fi, err := os.Open("erweima.jpg")
	if !check(err) {
		return
	}
	defer fi.Close()
	qrmatrix, err := qrcode.Decode(fi)
	check(err)
	logger.Println(qrmatrix.Content)
	pprof.StopCPUProfile()
}

func check(err error) bool {
	return utils.Check(err)
}
