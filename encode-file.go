package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type Item struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Content     string `json:"content,omitempty"`
	Secret      bool   `json:"secret,omitempty"`
}

func main() {

	if len(os.Args) != 4 {
		fmt.Println("Usage:")
		fmt.Println("  encode-file <key> <input> <desc>")
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

	input, err := ioutil.ReadFile(os.Args[2])
	if err != nil {
		fmt.Printf("Couldn't open file.\n")
		return
	}

	desc := os.Args[3]

	encoded := base64.StdEncoding.EncodeToString(input)

	item := &Item{
		Name:        os.Args[2],
		Description: desc,
		Content:     encoded,
		Secret:      false,
	}

	b, err := json.Marshal(&item)
	if err != nil {
		panic(err.Error())
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
	ciphertext := make([]byte, len(b))

	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext, b)

	h := fmt.Sprintf("%x", ciphertext)
	fmt.Printf("%s", h)

}
