package models

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/httplib"
	log "github.com/cihub/seelog"
	filedriver "github.com/goftp/file-driver"
	"github.com/goftp/server"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/takama/daemon"
	"github.com/yin-zt/gmd/pkg/config"
	"github.com/yin-zt/gmd/pkg/utils"
	random "math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Common struct {
}

type Daemon struct {
	daemon.Daemon
}

type Gmd struct {
	Util *Common
	Conf *config.Config
}

func init() {
	utils.InitHttpLib()
}

var Util = &Common{}
var gmd = NewGmd()

// NewGmd 创建一个指向Gmd的实例
// 同时配置了http的的默认设置，连接超时：60s;读写超时：60s; TLS传输；
func NewGmd() *Gmd {

	var (
		gmd *Gmd
	)

	setting := httplib.BeegoHTTPSettings{
		UserAgent:        "beegoServer",
		ConnectTimeout:   60 * time.Second,
		ReadWriteTimeout: 60 * time.Second,
		Gzip:             true,
		DumpBody:         true,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: true},
	}

	httplib.SetDefaultSetting(setting)

	gmd = &Gmd{Util: Util, Conf: config.NewConfig()}
	return gmd

}

// 封装fmt.Println函数
func print(args ...interface{}) {
	fmt.Println(args)
}

