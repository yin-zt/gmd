package models

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/httplib"
	log "github.com/cihub/seelog"
	filedriver "github.com/goftp/file-driver"
	"github.com/goftp/server"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"os"
	"strconv"
	"strings"
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

    #############   file相关    #################
    gmd fileserver -u username -p password -h hostIP -P port -d path

    #############   日常使用相关  #################
    gmd kv -k aa -v '{\"a1\": 123, \"b1\": 456}'

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

// Kv 执行方式：go run .\main.go kv -k aa -v '{\"a1\": 123, \"b1\": 456}'
// 数据将会使用leveldb组件存储在用户家目录的o.db下
func (this *Gmd) Kv(module string, action string) {
	var (
		home     string
		body     map[string]string
		filename string
		err      error
		db       *leveldb.DB
		k        string
		v        string
		data     []byte
		obj      interface{}
	)

	body = this.Util.GetArgsMap()
	fmt.Println(body)

	if _v, ok := body["k"]; ok {
		k = _v
	} else {
		fmt.Println("(error) -k(key) require")
		return
	}

	if _v, ok := body["v"]; ok {
		v = _v
		if v == "1" || v == "" {
			obj, v = this.StdinJson(module, action)
		}

		fmt.Println(v)
		if err = json.Unmarshal([]byte(v), &obj); err != nil {
			log.Error(err)
			return
		}
	}

	if home, err = this.Util.Home(); err != nil {
		home = "./"
	}
	opts := &opt.Options{
		CompactionTableSize: 1024 * 1024 * 20,
		WriteBuffer:         1024 * 1024 * 20,
	}
	filename = home + "/" + "o.db"

	db, err = leveldb.OpenFile(filename, opts)
	if err != nil {
		log.Error(err)
		return
	}

	if v == "" {
		data, err = db.Get([]byte(k), nil)
		fmt.Println(err)
		if err != nil {
			log.Error(err)
			return
		}
		fmt.Println(string(data))
	} else {
		err = db.Put([]byte(k), []byte(v), nil)
		if err != nil {
			log.Error(err)
			return
		}
		fmt.Println("ok")

	}

}

// StdinJson 使用方法是：echo "helloworld \n wocao" | gmd StdinJson  //但不会有输出
// 作用获取用户输出，并将输入内容连成字符串返回，并不直接调用，而是由其他函数调用
func (this *Gmd) StdinJson(module string, action string) (interface{}, string) {

	var lines []string
	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		lines = append(lines, input.Text())
	}
	in := strings.Join(lines, "")
	var obj interface{}
	if err := json.Unmarshal([]byte(in), &obj); err != nil {
		log.Error(err, in)
		obj = nil
	}
	return obj, in
}

// Request 使用方式 gmd request -u url -d data
// 支持gmd对给定url发起post或者get请求，如果有-d参数则发起post请求，否则发起get请求；
func (this *Gmd) Request(module string, action string) {
	var (
		err  error
		ok   bool
		body map[string]string
		v    string
		k    string
		u    string
		req  *httplib.BeegoHTTPRequest
		html string
	)
	data := this.Util.GetArgsMap()
	_ = data

	if v, ok = data["u"]; ok {
		u = v
	}

	if v, ok = data["url"]; ok {
		u = v
	}

	if u == "" {
		fmt.Println("(error) -u(url) require")
		return
	}

	if v, ok = data["d"]; ok {
		if err = json.Unmarshal([]byte(v), &body); err != nil {
			fmt.Println(err)
			return
		}
		req = httplib.Post(u)
		for k, v = range body {

			req.Param(k, v)

		}

		if v, ok = data["f"]; ok {

			if this.Util.IsExist(v) {
				req.PostFile("file", v)
			}

		}

		if html, err = req.String(); err != nil {
			log.Error(err)
			fmt.Println(err)
			return
		}
		fmt.Println(this.Util.GBKToUTF(html))
		return
	} else {
		req = httplib.Get(u)
		if html, err = req.String(); err != nil {
			log.Error(err)
			fmt.Println(err)
			return
		}
		fmt.Println(this.Util.GBKToUTF(html))
		return
	}

}

// Exec 支持调用gmd在linux和windows上执行命令，并返回结果
// example: go run main.go exec -c command   -> 编译后可执行：gmd exec -c command
func (this *Gmd) Exec(module string, action string) {
	var (
		command string
	)
	data := this.Util.GetArgsMap()
	if v, ok := data["c"]; ok {
		command = v
	}
	this.Util.ExecCmd([]string{command}, 10)
}
