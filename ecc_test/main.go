package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/klauspost/reedsolomon"
)

func main() {
	datashards := 3
	parityShards := 1

	var ecc reedsolomon.Encoder
	ecc, err := reedsolomon.New(datashards, parityShards)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	//initial generate seed
	rand.Seed(time.Now().UnixNano())
	str := randSeq(10)
	fmt.Println(str)

	originb := []byte(str)
	fmt.Println(originb)

	originalDataSize := len(originb)

	shards, err := MakeShards(ecc, originb)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	//set any shards to nil
	shards[0] = nil

	afterb, err := TryDecodeValue(shards, ecc, parityShards, datashards)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	//before cut
	fmt.Println(afterb)

	//cut tail 0
	diff := len(afterb) - originalDataSize
	afterb = afterb[:len(afterb)-diff]

	//after cut
	fmt.Println(afterb)
	strb := string(afterb)
	fmt.Println(strb)
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func MakeShards(enc reedsolomon.Encoder, data []byte) ([][]byte, error) {
	shards, err := enc.Split(data)
	if err != nil {
		return nil, err
	}

	if err := enc.Encode(shards); err != nil {
		return nil, err
	}

	return shards, nil
}

func TryDecodeValue(shards [][]byte, enc reedsolomon.Encoder, numPShards int, numDShards int) ([]byte, error) {
	if err := enc.Reconstruct(shards); err != nil {
		return nil, err
	}

	var value []byte
	for _, data := range shards[:numDShards] {
		value = append(value, data...)
	}

	return value, nil
}
