package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/dsaidgovsg/registrywatcher/log"
	"github.com/spf13/viper"
)

type DockerhubApi struct {
	url string
}

type AuthenticateResp struct {
	Token string `json:"token"`
}

func InitializeDockerhubApi(conf *viper.Viper) *DockerhubApi {
	client := DockerhubApi{
		url: conf.GetString("dockerhub_url"),
	}
	return &client
}

func (api *DockerhubApi) Authenticate(username string, pw string) (*string, error) {
	addr := fmt.Sprintf("%s%s", api.url, "/v2/users/login")
	log.LogAppInfo(fmt.Sprintf("dockerhub.users.login url=%s user=%s", addr, username))

	data := url.Values{}
	data.Set("username", username)
	data.Set("password", pw)
	b := bytes.NewBufferString(data.Encode())

	req, err := http.NewRequest("POST", addr, b)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var deserialized AuthenticateResp
	json.Unmarshal(body, &deserialized)

	return &deserialized.Token, nil
}
