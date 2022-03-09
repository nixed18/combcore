package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

func good_cryption(msg string) bool {
	if len(msg) < 8 {
		return false
	}

	if string(msg[:7]) == "error: " {
		return false
	}

	return true
}

func aes_encrypt(msgstring, keystring string) string {
	// Convert
	msg := []byte(msgstring)
	key := []byte(keystring)
	/*
	key, err := hex.DecodeString(keystring)
	if err != nil {
		log.Fatal("fn2948f", err, keystring)
	}*/

	block, err := aes.NewCipher(key)
	if err != nil {
		return "error: "+err.Error()
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "error: "+err.Error()
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "error: "+err.Error()
	}

	// Result = nonce + encrypted_data
	output := aesGCM.Seal(nonce, nonce, msg, nil)
	
	// Convert to hex-bnased string and return
	return fmt.Sprintf("%x", output)
}

func aes_decrypt(msgstring, keystring string) string {
	msg, err := hex.DecodeString(msgstring)
	if err != nil {
		return "error: "+err.Error()
	}
	key := []byte(keystring)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "error: "+err.Error()
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "error: "+err.Error()
	}

	nonce_size := aesGCM.NonceSize()

	// Split nonce and msg
	nonce := msg[:nonce_size]
	encrypted_text := msg[nonce_size:]

	output, err := aesGCM.Open(nil, nonce, encrypted_text, nil)
	if err != nil {
		return "error: "+err.Error()
	}

	// Convert
	return fmt.Sprintf("%s", output)

}