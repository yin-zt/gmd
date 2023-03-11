package models

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/yin-zt/gmd/pkg/utils"
	"github.com/yin-zt/mahonia"
	"io"
	"io/ioutil"
	random "math/rand"
	"net"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"time"
)

func init() {
	log.ReplaceLogger(utils.Logger)
}

// 处理模式，如果命令行参数数量大于2(例如： gmd arg1 arg2 ...)，且arg1和 args2 均不是以"-"开头，模式为arg1；
// 其他情况下模式为gmd
func (this *Common) GetModule() string {
	if len(os.Args) > 2 {
		if !strings.HasPrefix(os.Args[1], "-") && !strings.HasPrefix(os.Args[2], "-") {
			return os.Args[1]
		} else {
			return "gmd"
		}
	} else if len(os.Args) == 2 {
		return "gmd"
	} else {
		return "gmd"
	}
}

// 获取动作 位置参数数量大于等于2且第二个位置参数不是以"-"开头，则动作返回arg2;如果第二个参数以"-"开头，且第一个位置参数不是以"-"开头则，返回arg1
// 如果位置参数数量为1，且arg1不是以"-"开头，则返回arg1; 其他情况下，返回help
func (this *Common) GetAction() string {
	if len(os.Args) >= 3 {
		if !strings.HasPrefix(os.Args[2], "-") {
			return os.Args[2]
		} else if !strings.HasPrefix(os.Args[1], "-") {
			return os.Args[1]
		} else {
			return "help"
		}
	} else if len(os.Args) == 2 && !strings.HasPrefix(os.Args[1], "-") {
		return os.Args[1]
	} else {
		return "help"
	}

}

// GetArgsMap 将位置参数转换为字典，形如：./main.go -f aa -b --back ok --file file1
// 会被转换为 map[string]string{"f": "aa", "b": 1, "back": "ok", "file": "file1"}
// 如果--file 参数，会将file1内容读取出来存入file这个键中
// 传入的位置参数中val值最后不能带"$" 不然会影响切割
func (this *Common) GetArgsMap() map[string]string {

	return this.ParseArgs(strings.Join(os.Args, "$$$$"), "$$$$")

}

// ParseArgs 将传入字符串进行解析处理，传入字符串一般是[ -s$$$$xx$$$$-f$$$$filename$$$$--test$$$$aabb]等模式
func (this *Common) ParseArgs(args string, sep string) map[string]string {

	ret := make(map[string]string)

	var argv []string

	argv = strings.Split(args, sep)
	for i, v := range argv {
		if strings.HasPrefix(v, "-") && len(v) == 2 {
			if i+1 < len(argv) && !strings.HasPrefix(argv[i+1], "-") {
				ret[v[1:]] = argv[i+1]
			}
		}

	}
	for i, v := range argv {
		if strings.HasPrefix(v, "-") && len(v) == 2 {
			if i+1 < len(argv) && strings.HasPrefix(argv[i+1], "-") {
				ret[v[1:]] = "1"
			} else if i+1 == len(argv) {
				ret[v[1:]] = "1"
			}
		}

	}

	for i, v := range argv {
		if strings.HasPrefix(v, "--") && len(v) > 3 {
			if i+1 < len(argv) && !strings.HasPrefix(argv[i+1], "--") {
				ret[v[2:]] = argv[i+1]
			}
		}

	}
	for i, v := range argv {
		if strings.HasPrefix(v, "--") && len(v) > 3 {
			if i+1 < len(argv) && strings.HasPrefix(argv[i+1], "--") {
				ret[v[2:]] = "1"
			} else if i+1 == len(argv) {
				ret[v[2:]] = "1"
			}
		}

	}
	for k, v := range ret {
		if k == "file" && this.IsExist(v) {
			ret[k] = this.ReadFile(v)
		}
	}
	return ret

}

// IsExist 方法用于判断文件是否存在
func (this *Common) IsExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

// GetLocalIP 获取本地IP，先获取本地所有ip信息，并遍历每一个ip，如果ip中有10|172开头的ip直接返回；
// 否则，就返回127.0.0.1
func (this *Common) GetLocalIP() string {

	ips := this.GetAllIps()
	for _, v := range ips {
		if strings.HasPrefix(v, "10.") || strings.HasPrefix(v, "172.") || strings.HasPrefix(v, "172.") {
			return v
		}
	}
	return "127.0.0.1"

}

