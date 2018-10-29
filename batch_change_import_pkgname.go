package main

import (
    // "flag"
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"
    "strings"
)

type ReplaceHelper struct {
    Root    string //根目录
    OldText string //需要替换的文本
    NewText string //新的文本
}

func (h *ReplaceHelper) DoWrok() error {

    return filepath.Walk(h.Root, h.walkCallback)

}

func (h ReplaceHelper) walkCallback(path string, f os.FileInfo, err error) error {

    if err != nil {
        return err
    }
    if f == nil {
        return nil
    }
    if f.IsDir() {
        //fmt.Pringln("DIR:",path)
        return nil
    }

    if !strings.HasSuffix(path, ".go") {
        return nil
    }

    if strings.HasPrefix(path, ".") || strings.HasPrefix(path, "vendor") {
        return nil
    }

    //文件类型需要进行过滤

    fmt.Println(path)
    buf, err := ioutil.ReadFile(path)
    if err != nil {
        //err
        return err
    }
    content := string(buf)

    //替换
    newContent := strings.Replace(content, h.OldText, h.NewText, -1)

    //重新写入
    ioutil.WriteFile(path, []byte(newContent), 0)

    return err
}

func main() {

    // flag.Parse()
    helper := ReplaceHelper{
        Root:    ".",
        OldText: "github.com/hanchuanchuan/tidb",
        NewText: "github.com/hanchuanchuan/tidb",
    }
    err := helper.DoWrok()
    if err == nil {
        fmt.Println("done!")
    } else {
        fmt.Println("error:", err.Error())
    }
}