//读取存取进程pid的文件(/var/lib/gmd/gmd.pid)获取到进程pid
func (this *Gmd) getPids() []string {
	//cmd := `source /etc/profile ; ps aux|grep -w 'daemon -s daemon'|grep -v grep|awk '{print $2}'`
	//cmds := []string{
	//	"/bin/bash",
	//	"-c",
	//	cmd,
	//}
	//pid, _, _ := this.util.Exec(cmds, 10, nil)
	//log.Error(cmds)
	//return strings.Trim(pid, "\n ")

	//读取存取进程pid的文件(/var/lib/gmd/gmd.pid)获取到进程pid
	pid := this.Util.ReadFile(config.PID_FILE)
	return strings.Split(strings.TrimSpace(pid), "\n")
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

// 调用this.Util.Request(url, data)进行请求
func (this *Gmd) _Request(url string, data map[string]string) string {
	resp := this.Util.Request(url, data)
	return resp
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

// LocalIp 支持获取本地ip，并返回满足查询条件(内置)的首个ip地址
func (this *Gmd) Localip(module string, action string) {
	fmt.Println(this.Util.GetLocalIP())
}

// Ip 支持获取与外网通信的网卡ip
func (this *Gmd) Ip(module string, action string) {
	fmt.Println(this.Util.GetNetworkIP())
}

// Rand 使用方法：gmd rand
// 输出[0,1] 之间的一个float64类型的浮点数
func (this *Gmd) Rand(module string, action string) {
	r := random.New(random.NewSource(time.Now().UnixNano()))
	fmt.Println(r.Float64())
}

// Lower 使用方法： echo HELLO WORLD | gmd lower
// 将输入的字符串变为小写输出
func (this *Gmd) Lower(module string, action string) {
	_, in := this.StdinJson(module, action)
	fmt.Println(strings.ToLower(in))
}

// Color 使用方法是：gmd color -m message -c color  # default color is green
// 使用指定颜色将信息打印出来
func (this *Gmd) Color(module string, action string) {
	m := ""
	c := "green"
	argv := this.Util.GetArgsMap()
	if v, ok := argv["m"]; ok {
		m = v
	}
	if v, ok := argv["c"]; ok {
		c = v
	}
	fmt.Println(this.Util.Color(m, c))
}

// Wlog 使用方法是：gmd wlog -m "log message" -l level[info|warn|error]
// 调用gmd将日志根据传入的不同等级记录到log文件中
func (this *Gmd) Wlog(module string, action string) {
	m := ""
	l := "info"
	argv := this.Util.GetArgsMap()
	if v, ok := argv["m"]; ok {
		m = v
	}
	if m == "" {
		fmt.Println("-m(message) is require, -l(level) info,warn,error")
		return
	}
	if v, ok := argv["l"]; ok {
		l = v
	}
	if l == "warn" {
		log.Warn(m)
	} else if l == "error" {
		log.Error(m)
	} else {
		log.Info(m)
	}
	fmt.Println(m)
	log.Flush()
}

// Info 使用方法 gmd info
// 输出当前gmd client的版本和gmd server信息
func (this *Gmd) Info(module string, action string) {
	res := make(map[string]string)
	res["version"] = config.CONST_VERSION
	//res["server"] = Gmd_SERVER
	fmt.Println(this.Util.JsonEncodePretty(res))
}

// Machine_id 获取本节点的UUID
func (this *Gmd) Machine_id(module string, action string) {
	uuid := this.Util.GetProductUUID()
	fmt.Println(uuid)
}

// Uuid 使用方法是：gmd uuid
// 作用是：获取一个随机的UUID
func (this *Gmd) Uuid(module string, action string) {
	id := this.Util.GetUUID()
	fmt.Println(id)
}

// Randint 使用方法是：gmd randint -r start:end #default[r] 为：0:100
// 在一个区间生成一个随机数字，默认区间为[0:100]
func (this *Gmd) Randint(module string, action string) {
	start := 0
	end := 100
	argv := this.Util.GetArgsMap()
	if v, ok := argv["r"]; ok {
		ss := strings.Split(v, ":")
		if len(ss) == 2 {
			start, _ = strconv.Atoi(ss[0])
			end, _ = strconv.Atoi(ss[1])
		}

	}

	RandInt := func(min, max int) int {
		r := random.New(random.NewSource(time.Now().UnixNano()))
		if min >= max {
			return max
		}
		return r.Intn(max-min) + min
	}

	fmt.Println(RandInt(start, end))

}

// Shell 使用方式：gmd shell  -d path -f file -t 12 -u -a -x
// 如果本地存在fpath文件(path+file); 且没有位置参数-u；则执行fpath + 其他位置参数组成的命令
func (this *Gmd) Shell(module string, action string) {
	//var err error
	//var includeReg *regexp.Regexp
	src := ""
	argv := this.Util.GetArgsMap()
	file := ""
	dir := ""
	update := "0"
	debug := "0"
	timeout := -1
	ok := true
	// -u 则是
	if v, ok := argv["u"]; ok {
		update = v
	}

	// 如果位置参数带 -x 则是使用bash执行
	if v, ok := argv["x"]; ok {
		debug = v
	}

	// -t 指定执行时间
	if v, ok := argv["t"]; ok {
		timeout, _ = strconv.Atoi(v)
	}

	// -f 指定文件
	if file, ok = argv["f"]; !ok {
		fmt.Println("-f(filename) is required")
		return
	}

	// -d 则是指定目录；如果没有-d参数，则使用 shell目录
	if dir, ok = argv["d"]; !ok {
		dir = "shell"
	}

	// ScriptPath -> /tmp/script/shell
	path := this.Conf.ScriptPath + dir
	if !this.Util.IsExist(path) {
		log.Debug(os.MkdirAll(path, 0777))
	}
	os.Chmod(path, 0777)

	//includeRegStr := `#include\s+-f\s+(?P<filename>[a-zA-Z.0-9_-]+)?\s+-d\s+(?P<dir>[a-zA-Z.0-9_-]+)?|#include\s+-d\s+(?P<dir2>[a-zA-Z.0-9_-]+)?\s+-f\s+(?P<filename2>[a-zA-Z.0-9_-]+)?`
	// 函数传入目录和文件名两个参数，请求cli server的 http://cli server/cli/shell接口
	//DownloadShell := func(dir, file string) (string, error) {
	//	req := httplib.Post(this.Conf.EnterURL + "/" + this.Conf.DefaultModule + "/shell")
	//	req.Param("dir", dir)
	//	req.Param("file", file)
	//	return req.String()
	//}

	// includestr字符串中包含如下格式："-d path -f filename"
	// 函数最后会去调用DownloadShell
	//DownloadIncludue := func(includeStr string) string {
	//	type DF struct {
	//		Dir  string
	//		File string
	//	}
	//	df := DF{}
	//	parts := strings.Split(includeStr, " ")
	//	for i, v := range parts {
	//		if v == "-d" {
	//			df.Dir = parts[i+1]
	//		}
	//		if v == "-f" {
	//			df.File = parts[i+1]
	//		}
	//	}
	//
	//	if s, err := DownloadShell(df.Dir, df.File); err != nil {
	//		log.Error(err)
	//		return includeStr
	//	} else {
	//		return s
	//	}
	//
	//}

	// path -> /tmp/script/shell ; file -> -f filename
	fpath := path + "/" + file
	fpath = strings.Replace(fpath, "///", "/", -1)
	fpath = strings.Replace(fpath, "//", "/", -1)

	// 如果文件不存在当前目录，则请求gmd server下载文件
	if update == "1" || !this.Util.IsExist(fpath) {
		fmt.Println("call gmd server")
		//if src, err = DownloadShell(dir, file); err != nil {
		//	log.Error(err)
		//	return
		//}
		//
		//if includeReg, err = regexp.Compile(includeRegStr); err != nil {
		//	log.Error(err)
		//	return
		//}
		//
		//os.MkdirAll(filepath.Dir(fpath), 0777)
		//
		//// 从cli server侧下载的src路径与正则表达式匹配；满足匹配的值的字符串则传入函数DownloadIncludue中
		//// 函数DownloadIncludue则会调用DownloadShell函数下载
		//// 如此逻辑其实调用了两次
		//src = includeReg.ReplaceAllStringFunc(src, DownloadIncludue)
		//
		//this.Util.WriteFile(fpath, src)
	} else {
		src = this.Util.ReadFile(fpath)
	}

	// 对请求gmd server url http://gmd server/gmd/shell返回的字符串进行处理
	lines := strings.Split(src, "\n")
	is_python := false
	is_shell := false
	is_powershell := false
	if len(lines) > 0 {
		// 判断字符串是否包含 python
		is_python, _ = regexp.MatchString("python", lines[0])
		// 判断字符串是否包含 bash
		is_shell, _ = regexp.MatchString("bash", lines[0])
	}

	// 判断字符串是否包含 ps1 -> powershell
	if strings.HasSuffix(file, ".ps1") {
		is_powershell = true
	}

	os.Chmod(fpath, 0777)
	result := ""

	// 组成执行命令的命令列表
	cmds := []string{
		fpath,
	}
	if is_python {
		cmds = []string{
			"/usr/bin/env",
			"python",
			fpath,
		}
	}
	if is_shell {
		cmds = []string{
			"/bin/bash",
			fpath,
		}
		if debug == "1" {
			cmds = []string{
				"/bin/bash",
				"-x",
				fpath,
			}
		}
	}

	if is_powershell {
		cmds = []string{
			"powershell",
			fpath,
		}
	}

	argvMap := this.Util.GetArgsMap()

	// 检查执行 cli shell 命令时 位置参数是否有 -a
	// 如果有，则将 -a 后接的所有值使用 “ ”分隔后，再追加到cmds数组中
	if args, ok := argvMap["a"]; ok {
		cmds = append(cmds, strings.Split(args, " ")...)
	} else {
		var args []string
		var tflag bool
		tflag = false
		for i, v := range os.Args {
			if v == "-t" {
				tflag = true
				continue
			}
			if tflag {
				tflag = false
				continue
			}
			// 如果位置参数中不是-x和-u，则将此参数放入列表args中
			if v != "-x" && v != "-u" {
				args = append(args, os.Args[i])
			}
		}
		//fmt.Println("update:",update,"debug:",debug)
		//fmt.Println("args:",args)
		// 把args列表中第6个及之后的值加入命令列表cmds中
		os.Args = args
		cmds = append(cmds, os.Args[6:]...)
		//fmt.Println("cmds", cmds)
	}

	// 本地执行脚本cmds并输出结果
	result, _, _ = this.Util.Exec(cmds, timeout, nil)
	fmt.Println(result)
}
