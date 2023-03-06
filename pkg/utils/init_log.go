package utils

import (
	log "github.com/cihub/seelog"
	"github.com/yin-zt/gmd/pkg/config"
	"os"
)

var (
	Logger log.LoggerInterface
)

func GetLog() log.LoggerInterface {
	os.MkdirAll("/var/log/", 07777)
	os.MkdirAll("/var/lib/cli", 07777)

	logger, err := log.LoggerFromConfigAsBytes([]byte(config.LogConfigStr))

	if err != nil {
		log.Error(err)
		panic("init log fail")
	}
	Logger = logger
	return Logger
}
