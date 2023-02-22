package models

import (
	"os"
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
