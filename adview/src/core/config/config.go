package config

import (
	"strings"

	"github.com/Unknwon/goconfig"
)

//DSP Config file class

type ItemConf struct {
	//本DSP对外服务地址，用于WinNotice,Click等通讯
	QueryHost string
	QueryPort int
}

func get_str(s string) string {
	return strings.Trim(s, "\" ")
}

func NewItemConf(confFile string) (conf *ItemConf) {
	cfg, err := goconfig.LoadConfigFile(confFile)

	if nil != err {
		panic(err)
	}
	queryHost, _ := cfg.GetValue("", "query.host")
	queryPort := cfg.MustInt("", "query.port", 80)

	queryHost = get_str(queryHost)
	if len(queryHost) <= 0 {
		queryHost = "0.0.0.0"
	}
	conf = &ItemConf{
		QueryHost: get_str(queryHost),
		QueryPort: queryPort,
	}
	return
}