// GetNetworkIP 作用是获取与"外网"通信的网卡IP
func (this *Common) GetNetworkIP() string {
	var (
		err  error
		conn net.Conn
	)
	if conn, err = net.Dial("udp", "8.8.8.8:80"); err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().String()
	idx := strings.LastIndex(localAddr, ":")
	return localAddr[0:idx]

}

// GetAllIps 获取主机的所有IP信息
func (this *Common) GetAllIps() []string {
	ips := []string{}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}
	for _, addr := range addrs {
		ip := addr.String()
		pos := strings.Index(ip, "/")
		if match, _ := regexp.MatchString("(\\d+\\.){3}\\d+", ip); match {
			if pos != -1 {
				ips = append(ips, ip[0:pos])
			}
		}
	}
	return ips
}

// Home 返回当前用户的家目录
func (this *Common) Home() (string, error) {
	user, err := user.Current()
	if nil == err {
		return user.HomeDir, nil
	}

	if "windows" == runtime.GOOS {
		return this.homeWindows()
	}

	return this.homeUnix()
}

// homeWindows获取windows用户家目录
func (this *Common) homeWindows() (string, error) {
	// 获取家目录磁盘盘符 C:   内置的环境变量
	drive := os.Getenv("HOMEDRIVE")
	// 获取用户家目录  内置的环境变量
	path := os.Getenv("HOMEPATH")
	home := drive + path
	if drive == "" || path == "" {
		// 获取用户家目录，内置环境变量
		home = os.Getenv("USERPROFILE")
	}
	if home == "" {
		return "", errors.New("HOMEDRIVE, HOMEPATH, and USERPROFILE are blank")
	}

	return home, nil
}

// homeUnix获取linux系统当前用户家目录
func (this *Common) homeUnix() (string, error) {
	// First prefer the HOME environmental variable
	if home := os.Getenv("HOME"); home != "" {
		return home, nil
	}

	// If that fails, try the shell
	var stdout bytes.Buffer
	cmd := exec.Command("sh", "-c", "eval echo ~$USER")
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}

	result := strings.TrimSpace(stdout.String())
	if result == "" {
		return "", errors.New("blank output when reading home directory")
	}

	return result, nil
}

// GetProductUUID 获取本节点的UUID
func (this *Common) GetProductUUID() string {

	if "windows" == runtime.GOOS {
		uuid := this.windowsProductUUID()
		return uuid
	}

	filename := "/usr/local/gmd/machine_id"
	uuid := this.ReadFile(filename)
	if !this.IsExist("/usr/local/gmd/") {
		os.Mkdir("/usr/local/gmd/", 0744)
	}
	if uuid == "" {
		uuid := this.GetUUID()
		this.WriteFile(filename, uuid)
	}
	return strings.Trim(uuid, "\n ")

}

// windowsProductUUID方法是在windows系统下先判断是否在用户家目录下存在.machine_id文件
// 如果存在，则判断里面是否存在本机的uuid，如果没有则获取本机uuid再写入文件中
func (this *Common) windowsProductUUID() string {
	user, err := user.Current()
	if err != nil {
		log.Debug(err)
		return ""
	}

	filename := user.HomeDir + "/.machine_id"
	var uuid string
	if !this.IsExist(filename) {
		uuid = this.GetUUID()
		this.WriteFile(filename, uuid)
		return uuid
	}

	uuid = this.ReadFile(filename)

	if uuid == "" {
		uuid = this.GetUUID()
		this.WriteFile(filename, uuid)
		return uuid
	}

	return strings.Trim(uuid, "\n")
}

// GetUUID 获取随机生成的UUID
func (this *Common) GetUUID() string {

	b := make([]byte, 48)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	id := this.MD5(base64.URLEncoding.EncodeToString(b))
	return fmt.Sprintf("%s-%s-%s-%s-%s", id[0:8], id[8:12], id[12:16], id[16:20], id[20:])

}

// MD5 输出字符串的md5值
func (this *Common) MD5(str string) string {

	md := md5.New()
	md.Write([]byte(str))
	return fmt.Sprintf("%x", md.Sum(nil))
}

// ReadFile 将文件内容读取出来并返回
func (this *Common) ReadFile(path string) string {
	if this.IsExist(path) {
		fi, err := os.Open(path)
		if err != nil {
			return ""
		}
		defer fi.Close()
		fd, err := ioutil.ReadAll(fi)
		return string(fd)
	} else {
		return ""
	}
}

