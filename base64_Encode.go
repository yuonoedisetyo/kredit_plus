package main

import (
	b64 "encoding/base64"
)

func base64Encode(string_val string) string {
	sEnc := b64.StdEncoding.EncodeToString([]byte(string_val))
	return sEnc
}
