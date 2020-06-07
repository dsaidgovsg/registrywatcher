// +build integration

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dsaidgovsg/registrywatcher/client"
	"github.com/dsaidgovsg/registrywatcher/utils"
	"github.com/stretchr/testify/assert"
)

type RepoSummaryResult struct {
	Map map[string]string `json:"testrepo" binding:"required"`
}

func TestRepoSummaryHandler(t *testing.T) {
	te := client.SetUpClientTest(t)
	router := SetUpRouter(te.Conf, te.Clients)
	defer te.TearDown()
	var rtn RepoSummaryResult

	// populate with new tags
	tags := []string{"v1.0.0"}
	for _, tag := range tags {
		te.PushNewTag(tag, "latest")
	}
	te.Clients.PopulateCaches(te.TestRepoName)

	request, _ := http.NewRequest("GET", "/repos", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	_ = json.NewDecoder(response.Body).Decode(&rtn)

	// tag is latest by default, which is v1.0.0
	tag := rtn.Map["pinned_tag_value"]
	assert.Equal(t, "v1.0.0", tag, "OK tags correspond")

	// update pinnedTag
	newTag := "v0.1.0"
	te.UpdatePinnedTag(newTag)

	request, _ = http.NewRequest("GET", "/repos", nil)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	_ = json.NewDecoder(response.Body).Decode(&rtn)

	// after setting tag to newTag, endpoint should reflect the updated status
	tag = rtn.Map["pinned_tag_value"]
	assert.Equal(t, newTag, tag, "OK tags correspond")
}

type GetTagResult struct {
	Tag string `json:"repo_tag" binding:"required"`
}

// also tests RepinnedTagHandler since setUp and tearDown is expensive
func TestGetTagHandler(t *testing.T) {
	te := client.SetUpClientTest(t)
	router := SetUpRouter(te.Conf, te.Clients)
	defer te.TearDown()
	var rtn GetTagResult

	// populate with new tags
	tags := []string{"v1.0.0"}
	for _, tag := range tags {
		te.PushNewTag(tag, "latest")
	}

	request, _ := http.NewRequest("GET", fmt.Sprintf("/tags/%s", te.TestRepoName), nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	_ = json.NewDecoder(response.Body).Decode(&rtn)

	// tag is latest by default, which is v1.0.0
	tag := rtn.Tag
	assert.Equal(t, "v1.0.0", tag, "OK tags correspond")

	// update pinnedTag
	newTag := "v0.0.1"
	te.UpdatePinnedTag(newTag)

	request, _ = http.NewRequest("GET", fmt.Sprintf("/tags/%s", te.TestRepoName), nil)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	_ = json.NewDecoder(response.Body).Decode(&rtn)

	// tag is latest by default, which is v1.0.0
	tag = rtn.Tag
	assert.Equal(t, newTag, tag, "OK tags correspond")

	// push new tag
	newTag = "v2.0.0"
	te.PushNewTag(newTag, "latest")

	// reset pinnedTag
	request, _ = http.NewRequest("POST", fmt.Sprintf("/tags/%s/reset", te.TestRepoName), nil)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)

	// tag is reset to latest, which is now v2.0.0
	tag, _ = te.Clients.PostgresClient.GetPinnedTag(te.TestRepoName)
	assert.Equal(t, tag, "", "OK tags is latest")
	tags, _ = te.Clients.DockerRegistryClient.GetAllTags(te.TestRepoName)
	tagValue, _ := utils.GetLatestReleaseTag(tags)
	assert.Equal(t, newTag, tagValue, "OK latest tag is v2.0.0")

	// test querying on a repo that's not being watched
	request, _ = http.NewRequest("GET", "/tags/nonexistent-repo", nil)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	_ = json.NewDecoder(response.Body).Decode(&rtn)
	assert.Equal(t, 400, response.Code, "OK response is expected")
}

func TestDeployTagHandler(t *testing.T) {
	te := client.SetUpClientTest(t)
	router := SetUpRouter(te.Conf, te.Clients)
	defer te.TearDown()

	// populate with new tags
	tags := []string{"v1.0.0"}
	for _, tag := range tags {
		te.PushNewTag(tag, "latest")
	}

	// test with missing params (at least 1 argument must be provided)
	data := []byte(`{}`)
	request, _ := http.NewRequest("POST", fmt.Sprintf("/tags/%s", te.TestRepoName), bytes.NewBuffer(data))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	assert.Equal(t, 400, response.Code, "OK response is expected")

	// test with invalid params
	data = []byte(`{"bet_tag":"hi"}`)
	request, _ = http.NewRequest("POST", fmt.Sprintf("/tags/%s", te.TestRepoName), bytes.NewBuffer(data))
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	assert.Equal(t, 400, response.Code, "OK response is expected")

	// test with valid params of invalid type
	data = []byte(`{"pinned_tag":"hi", "auto_deploy: "true"}`)
	request, _ = http.NewRequest("POST", fmt.Sprintf("/tags/%s", te.TestRepoName), bytes.NewBuffer(data))
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	assert.Equal(t, 400, response.Code, "OK response is expected")

	// test with invalid repoName
	data = []byte(`{"pinned_tag":"v0.0.1"}`)
	request, _ = http.NewRequest("POST", "/tags/nonexistent-repo", bytes.NewBuffer(data))
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	assert.Equal(t, 400, response.Code, "OK response is expected")

	// test with invalid pinnedTag
	data = []byte(`{"pinned_tag":"v0.5.0"}`)
	request, _ = http.NewRequest("POST", fmt.Sprintf("/tags/%s", te.TestRepoName), bytes.NewBuffer(data))
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	assert.Equal(t, 400, response.Code, "OK response is expected")

	// test with valid pinnedTag
	data = []byte(`{"pinned_tag":"v1.0.0"}`)
	request, _ = http.NewRequest("POST", fmt.Sprintf("/tags/%s", te.TestRepoName), bytes.NewBuffer(data))
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	assert.Equal(t, 200, response.Code, "OK response is expected")

	// test with valid autoDeploy
	data = []byte(`{"auto_deploy":true}`)
	request, _ = http.NewRequest("POST", fmt.Sprintf("/tags/%s", te.TestRepoName), bytes.NewBuffer(data))
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	assert.Equal(t, 200, response.Code, "OK response is expected")

	// test with valid pinnedTag and pinnnedTag
	data = []byte(`{"pinned_tag":"v1.0.0", "auto_deploy":true}`)
	request, _ = http.NewRequest("POST", fmt.Sprintf("/tags/%s", te.TestRepoName), bytes.NewBuffer(data))
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	assert.Equal(t, 200, response.Code, "OK response is expected")
}
