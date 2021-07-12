package client

import (
	"github.com/spf13/viper"
)

type DockerhubApi struct {
	url		string
}


func InitializeDockerhubApi(conf *viper.Viper) (*DockerhubApi) {
	client := DockerhubApi{
		url: conf.GetString("dockerhub_url"),
	}
	return &client
}
