package main

import (
	"fmt"

	"github.com/klauspost/reedsolomon"
)

func main() {
	ds := 1
	ps := 2
	enc, err := reedsolomon.New(ds, ps)
	data := make([][]byte, ds+ps)
	for i := 0; i < ds+ps; i++ {
		data[i] = make([]byte, 2)
	}
	for i, in := range data[:ds] {
		for j := range in {
			in[j] = byte((i + j) & 0xff)
		}
	}

	fmt.Printf("%#v\n", data)
	err = enc.Encode(data)
	fmt.Printf("%#v\n", data)
	ok, err := enc.Verify(data)
	println(ok)
	data[0] = nil
	fmt.Printf("%#v\n", data)
	err = enc.Reconstruct(data)
	println(err)
	fmt.Printf("%#v\n", data)
}