// WriteFile 把输入参数的内容变量写到文件中；如果存在文件，则先删除后创建；如果不存在则直接创建
func (this *Common) WriteFile(path string, content string) bool {
	var f *os.File
	var err error
	if this.IsExist(path) {
		err = os.Remove(path)
		if err != nil {
			return false
		}
		f, err = os.Create(path)
	} else {
		f, err = os.Create(path)
	}

	if err == nil {
		defer f.Close()
		if _, err = io.WriteString(f, content); err == nil {
			//log.Debug(err)
			return true
		} else {
			return false
		}
	} else {
		//log.Warn(err)
		return false
	}

}

// GBKToUTF 作用是将GBK编码的字符串转换为UTF-8编码的字符串
func (this *Common) GBKToUTF(str string) string {
	decoder := mahonia.NewDecoder("GBK")
	if decoder != nil {
		if str, ok := decoder.ConvertStringOK(str); ok {
			return str
		}
	}
	return str
}

// 在本地执行cmd列表里的命令，程序在linux下将会如下执行：
// linux: bash -c cmd1 cmd2 cmd3
func (this *Common) ExecCmd(cmd []string, timeout int) string {

	var cmds []string

	if "windows" == runtime.GOOS {
		cmds = []string{
			"cmd",
			"/C",
		}
		for _, v := range cmd {
			cmds = append(cmds, v)
		}

	} else {
		cmds = []string{
			"/bin/bash",
			"-c",
		}
		for _, v := range cmd {
			cmds = append(cmds, v)
		}

	}
	result, _, _ := this.Exec(cmds, timeout, nil)
	fmt.Println(strings.Trim(result, "\n"))
	return result
}

// 如果task_id 为nil, 则在/tmp目录下使用随机数创建一个文件，并打开此文件用来保存执行cmd命令的输出；
// 调用exec.CommandContext来执行cmd
func (this *Common) Exec(cmd []string, timeout int, kw map[string]string) (string, string, int) {
	defer func() {
		if re := recover(); re != nil {
			buffer := debug.Stack()
			log.Error("Exec")
			log.Error(re)
			log.Error(string(buffer))
		}
	}()
	//var out bytes.Buffer

	var fp *os.File
	var err error
	var taskId string
	var fpath string
	var data []byte

	//生成一个任务id
	taskId = time.Now().Format("20060102150405") + fmt.Sprintf("%d", time.Now().Unix())

	// 在tmp目录下创建文件用来存储执行命令生成的输出
	home_path := ""
	if runtime.GOOS == "windows" {
		home_path = "D:\\temp\\"
	} else {
		home_path = "/tmp/"
	}

	fpath = home_path + taskId + fmt.Sprintf("_%d", random.New(random.NewSource(time.Now().UnixNano())).Intn(60))
	fp, err = os.OpenFile(fpath, os.O_CREATE|os.O_RDWR, 0666)

	if err != nil {
		log.Error(err)
		return "", err.Error(), -1
	}
	defer fp.Close()

	// golang 执行操作系统上的脚本或者命令
	duration := time.Duration(timeout) * time.Second
	if timeout == -1 {
		duration = time.Duration(60*60*24*365) * time.Second
	}
	ctx, _ := context.WithTimeout(context.Background(), duration)

	var path string

	// linux 操作系统默认使用"/bin/bash -c " 模式
	var command *exec.Cmd
	command = exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	if "windows" == runtime.GOOS {
		//		command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

		if len(cmd) > 2 {
			cc := strings.Split(cmd[2], " ")
			if cc[0] == "powershell" {
				os.Mkdir(home_path+"/"+"tmp", 0777)
				path = home_path + "/" + "tmp" + "/" + this.GetUUID() + ".ps1"
				ioutil.WriteFile(path, []byte(strings.Join(cc[1:], " ")), 0777)
				command = exec.CommandContext(ctx, "powershell", []string{path}...)
			}
		}
	}
	// 脚本执行后输出到fp中，也就是上面创建的临时文件
	command.Stdin = os.Stdin
	command.Stdout = fp
	command.Stderr = fp

	// 清理创建的fpath文件 和 path文件(windows下执行powershell才会生成)
	RemoveFile := func() {
		fp.Close()
		if path != "" {
			os.Remove(path)
		}
		if fpath != "" {
			os.Remove(fpath)
		}
	}
	_ = RemoveFile
	// 函数退出前，把flag 改为false, 即停止线程读取fpath文件内容
	// 删除fpath 和 path变量指向的文件
	defer RemoveFile()
	// 执行command命令
	err = command.Run()
	// 如果command执行出错，则将命令刷入日志文件中
	// fp文件保存数据
	if err != nil {
		if len(kw) > 0 {
			log.Info(kw)
			log.Error("error:"+err.Error(), "\ttask_id:"+fpath, "\tcmd:"+cmd[2], "\tcmd:"+strings.Join(cmd, " "))
		} else {
			log.Info("task_id:"+fpath, "\tcmd:"+cmd[2], "\tcmd:"+strings.Join(cmd, " "))
		}
		log.Flush()
		fp.Sync()
		//fp.Seek(0, 2)
		data, err = ioutil.ReadFile(fpath)
		if err != nil {
			log.Error(err)
			log.Flush()
			return string(data), err.Error(), -1
		}
		return string(data), "", -1
	}
	status := -1
	// 获取command 命令执行退出状态码并赋值给status，正常退出码赋值为0
	sysTemp := command.ProcessState
	if sysTemp != nil {
		status = sysTemp.Sys().(syscall.WaitStatus).ExitStatus()
	}
	//fp.Seek(0, 2)
	// 将内存中fp的数据输入文件中
	fp.Sync()
	// 读取fpath文件内容
	data, err = ioutil.ReadFile(fpath)
	// 如果操作系统是windows，则将内容使用GBK解码，并最终将执行结果\""\执行状态返回
	if this.IsWindows() {
		decoder := mahonia.NewDecoder("GBK")
		if decoder != nil {
			if str, ok := decoder.ConvertStringOK(string(data)); ok {
				return str, "", status
			}
		}
	}
	// 如果打开文件失败，则返回data数据
	if err != nil {
		log.Error(err, cmd)
		return string(data), err.Error(), -1
	}

	// 最后返回data数据 “” 和command命令退出码
	return string(data), "", status
}

