package main

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/yin-zt/gmd/pkg/models"
	"github.com/yin-zt/gmd/pkg/utils"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"os"
	"reflect"
)

var this = &models.Gmd{}

func init() {
	log.ReplaceLogger(utils.GetLog())
	log.Info("hhhhhhhhhhhhhhhhhh")
	log.Flush()
}

func main() {
	obj := reflect.ValueOf(this)
	module, action := "default", "gmd"
	fmt.Println("hello wor")
	log.Error("hello world")
	log.Info("霓虹")
	log.Flush()
	if len(os.Args) == 1 {
		this.Help(module, action)
		return
	}
	module = this.Util.GetModule()
	action = this.Util.GetAction()

	c := cases.Title(language.Dutch)

	if obj.MethodByName(c.String(action)).IsValid() {
		obj.MethodByName(c.String(action)).Call([]reflect.Value{reflect.ValueOf(module), reflect.ValueOf(action)})
	} else {
		obj.MethodByName("Default").Call([]reflect.Value{reflect.ValueOf(module), reflect.ValueOf(action)})
	}

}
