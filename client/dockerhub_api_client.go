package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/dsaidgovsg/registrywatcher/log"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type DockerhubApi struct {
	url       string
	namespace string
}

type AuthenticateResp struct {
	Token string `json:"token"`
}

type CheckImageResp struct {
	Results []CheckImageResults `json:"results"`
}

type CheckImageResults struct {
	Tag       string `json:"tag"`
	IsCurrent bool   `json:"is_current"`
}

func InitializeDockerhubApi(conf *viper.Viper) *DockerhubApi {
	client := DockerhubApi{
		url:       conf.GetString("dockerhub_url"),
		namespace: conf.GetString("dockerhub_namespace"),
	}
	return &client
}

func (api *DockerhubApi) Authenticate(username string, pw string) (*string, error) {
	addr := fmt.Sprintf("%s%s", api.url, "/v2/users/login")
	log.LogAppInfo(fmt.Sprintf("dockerhub.users.login url=%s", addr))

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

func (api *DockerhubApi) CheckImageIsCurrent(repository, digest string, checkTag string, jwt *string) (*bool, error) {
	endpoint := fmt.Sprintf("/v2/namespaces/%s/repositories/%s/images/%s/tags", api.namespace, repository, digest)
	addr := fmt.Sprintf("%s%s", api.url, endpoint)
	log.LogAppInfo(fmt.Sprintf("dockerhub.namespace.repositories.images url=%s", addr))

	req, err := http.NewRequest("GET", addr, nil)
	if err != nil {
		return nil, err
	}

	auth_token := fmt.Sprintf("JWT %s", *jwt)
	req.Header.Set("Authorization", auth_token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var deserialized CheckImageResp
	json.Unmarshal(body, &deserialized)

	for _, item := range deserialized.Results {
		if item.Tag == checkTag {
			return &item.IsCurrent, nil
		}
	}
	// image tag does not match the tag to be checked
	// this throws an error as the stored digest should match the tag
	return nil, errors.New(fmt.Sprintf("Digest does not have tag %s", checkTag))
}
