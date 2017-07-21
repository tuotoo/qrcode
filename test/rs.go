package main

import (
	"fmt"
	"github.com/klauspost/reedsolomon"
)

func main() {
	ds := 1
	ps := 1
	enc, err := reedsolomon.New(ds, ps)
	data := make([][]byte, ds+ps)
	text := "hello world"
	data[0]=[]byte(text)
	data[1] = make([]byte, 11)

	fmt.Printf("%#v\n", data)
	err = enc.Encode(data)
	fmt.Printf("%#v\n", data)
	ok, err := enc.Verify(data)
	println(ok)
	data[0] = []byte{0xff,0xff}
	fmt.Printf("%#v\n", data)
	ok, err = enc.Verify(data)
	println(ok)
	err = enc.Reconstruct(data)
	println(err)
	fmt.Printf("%#v\n", data)
	ok, err = enc.Verify(data)
	println(ok)
}
