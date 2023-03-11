package config

import (
	log "github.com/cihub/seelog"
	"os"
	"os/user"
	"runtime"
	"strings"
)

// NewConfig 作用是创建一个Config结构体实例，包含：
// DefaultModule := "gmd";scriptPath = "/tmp/script/"
func NewConfig() *Config {

	DefaultModule := "gmd"
	var scriptPath = "/tmp/script/"

	// 通过环境变量获取"GMD_DEBUG", 如果有设置，则DEBUG = true
	if _debug := os.Getenv("GMD_DEBUG"); _debug != "" {
		DEBUG = true
	}

	// 如果操作系统为windows，scriptPath = user.HomeDir + "\\scirpt\\"
	if "windows" == runtime.GOOS {
		if user, err := user.Current(); err == nil {
			scriptPath = user.HomeDir + "\\scirpt\\"
		}
	}

	conf := &Config{
		DefaultModule: DefaultModule,
		Salt:          "",
		EtcdConf: &EtcdConf{
			Prefix: "",
			Server: []string{},
		},
		ScriptPath:    scriptPath,
		DefaultAction: "help",
		ShellStr:      "",
		Commands:      make(chan interface{}, 1000),
		Args:          os.Args,
		_ArgsSep:      "$$$$",
		_Args:         strings.Join(os.Args, "$$$$"),
	}

	if err := os.MkdirAll(conf.ScriptPath, 0777); err != nil {
		log.Error(err)
		os.Exit(-1)
	}

	return conf
}
