package config

import (
	"flag"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"path/filepath"
	"voucher/pkg/utils"
)

type AppConfig struct {
	App App `yaml:"app"`
}

type App struct {
	Mysql      Mysql  `yaml:"mysql"`
	Redis      Redis  `yaml:"redis"`
	TimeFormat string `yaml:"timeFormat"`
	MessageQueueSize int `yaml:"messageQueueSize"`
	Port int `yaml:"port"`
}

type Mysql struct {
	User string `yaml:"user"`
	Password string `yaml:"password"`
	Dbname string `yaml:"dbname"`
	Address string `yaml:"addr"`
}

type Redis struct {
	Address string `yaml:"address"`
	Network string `yaml:"network"`
	Password string `yaml:"password"`
	Dbname int `yaml:"dbname"`
	NotAllocPreheatSize int `yaml:"notAllocPreheatSize"`
	NotRedeemPreheatSize int `yaml:"notRedeemPreheatSize"`
	ExpireTime int `yaml:"expireTime"`
}

var AppConf *AppConfig

var confFile = flag.String("c", "", "conf file location")

func init() {

	confPath := filepath.Join(utils.ExecFileDir(), "..", "config", "conf.yaml")
	if *confFile != "" {
		confPath = *confFile
	}

	file, err := ioutil.ReadFile(confPath)
	if err != nil {
		log.Fatalln(err)
	}
	AppConf = &AppConfig{}
	err = yaml.Unmarshal(file, AppConf)
}

