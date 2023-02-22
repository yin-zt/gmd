package models

import "fmt"

type Gmd struct {
	Util Common
}

func (this *Gmd) Help(module string, action string) {
	resp := `
    #############   shell相关   #################
    echo hello | gmd len       ## 字符串长度
    echo hello | gmd upper     ## 字符串转大写
    echo HELLO | gmd lower     ## 字符串转小写
`
	fmt.Println(resp)
}
