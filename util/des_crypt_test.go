package util

import (
	"fmt"
	"testing"
)

func TestDesEncrypt(t *testing.T) {

	password := "Ne536GYDRjirhf*$^R"
	encode, err := DesEncrypt(password)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(encode)
	decode := DesDecrypt(encode)
	fmt.Println(decode)
}
