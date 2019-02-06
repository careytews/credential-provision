package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
)

func main() {

	if len(os.Args) != 3 {
		fmt.Println("Usage:")
		fmt.Println("  decode <key> <input>")
		os.Exit(1)
	}

	key, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Printf("Couldn't open file.\n")
		return
	}

	aeskey := make([]byte, len(key)/2)

	_, err = hex.Decode(aeskey, key)
	if err != nil {
		fmt.Println("Hex decode: " + err.Error())
		return
	}

	input, _ := ioutil.ReadFile(os.Args[2])

	inputb := make([]byte, len(input)/2)

	_, err = hex.Decode(inputb, input)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	block, err := aes.NewCipher(aeskey)
	if err != nil {
		panic(err.Error())
	}

	// IV is the counter.  This is just 138.
	iv := make([]byte, 16)
	for i := 0; i < 16; i++ {
		iv[i] = 0
	}
	iv[15] = 138

	// Ciphertext is big as plaintext.
	ciphertext := make([]byte, len(inputb))

	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext, inputb)

	fmt.Printf("%s\n", ciphertext)

}
