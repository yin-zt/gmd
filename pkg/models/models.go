package models

import (
	"fmt"
	log "github.com/cihub/seelog"
	filedriver "github.com/goftp/file-driver"
	"github.com/goftp/server"
	"strconv"
)

type Gmd struct {
	Util Common
}

// 封装fmt.Println函数
func print(args ...interface{}) {
	fmt.Println(args)
}

// Help 介绍gmd工具的基本使用方法
func (this *Gmd) Help(module string, action string) {
	resp := `
    #############   shell相关   #################
    echo hello | gmd len       ## 字符串长度
    echo hello | gmd upper     ## 字符串转大写
    echo HELLO | gmd lower     ## 字符串转小写
`
	print(resp)
}

// Ftpserver
/*
使用方法是：gmd ftpserver -u username -p password -h hostIP -P port -d path
默认值为 -u:root  -p:本机UUID  -h:本地IP   -P:2121 -d:用户家目录
作用是：在本地使用用户家目录或者指定目录起一个ftp服务
*/
func (this *Gmd) Ftpserver(module string, action string) {

	argv := this.Util.GetArgsMap()
	user := "root"
	pass := this.Util.GetProductUUID()
	host := this.Util.GetLocalIP()
	port := 2121
	root := "/"
	if v, err := this.Util.Home(); err == nil {
		root = v
	}
	if v, ok := argv["u"]; ok {
		user = v
	}
	if v, ok := argv["h"]; ok {
		host = v
	}
	if v, ok := argv["p"]; ok {
		pass = v
	}
	if v, ok := argv["P"]; ok {
		port, _ = strconv.Atoi(v)

	}

	if v, ok := argv["d"]; ok {
		root = v
	}

	factory := &filedriver.FileDriverFactory{
		RootPath: root,
		Perm:     server.NewSimplePerm("user", "group"),
	}

	opts := &server.ServerOpts{
		Factory:  factory,
		Port:     port,
		Hostname: host,
		Auth:     &server.SimpleAuth{Name: user, Password: pass},
	}

	ftp := server.NewServer(opts)

	err := ftp.ListenAndServe()
	if err != nil {
		log.Error("Error starting ftp server:", err)
	}

}
