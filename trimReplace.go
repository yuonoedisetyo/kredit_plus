package main 

import (
	"strings"
)

func trimReplace(txt_value string) string {

	text1 := strings.Trim(txt_value," ")

	text2 := strings.ReplaceAll(text1,"'","''")

	text3 := strings.ReplaceAll(text2,"\"","\"")

	return text3

}