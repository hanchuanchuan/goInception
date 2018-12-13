package main

import (
	"fmt"
	"reflect"
)

type User struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	addr string `json:"_"`
}

func main() {
	u := User{Id: 1001, Name: "aaa", addr: "bbb"}
	t := reflect.TypeOf(u)
	v := reflect.ValueOf(u)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).CanInterface() { //判断是否为可导出字段
			fmt.Printf("%s %s = %v -tag:%s \n",
				t.Field(i).Name,
				t.Field(i).Type,
				v.Field(i).Interface(),
				t.Field(i).Tag)
		}
	}

}
