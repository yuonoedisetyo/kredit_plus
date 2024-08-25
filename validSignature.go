package main

import (
	"crypto/md5"
	"encoding/hex"
	//"log"
)

func validSignature(body string, signature string) string {

	body_base64 := base64Encode(body)
	generateSignature := signature + body_base64

	hasher := md5.New()
	hasher.Write([]byte(generateSignature))

	body_base64 = ""
	generateSignature = ""

	return hex.EncodeToString(hasher.Sum(nil))
}
