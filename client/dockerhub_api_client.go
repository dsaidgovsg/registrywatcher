package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/dsaidgovsg/registrywatcher/log"
	"github.com/dsaidgovsg/registrywatcher/testutils"
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

type CheckImageResp struct {
	Results []TagWithStatus `json:"results"`
}

type TagWithStatus struct {
	Tag       string `json:"tag"`
	IsCurrent bool   `json:"is_current"`
}

type GetTagDigestResp struct {
	Results []GetTagDigestResult `json:"results"`
}

type GetTagDigestResult struct {
	Digest string          `json:"digest"`
	Tags   []TagWithStatus `json:"tags"`
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
	log.LogAppErr("in authenticate", nil)
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

func (api *DockerhubApi) CheckImageIsCurrent(repository, digest string, checkTag string) (
	*bool, error) {
	endpoint := fmt.Sprintf("/v2/namespaces/%s/repositories/%s/images/%s/tags",
		api.namespace, repository, digest)
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

	var deserialized CheckImageResp
	json.Unmarshal(body, &deserialized)

	for _, item := range deserialized.Results {
		if item.Tag == checkTag {
			return &item.IsCurrent, nil
		}
	}
	// image tag does not match the tag to be checked
	// this throws an error as the stored digest should match the tag
	return nil, errors.New(fmt.Sprintf("Digest %s does not have tag %s", digest, checkTag))
}

// We can't just use the registry GetTagDigest because the digests from the
// registry and from the dockerhub API do not match
func (api *DockerhubApi) GetTagDigestFromApi(repository string, checkTag string) (
	*string, error) {
	endpoint := fmt.Sprintf("/v2/namespaces/%s/repositories/%s/images?", api.namespace,
		repository)
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

	var deserialized GetTagDigestResp
	json.Unmarshal(body, &deserialized)

	for _, item := range deserialized.Results {
		for _, tag := range item.Tags {
			if tag.Tag == checkTag {
				if tag.IsCurrent {
					return &item.Digest, nil
				}
			}
		}
	}

	// image tag not found in repository's 100 last active tags
	return nil, errors.New(fmt.Sprintf("Tag %s not found in repository %s", checkTag, repository))
}

func InitializeDockerhubTestApi(mds *testutils.MockDockerhubServer) *DockerhubApi {
	client := DockerhubApi{
		url:       mds.Ts.URL,
		namespace: "namespace",
		username:  "username",
		secret:    "secret",
		token:     "fake token",
	}
	return &client
}
