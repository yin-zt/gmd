package models

import (
	"bufio"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
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
	"net/http"
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

// Httpserver 使用方法是：gmd httpserver -h host -p port -d path
// 默认host为10或172开头的ip，如果没有则使用127.0.0.1；port默认为8000；d默认为用户家目录
// 作用是将用户家目录或者指定目录下的文件以http服务的方式提供出去给人访问或下载
// 下载方式可以使用curl: http://path/filename -o outputfile
func (this *Gmd) Httpserver(module string, action string) {

	argv := this.Util.GetArgsMap()

	host := this.Util.GetLocalIP()
	port := "8000"
	root := "/"
	if v, err := this.Util.Home(); err == nil {
		root = v
	}

	if v, ok := argv["p"]; ok {
		port = v

	}

	if v, ok := argv["h"]; ok {
		host = v

	}

	if v, ok := argv["d"]; ok {
		root = v
	}

	// 设置http服务的根目录
	h := http.FileServer(http.Dir(root))
	fmt.Println(fmt.Sprintf("http server listen %s:%s", host, port))
	err := http.ListenAndServe(fmt.Sprintf("%s:%s", host, port), h)

	if err != nil {
		log.Error("Error starting http server:", err)
		fmt.Println(err)
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

// Md5 使用方法是：gmd md5 -s "string" || gmd md5 -f filename
// 目的是输出字符串string和filename文件的md5值
func (this *Gmd) Md5(module string, action string) {

	s := ""
	fn := ""
	argv := this.Util.GetArgsMap()
	if v, ok := argv["s"]; ok {
		s = v
		fmt.Println(this.Util.MD5(s))
		return
	}
	if v, ok := argv["f"]; ok {
		fn = v
	}
	if fn != "" {
		fmt.Println(this.Util.MD5File(fn))
		return
	}
	_, s = this.StdinJson(module, action)
	fmt.Println(this.Util.MD5(s))

}

// Cut 使用方法：gmd cut -s "abcdefgf" -p "2:5" | echo "longstr" | cut -p "2:-1"
// 作用是按照指定的切割间隔，切割指定字符串
// 支持用户交互输入切割字符串
func (this *Gmd) Cut(module string, action string) {

	s := ""
	start := 0
	end := -1
	argv := this.Util.GetArgsMap()
	if v, ok := argv["s"]; ok {
		s = v
	}
	if s == "" {
		s = this.StdioStr(module, action)

	}
	if v, ok := argv["p"]; ok {
		ss := strings.Split(v, ":")
		if len(ss) == 2 {
			start, _ = strconv.Atoi(ss[0])
			if ss[1] != "" {
				end, _ = strconv.Atoi(ss[1])
			} else {
				end = len(s)
			}
		}
		if len(ss) == 1 {
			start, _ = strconv.Atoi(ss[0])
			end = len(s)
		}

	} else {
		end = len(s)
	}

	fmt.Println(s[start:end])

}

// Split 使用方法： echo "hello world" | gmd split -s " "
// 将输入字符串按照给定的字符进行分隔
// 支持用户交付输入切割字符串
func (this *Gmd) Split(module string, action string) {

	s := ""
	sep := ","

	argv := this.Util.GetArgsMap()

	if v, ok := argv["s"]; ok {
		sep = v
	}

	if s == "" {
		s = this.StdioStr(module, action)

	}

	if reg, err := regexp.Compile(sep); err == nil {
		ss := reg.Split(s, -1)
		fmt.Println(this.Util.JsonEncodePretty(ss))
	} else {
		fmt.Println(s)
	}

}

// StdioStr 的作用是获取用户输入，并组成字符串输出
func (this *Gmd) StdioStr(module string, action string) string {
	var lines []string
	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		lines = append(lines, input.Text())
	}
	return strings.Join(lines, "\n")

}

// Replace 使用方法是：gmd replace -s worldhello -n FUCK -o world
// 将输入字符串s中的子串(o,也即-o参数的值)全部替换为 -n参数的值
func (this *Gmd) Replace(module string, action string) {

	o := ""
	s := ""
	n := ""

	argv := this.Util.GetArgsMap()

	if v, ok := argv["o"]; ok {
		o = v
	}
	if v, ok := argv["s"]; ok {
		s = v
	}
	if v, ok := argv["n"]; ok {
		n = v
	}

	if s == "" {
		s = this.StdioStr(module, action)
		//_, s = this.StdinJson(module, action)

	}
	reg := regexp.MustCompile(o)
	fmt.Println(reg.ReplaceAllString(s, n))
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

// Match 使用方法是：gmd match -s "hell(i)45oworld" -m "[\d+]+" -o "i";
// -o 可搭配值为[i|a|m]; 其中i表示只匹配一个，且会在 -m 值前面加上 (?i)表示不区分大小写; a和m表示遍历字符串匹配所有
// 作用是正则匹配，注意 -m 参数的值不能以"$"结尾，不然影响切割
func (this *Gmd) Match(module string, action string) {

	m := ""
	o := ""
	s := ""

	is_all := false

	argv := this.Util.GetArgsMap()
	fmt.Println(argv)
	if v, ok := argv["m"]; ok {
		m = v
	}
	if v, ok := argv["o"]; ok {
		o = v
	}
	if v, ok := argv["s"]; ok {
		s = v
	}

	if s == "" {
		_, s = this.StdinJson(module, action)

	}
	for i := 0; i < len(o); i++ {
		if string(o[i]) == "i" {
			m = "(?i)" + m
		}
		if string(o[i]) == "a" || string(o[i]) == "m" {
			is_all = true
		}
	}

	if reg, err := regexp.Compile(m); err == nil {

		if is_all {
			ret := reg.FindAllString(s, -1)
			if len(ret) > 0 {
				fmt.Println(this.Util.JsonEncodePretty(ret))
			} else {
				fmt.Println("")
			}
		} else {
			ret := reg.FindString(s)
			fmt.Println(ret)
		}

	}
}

// Keys 使用方法是：echo '{"aa": "bb", "test": "hello world"}' | gmd keys
// 作用是将传入的类字典字符串，以字典的格式输出，但输出变量依然是字符串
func (this *Gmd) Keys(module string, action string) {

	obj, _ := this.StdinJson(module, action)
	fmt.Println(obj)
	var keys []string
	switch obj.(type) {
	case map[string]interface{}:
		for k, _ := range obj.(map[string]interface{}) {
			keys = append(keys, k)
		}
	}
	fmt.Println(this.Util.JsonEncodePretty(keys))
}

// Len 使用方法是：echo "aabbcc" | gmd len  或者  echo '{"key1": "val1", "key2": "val2"}' | gmd len
// 作用是返回输出字符串长度或者类字典数据中key的个数
func (this *Gmd) Len(module string, action string) {

	obj, in := this.StdinJson(module, action)
	switch obj.(type) {
	case []interface{}:
		fmt.Println(len(obj.([]interface{})))
		return

	case map[string]interface{}:
		i := 0
		for _ = range obj.(map[string]interface{}) {
			i = i + 1
		}
		fmt.Println(i)
		return
	}

	fmt.Println(len(in))
}

// Kvs 使用方法是：echo [k1, k2, k3] | gmd kvs  || echo '{"k1": "v1", "k2": "v2"}' | gmd kvs
// echo '{"aa": {"test":"helloworld"}, "bb": {"key1": "value"}}' | gmd kvs
// 作用是将输入的列表样式的字符串或者字典样式的字符串，转换为显示友好的字典样式
func (this *Gmd) Kvs(module string, action string) {

	obj, _ := this.StdinJson(module, action)
	var keys []string
	switch obj.(type) {
	case map[string]interface{}:
		for k, v := range obj.(map[string]interface{}) {
			switch v.(type) {
			case map[string]interface{}, []interface{}:
				if b, e := json.Marshal(v); e == nil {
					s := strings.Replace(string(b), "\\", "\\\\", -1)
					s = strings.Replace(s, "\"", "\\\"", -1)
					keys = append(keys, fmt.Sprintf(k+"=\"%s\"", s))
				}
			default:
				keys = append(keys, fmt.Sprintf(k+"=\"%s\"", v))
			}
		}
	case []interface{}:
		for i, v := range obj.([]interface{}) {

			keys = append(keys, fmt.Sprintf("a%d=\"%s\"", i, v))

		}

	}
	fmt.Println(strings.Join(keys, "\n"))
}

// Join 使用方法是 echo '["aa", "bb", "cc"]' | gmd join -s "-" -w "GG"
// 作用是将输出的字符串列表使用指定字符拼接，-w表示列表元素拼接前在其前后端加上val，-t参数目前没有作用
func (this *Gmd) Join(module string, action string) {

	obj, _ := this.StdinJson(module, action)
	sep := ","
	wrap := ""
	trim := ""
	argv := this.Util.GetArgsMap()
	if v, ok := argv["s"]; ok {
		sep = v
	}
	if v, ok := argv["t"]; ok {
		trim = v
	}
	if v, ok := argv["w"]; ok {
		wrap = v
	}
	var lines []string
	switch obj.(type) {
	case []interface{}:
		for _, v := range obj.([]interface{}) {
			if trim != "" {
				if v == nil || fmt.Sprintf("%s", v) == "" {
					continue
				}
			}
			if wrap != "" {
				lines = append(lines, fmt.Sprintf("%s%s%s", wrap, v, wrap))
			} else {
				lines = append(lines, fmt.Sprintf("%s", v))
			}
		}
	}
	fmt.Println(strings.Join(lines, sep))

}

// Sqlite 使用方法 gmd sqlite -s sql -f filename -t tablename
// 目前这个命令只适配sqlite数据库的的操作
func (this *Gmd) Sqlite(module string, action string) {
	var (
		err      error
		v        interface{}
		s        string
		isUpdate bool
		s2       string
		rows     []map[string]interface{}
		count    int64
		db       *sql.DB
	)

	f := ""
	t := "data"
	h := `
   -s sql
   -f filename
   -t tablename
`
	argv := this.Util.GetArgsMap()
	if v, ok := argv["f"]; ok {
		f = v
	}

	if _, ok := argv["h"]; ok || len(argv) == 0 {
		fmt.Println(h)
		return
	}
	if v, ok := argv["t"]; ok {
		t = v
	}
	if v, ok := argv["s"]; ok {
		s = v
	}
	if s == "" {
		v, s = this.StdinJson(module, action)
	}
	_ = s
	if s == "" {
		fmt.Println("(error) -s input null")
		return
	}

	// 此操作是判断sql执行语句是select还是update操作
	s2 = strings.TrimSpace(strings.ToLower(s))
	if strings.HasPrefix(s2, "select") {
		isUpdate = false
	}

	if v != nil {
		if db, err = this.Util.SqliteInsert(f, t, v.([]interface{})); err != nil {
			log.Error(err)
			fmt.Println(err)
			return
		}
		_ = db
		fmt.Println("ok")
		return
	}
	if !isUpdate {

		count, err = this.Util.SqliteExec(f, s)

		if err != nil {
			log.Error(err)
			fmt.Println(err)
			return
		}
		fmt.Println(fmt.Sprintf("ok(%d)", count))

	} else {
		rows, err = this.Util.SqliteQuery(f, s)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(this.Util.JsonEncodePretty(rows))
	}
}

// Pq 使用方法是 gmd pq -m html -f xxx.html
// 目的是查询html文件中所有href属性的值，并将每个html中的href组件的链接和内容以字典方式进入列表中返回
func (this *Gmd) Pq(module string, action string) {
	var (
		err   error
		dom   *goquery.Document
		title string
		href  string
		ok    bool
		text  string
		html  string
	)
	fmt.Println(ok)
	u := ""
	s := "html"
	m := "text"
	f := ""
	c := ""
	a := "link"

	argv := this.Util.GetArgsMap()
	if v, ok := argv["f"]; ok {
		f = v
	}
	if v, ok := argv["s"]; ok {
		s = v
	}
	if v, ok := argv["m"]; ok {
		m = v
	}

	if v, ok := argv["a"]; ok {
		a = v
	}

	if f != "" {

		c = this.Util.ReadFile(f)

	}

	if c == "" {
		var lines []string
		input := bufio.NewScanner(os.Stdin)
		for input.Scan() {
			lines = append(lines, input.Text())
		}
		c = strings.Join(lines, "")
	}

	_ = c
	_ = m
	_ = s
	_ = u
	_ = err

	dom, err = goquery.NewDocumentFromReader(strings.NewReader(c))

	var result []interface{}
	dom.Find(s).Each(func(i int, selection *goquery.Selection) {

		if a == "link" {
			href, ok = selection.Attr("href")
			title = selection.Text()
			item := make(map[string]string)
			item["href"] = strings.TrimSpace(href)
			item["title"] = strings.TrimSpace(title)
			result = append(result, item)
		} else if a == "table" {
			var rows []string
			selection.Find("table tr").Each(func(i int, selection *goquery.Selection) {
				var row []string
				selection.Find("td").Each(func(i int, selection *goquery.Selection) {
					row = append(row, strings.TrimSpace(selection.Text()))
				})
				rows = append(rows, strings.Join(row, "######"))
			})

			result = append(result, strings.Join(rows, "$$$$$"))

		} else {

			if m == "text" {
				text = selection.Text()
				result = append(result, text)
			}
			if m == "html" {
				fmt.Println(123)
				html, err = selection.Html()
				result = append(result, html)
			}
		}
	})

	fmt.Println(this.Util.JsonEncodePretty(result))

}

// Json_val 使用方法是 echo '{"tt":"helloworld", "bb": "fufu"}' | gmd json_val
// 直接调用jq方法
func (this *Gmd) Json_val(module string, action string) {
	this.Jq(module, action)
}

// Jq 使用方法是echo '{"tt":"helloworld", "bb": "fufu"}' | gmd jq
// json解析器，-k json中的key，嵌套时使用逗号分隔，如 -k data,rows,ip
func (this *Gmd) Jq(module string, action string) {

	data, _ := this.StdinJson(module, action)
	if data == nil {
		return
	}
	key := ""
	var obj interface{}
	argv := this.Util.GetArgsMap()
	if v, ok := argv["k"]; ok {
		key = v
	}
	var ks []string
	if strings.Contains(key, ",") {
		ks = strings.Split(key, ",")
	} else {
		ks = strings.Split(key, ".")
	}

	obj = data

	// 解析传入data的接口字符串中，是否为字典类型，如果是则查找是否字典中有key这个键，有则返回key对应的值；
	// 如果不是字典类型，则返回nil
	ParseDict := func(obj interface{}, key string) interface{} {
		switch obj.(type) {
		case map[string]interface{}:
			if v, ok := obj.(map[string]interface{})[key]; ok {
				return v
			}
		}
		return nil

	}

	// 判断传入的obj接口类型的值是否为列表样式，如果是则继续判断key是数字样式还是字符串样式，
	// 如果是数字样式则，返回列表中索引下标为key的值；如果是字符串样式，则返回遍历列表中所有元素()
	ParseList := func(obj interface{}, key string) interface{} {
		var ret []interface{}
		switch obj.(type) {
		case []interface{}:
			if ok, _ := regexp.MatchString("^\\d+$", key); ok {
				i, _ := strconv.Atoi(key)
				return obj.([]interface{})[i]
			}

			for _, v := range obj.([]interface{}) {
				switch v.(type) {
				case map[string]interface{}:
					if key == "*" {
						for _, vv := range v.(map[string]interface{}) {
							ret = append(ret, vv)
						}
					} else {
						if vv, ok := v.(map[string]interface{})[key]; ok {
							ret = append(ret, vv)
						}
					}
				case []interface{}:
					if key == "*" {
						for _, vv := range v.([]interface{}) {
							ret = append(ret, vv)
						}
					} else {
						ret = append(ret, v)
					}
				}
			}
		}
		return ret
	}
	if key != "" {
		for _, k := range ks {
			switch obj.(type) {
			case map[string]interface{}:
				obj = ParseDict(obj, k)
			case []interface{}:
				obj = ParseList(obj, k)
			}
		}
	}

	switch obj.(type) {
	case map[string]interface{}, []interface{}:
		fmt.Println(this.Util.JsonEncodePretty(obj))
	default:
		fmt.Println(obj)
	}

}

// Jf 用法：gmd jf -c * -w "condition" --limit 10
// json 数组过滤器  -w sql中的where条件 -c sql中的column
// 此命令只适配sqlite3数据库
func (this *Gmd) Jf(module string, action string) {

	data, _ := this.StdinJson(module, action)

	c := "*"
	w := "1=1"
	s := "select %s from data where 1=1 and %s %s"

	limit := ""
	argv := this.Util.GetArgsMap()
	if v, ok := argv["c"]; ok {
		c = v
	}

	if v, ok := argv["w"]; ok {
		w = v
	}

	if v, ok := argv["limit"]; ok {
		limit = " limit " + v
	}

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Error(err)
		log.Flush()
		return
	}

	Push := func(db *sql.DB, records []interface{}) error {
		hashKeys := map[string]struct{}{}

		keyword := []string{"ALTER",
			"CLOSE",
			"COMMIT",
			"CREATE",
			"DECLARE",
			"DELETE",
			"DENY",
			"DESCRIBE",
			"DOMAIN",
			"DROP",
			"EXECUTE",
			"EXPLAN",
			"FETCH",
			"GRANT",
			"INDEX",
			"INSERT",
			"OPEN",
			"PREPARE",
			"PROCEDURE",
			"REVOKE",
			"ROLLBACK",
			"SCHEMA",
			"SELECT",
			"SET",
			"SQL",
			"TABLE",
			"TRANSACTION",
			"TRIGGER",
			"UPDATE",
			"VIEW",
			"GROUP"}

		_ = keyword

		//for _, record := range records {
		//	switch record.(type) {
		//	case map[string]interface{}:
		//		for key, _ := range record.(map[string]interface{}) {
		//			key2 := key
		//			if this.util.Contains(strings.ToUpper(key), keyword) {
		//				key2 = "_" + key
		//				record.(map[string]interface{})[key2] = record.(map[string]interface{})[key]
		//				delete(record.(map[string]interface{}), key)
		//			}
		//			hashKeys[key2] = struct{}{}
		//		}
		//	}
		//}

		for _, record := range records {
			switch record.(type) {
			case map[string]interface{}:
				for key, _ := range record.(map[string]interface{}) {
					if strings.HasPrefix(key, "`") {
						continue
					}
					key2 := fmt.Sprintf("`%s`", key)
					record.(map[string]interface{})[key2] = record.(map[string]interface{})[key]
					delete(record.(map[string]interface{}), key)
					hashKeys[key2] = struct{}{}
				}
			}
		}

		keys := []string{}

		for key, _ := range hashKeys {
			keys = append(keys, key)
		}
		//		db.Exec("DROP TABLE data")
		query := "CREATE TABLE data (" + strings.Join(keys, ",") + ")"
		if _, err := db.Exec(query); err != nil {
			log.Error(query)
			log.Flush()
			return err
		}

		for _, record := range records {
			recordKeys := []string{}
			recordValues := []string{}
			recordArgs := []interface{}{}

			switch record.(type) {
			case map[string]interface{}:

				for key, value := range record.(map[string]interface{}) {
					recordKeys = append(recordKeys, key)
					recordValues = append(recordValues, "?")
					recordArgs = append(recordArgs, value)
				}

			}

			query := "INSERT INTO data (" + strings.Join(recordKeys, ",") +
				") VALUES (" + strings.Join(recordValues, ", ") + ")"

			statement, err := db.Prepare(query)
			if err != nil {
				log.Error(err, "can't prepare query: %s", query, recordKeys, recordArgs, recordValues)
				log.Flush()
				continue

			}

			_, err = statement.Exec(recordArgs...)
			if err != nil {
				log.Error(
					err, "can't insert record",
				)

			}
			statement.Close()
		}

		return nil
	}

	switch data.(type) {
	case []interface{}:
		err = Push(db, data.([]interface{}))
		if err != err {
			fmt.Println(err)
			return
		}
	default:
		msg := "(error) just support list"
		fmt.Println(msg)
		log.Error(msg)
		return

	}

	defer db.Close()
	if err == nil {
		db.SetMaxOpenConns(1)
	} else {
		log.Error(err.Error())
	}

	s = fmt.Sprintf(s, c, w, limit)

	rows, err := db.Query(s)

	if err != nil {
		log.Error(err, s)
		log.Flush()
		fmt.Println(err)
		return
	}
	defer rows.Close()

	records := []map[string]interface{}{}
	for rows.Next() {
		record := map[string]interface{}{}

		columns, err := rows.Columns()
		if err != nil {
			log.Error(
				err, "unable to obtain rows columns",
			)
			continue
		}

		pointers := []interface{}{}
		for _, column := range columns {
			var value interface{}
			pointers = append(pointers, &value)
			record[column] = &value
		}

		err = rows.Scan(pointers...)
		if err != nil {
			log.Error(err, "can't read result records")
			continue
		}

		for key, value := range record {
			indirect := *value.(*interface{})
			if value, ok := indirect.([]byte); ok {
				record[key] = string(value)
			} else {
				record[key] = indirect
			}
		}

		records = append(records, record)
	}

	fmt.Println(Util.JsonEncodePretty(records))

}

// Telnet 使用方法是 gmd telnet -h host -t time
// 示例：gmd telnet -h 8.8.8.8:53 -t 5
func (this *Gmd) Telnet(module string, action string) {
	var host string
	var response any
	tNum := 5
	argv := this.Util.GetArgsMap()
	if v, ok := argv["h"]; ok {
		host = v
	}
	if t, ok := argv["t"]; ok {
		if num, err := strconv.Atoi(t); err != nil {
			response = "-t number must be a number string"
			panic(response)
		} else {
			tNum = num
		}
	}
	this.Util.TelnetCheck(host, tNum)
}
