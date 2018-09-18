package main

import (
	"github.com/tuotoo/qrcode"
	"log"
	"os"
	"runtime/pprof"
)

var logger = log.New(os.Stdout, "\r\n", log.Ldate|log.Ltime|log.Lshortfile)

func main() {
	f, err := os.Create("cpu-profile.prof")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()
	fi, err := os.Open("qrcode12.png")
	if !check(err) {
		return
	}
	defer fi.Close()
	qrcode.SetDebug(false)
	qrMatrix, err := qrcode.Decode(fi)
	if !check(err){
		return
	}
	logger.Println(qrMatrix.Content)
}

func check(err error) bool {
	if err != nil {
		logger.Println(err)
	}
	return err == nil
}
