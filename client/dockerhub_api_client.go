package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/dsaidgovsg/registrywatcher/log"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type DockerhubApi struct {
	url       string
	namespace string
	token     string
	username  string
	secret    string
}

type AuthenticateResp struct {
	Token string `json:"token"`
}

type GetRepoTagsResp struct {
	Results []RepoTag `json:"results"`
}

type RepoTag struct {
	// Tag string `json:"tag"`
	// IsCurrent bool   `json:"is_current"`
	Name   string      `json:"name"`
	Images []RepoImage `json:"images"`
}

type RepoImage struct {
	Digest string `json:"digest"`
}

type GetTagDigestResp struct {
	Results []GetTagDigestResult `json:"results"`
}

type GetTagDigestResult struct {
	Digest string    `json:"digest"`
	Tags   []RepoTag `json:"tags"`
}

func InitializeDockerhubApi(conf *viper.Viper) (*DockerhubApi, error) {
	client := DockerhubApi{
		url:       conf.GetString("dockerhub_url"),
		namespace: conf.GetString("dockerhub_namespace"),
		username:  conf.GetString("dockerhub_username"),
		secret:    conf.GetString("dockerhub_secret"),
		token:     "",
	}

	jwt, err := client.Authenticate()
	if err != nil {
		return &client, err
	}

	client.token = *jwt
	return &client, nil
}

func (api *DockerhubApi) Authenticate() (*string, error) {
	addr := fmt.Sprintf("%s%s", api.url, "/v2/users/login")
	log.LogAppInfo(fmt.Sprintf("dockerhub.users.login url=%s", addr))

	data := map[string]string{"username": api.username, "password": api.secret}
	jsonData, err := json.Marshal(data)

	req, err := http.NewRequest("POST", addr, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.LogAppInfo("Error obtaining JWT")
		errMsg := fmt.Sprintf("Response status %d message %s", resp.StatusCode, string(body))
		return nil, errors.New(errMsg)
	}

	var deserialized AuthenticateResp
	json.Unmarshal(body, &deserialized)

	return &deserialized.Token, nil
}

func (api *DockerhubApi) CheckImageIsCurrent(repository, digest string, tagName string) (
	*bool, error) {
	endpoint := fmt.Sprintf("/v2/namespaces/%s/repositories/%s/tags", api.namespace, repository)
	addr := fmt.Sprintf("%s%s", api.url, endpoint)
	log.LogAppInfo(fmt.Sprintf("dockerhub check if image is current, url=%s", addr))

	req, err := http.NewRequest("GET", addr, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.token))
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Obtain JWT if it has expired
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		log.LogAppInfo("Obtaining new JWT")
		jwt, err := api.Authenticate()
		if err != nil {
			return nil, err
		}

		api.token = *jwt
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.token))
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Response status %d message %s", resp.StatusCode, string(body))
		return nil, errors.New(errMsg)
	}

	var deserialized GetRepoTagsResp
	json.Unmarshal(body, &deserialized)

	for _, item := range deserialized.Results {
		if item.Name == tagName {
			isCurrent := item.Images[0].Digest == digest
			return &isCurrent, nil
		}
	}
	// image tag does not match the tag to be checked
	// this means the cached digest belongs to a previous tag
	// return false (not current) because we want the cache to be updated
	log.LogAppInfo(fmt.Sprintf(fmt.Sprintf("Digest %s does not have tag %s", digest, tagName)))
	isCurrent := false
	return &isCurrent, nil
}

// Note that the digest returned from the Dockerhub API does not match
// the digest from docker registry manifest V2 API
func (api *DockerhubApi) GetTagDigestFromApi(repository string, tagName string) (
	*string, error) {
	endpoint := fmt.Sprintf("/v2/namespaces/%s/repositories/%s/tags", api.namespace, repository)
	queryParams := fmt.Sprintf("currently_tagged=%s&page_size=%s&ordering=%s", "true",
		"100", "-last_activity")

	addr := fmt.Sprintf("%s%s%s", api.url, endpoint, queryParams)
	log.LogAppInfo(fmt.Sprintf("dockerhub get tag digest url=%s", addr))

	req, err := http.NewRequest("GET", addr, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.token))
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Obtain JWT if it has expired
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		log.LogAppInfo("Obtaining new JWT")
		jwt, err := api.Authenticate()
		if err != nil {
			return nil, err
		}

		api.token = *jwt
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.token))
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Response status %d message %s", resp.StatusCode, string(body))
		return nil, errors.New(errMsg)
	}

	var deserialized GetRepoTagsResp
	json.Unmarshal(body, &deserialized)

	for _, item := range deserialized.Results {
		if item.Name == tagName {
			return &item.Images[0].Digest, nil
		}
	}

	// image tag not found in repository's 100 last active tags
	return nil, errors.New(fmt.Sprintf("Tag %s not found in repository %s", tagName, repository))
}
