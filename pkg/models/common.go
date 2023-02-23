package models

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	log "github.com/cihub/seelog"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"runtime"
	"strings"
)

type Common struct {
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
