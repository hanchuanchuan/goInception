package util

import (
	"bytes"
	"crypto/des"
	"encoding/hex"
	"fmt"
)

var (
	KEY = []byte{62, 10, 13, 4, 54, 21, 33, 7}
)

//
//func main() {
//
//	password := "wfkdsjhkjdskvjhfdjkv东风股份大V非日常"
//	encode, err := DesEncrypt(password,key)
//	if err != nil {
//		fmt.Println(err.Error())
//		return
//	}
//	fmt.Println(encode)
//	decode, err := DesDecrypt(encode, key)
//	if err != nil {
//		fmt.Println(err.Error())
//		return
//	}
//	fmt.Println(decode)
//}

//密钥必须是64位，所以key必须是长度为8的byte数组
func DesEncrypt(text string) (string, error) {
	if len(KEY) != 8 {
		return "", fmt.Errorf("DES加密算法要求key必须是64位bit")
	}
	block, err := des.NewCipher(KEY) //用des创建一个加密器cipher
	if err != nil {
		return "", err
	}
	src := []byte(text)
	blockSize := block.BlockSize()    //分组的大小，blockSize=8
	src = ZeroPadding(src, blockSize) //填充成64位整倍数

	out := make([]byte, len(src)) //密文和明文的长度一致
	dst := out
	for len(src) > 0 {
		//分组加密
		block.Encrypt(dst, src[:blockSize]) //对src进行加密，加密结果放到dst里
		//移到下一组
		src = src[blockSize:]
		dst = dst[blockSize:]
	}
	return "DES(" + hex.EncodeToString(out) + ")", nil
}

//DesDecrypt DES解密
//密钥必须是64位，所以key必须是长度为8的byte数组
func DesDecrypt(text string) string {
	tLen := len(text)
	if tLen < 5 {
		return text
	}
	if text[:4] != "DES(" || text[tLen-1:tLen] != ")" {
		return text
	}
	temp := text[4 : tLen-1]
	src, err := hex.DecodeString(temp) //转成[]byte
	if err != nil {
		return text
	}
	block, err := des.NewCipher(KEY)
	if err != nil {
		return text
	}

	blockSize := block.BlockSize()
	out := make([]byte, len(src))
	dst := out
	for len(src) > 0 {
		//分组解密
		block.Decrypt(dst, src[:blockSize])
		src = src[blockSize:]
		dst = dst[blockSize:]
	}
	return string(bytes.TrimRight(out, string([]byte{0})))
}

func ZeroPadding(src []byte, blockSize int) []byte {
	length := len(src)
	if length%blockSize == 0 {
		return src
	}
	blockCount := length / blockSize
	out := make([]byte, (blockCount+1)*blockSize)
	for i := 0; i < len(src); i++ {
		out[i] = src[i]
	}
	return out
}
