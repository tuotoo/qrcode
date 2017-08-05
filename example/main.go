package main

import (
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
	fi, err := os.Open("qrcode.jpg")
	if !check(err) {
		return
	}
	defer fi.Close()
	qrcode.Debug = true
	qrmatrix, err := qrcode.Decode(fi)
	check(err)
	logger.Println(qrmatrix.Content)
	pprof.StopCPUProfile()
}

func check(err error) bool {
	if err != nil {
		logger.Println(err)
	}
	return err == nil
}