// IsWindows 判断是否为windows操作系统
func (this *Common) IsWindows() bool {

	if "windows" == runtime.GOOS {
		return true
	}
	return false

}

// Color 将输入字符串使用指定颜色进行打印返回
// m为输入字符串内容；c为字符串指定的颜色
func (this *Common) Color(m string, c string) string {
	color := func(m string, c string) string {
		colorMap := make(map[string]string)
		if c == "" {
			c = "green"
		}
		black := fmt.Sprintf("\033[30m%s\033[0m", m)
		red := fmt.Sprintf("\033[31m%s\033[0m", m)
		green := fmt.Sprintf("\033[32m%s\033[0m", m)
		yello := fmt.Sprintf("\033[33m%s\033[0m", m)
		blue := fmt.Sprintf("\033[34m%s\033[0m", m)
		purple := fmt.Sprintf("\033[35m%s\033[0m", m)
		white := fmt.Sprintf("\033[37m%s\033[0m", m)
		glint := fmt.Sprintf("\033[5;31m%s\033[0m", m)
		colorMap["black"] = black
		colorMap["red"] = red
		colorMap["green"] = green
		colorMap["yello"] = yello
		colorMap["yellow"] = yello
		colorMap["blue"] = blue
		colorMap["purple"] = purple
		colorMap["white"] = white
		colorMap["glint"] = glint
		if v, ok := colorMap[c]; ok {
			return v
		} else {
			return colorMap["green"]
		}
	}
	return color(m, c)
}

// JsonEncodePretty 会尝试将传入的接口类型的变量编码成json格式输出
func (this *Common) JsonEncodePretty(o interface{}) string {

	resp := ""
	switch o.(type) {
	case map[string]interface{}:
		if data, err := json.Marshal(o); err == nil {
			resp = string(data)
		}
	case map[string]string:
		if data, err := json.Marshal(o); err == nil {
			resp = string(data)
		}
	case []interface{}:
		if data, err := json.Marshal(o); err == nil {
			resp = string(data)
		}
	case []string:
		if data, err := json.Marshal(o); err == nil {
			resp = string(data)
		}
	case string:
		resp = o.(string)

	default:
		if data, err := json.Marshal(o); err == nil {
			resp = string(data)
		}

	}
	var v interface{}
	if ok := json.Unmarshal([]byte(resp), &v); ok == nil {
		if buf, ok := json.MarshalIndent(v, "", "  "); ok == nil {
			resp = string(buf)
		}
	}
	return resp

}
