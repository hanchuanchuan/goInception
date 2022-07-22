package util

import (
	"fmt"
	"testing"
)

func TestDesEncrypt(t *testing.T) {
	key := "12345678"
	password := "ItIsMyPassword"
	encode, err := DesEncrypt(password, key)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(encode)
	decode := DesDecrypt(encode, key)
	fmt.Println(decode)
}
