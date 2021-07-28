package client

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/dsaidgovsg/registrywatcher/config"
	"github.com/dsaidgovsg/registrywatcher/log"
	"github.com/dsaidgovsg/registrywatcher/testutils"
	"github.com/dsaidgovsg/registrywatcher/utils"
	nomad "github.com/hashicorp/nomad/api"
	"github.com/spf13/viper"

	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"
)

type testEngine struct {
	containerIDs []string // keep track of container IDs to be destroyed at tearDown()
	Conf         *viper.Viper
	helper       *testutils.TestHelper
	Clients      *Clients
	ImageTagMap  map[string][]string // digest: [tag, is_current]
	Ts           *httptest.Server
	TestRepoName string
}

func (te *testEngine) printState() {
	registryTags, _ := te.Clients.DockerRegistryClient.GetAllTags(te.TestRepoName)
	fmt.Println("registry tags", registryTags)

	cachedTags, _ := te.Clients.GetCachedTags(te.TestRepoName)
	fmt.Println("cached tags", cachedTags)

	pinnedTag, _ := te.Clients.GetFormattedPinnedTag(te.TestRepoName)
	registryTagDigest, _ := te.Clients.DockerhubApi.GetTagDigestFromApi(te.TestRepoName, pinnedTag)
	fmt.Println("registry tag digest", *registryTagDigest)

	cachedTagDigest, _ := te.Clients.GetCachedTagDigest(te.TestRepoName)
	fmt.Println("cached tag digest", cachedTagDigest)
}

func SetUpClientTest(t *testing.T) *testEngine {
	conf := config.SetUpConfig("test")

	te := testEngine{
		Conf:   conf,
		helper: testutils.NewTestHelper(conf),
	}

	// start registry
	regID, _, err := te.helper.StartRegistry()
	if err != nil {
		te.helper.RemoveContainer(regID)
		panic(fmt.Errorf("starting registry container failed: %v", err))
	}

	// start postgres
	pgID, err := te.helper.StartPostgres()
	if err != nil {
		te.helper.RemoveContainer(pgID, regID)
		panic(fmt.Errorf("starting postgres container failed: %v", err))
	}

	// add registry and postgres container ID to be removed later
	te.containerIDs = append(te.containerIDs, regID)
	te.containerIDs = append(te.containerIDs, pgID)

	// initialize mock server
	te.ImageTagMap = make(map[string][]string)

	// initialize the clients
	te.Clients = SetUpTestClients(t, conf)

	// we use this so much might as well keep it in the struct
	te.TestRepoName = te.Conf.GetStringSlice("watched_repositories")[0]

	// set up mock Dockerhub API and imageTag store
	router := mux.NewRouter()

	router.HandleFunc("/v2/users/login", func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte(`{"token": "test token"}`))
	}).Methods("GET")

	router.HandleFunc("/v2/namespaces/{namespace}/repositories/{repo}/images/{digest}/tags",
		func(res http.ResponseWriter, req *http.Request) {
			vars := mux.Vars(req)
			digest := vars["digest"]

			res.Header().Set("Content-Type", "application/json")
			res.WriteHeader(http.StatusOK)

			if tag, ok := te.ImageTagMap[digest]; ok {
				isCurrent, _ := strconv.ParseBool(tag[1])
				tags := []TagWithStatus{{Tag: tag[0], IsCurrent: isCurrent}}
				results := CheckImageResp{Results: tags}
				data, _ := json.Marshal(results)
				res.Write(data)
			} else {
				results := CheckImageResp{Results: nil}
				data, _ := json.Marshal(results)
				res.Write(data)
			}
		}).Methods("GET")

	router.HandleFunc("/v2/namespaces/namespace/repositories/testrepo/images",
		func(res http.ResponseWriter, req *http.Request) {
			var resSlice []GetTagDigestResult
			for image, tag := range te.ImageTagMap {
				isCurrent, _ := strconv.ParseBool(tag[1])
				tags := []TagWithStatus{{Tag: tag[0], IsCurrent: isCurrent}}
				resSlice = append(resSlice, GetTagDigestResult{Digest: image, Tags: tags})
			}
			res.Header().Set("Content-Type", "application/json")
			res.WriteHeader(http.StatusOK)
			results := GetTagDigestResp{Results: resSlice}
			data, _ := json.Marshal(results)
			res.Write(data)
		})

	ts := httptest.NewServer(router)
	te.Ts = ts

	dockerhubApi := DockerhubApi{
		url:       ts.URL,
		namespace: "namespace",
		username:  "username",
		secret:    "secret",
		token:     "fake token",
	}

	te.Clients.DockerhubApi = &dockerhubApi
	return &te
}

func (te *testEngine) RegisterJob() {
	jobID := utils.GetRepoNomadJob(te.Conf, te.TestRepoName)
	tags, _ := te.Clients.DockerRegistryClient.GetAllTags(te.TestRepoName)
	dockerImage := fmt.Sprintf("%s:%s", te.TestRepoName, tags[0])
	job := testJob(jobID, dockerImage)
	jobs := te.Clients.NomadClient.nc.Jobs()
	_, _, err := jobs.Register(job, nil)
	if err != nil {
		panic(fmt.Errorf("starting nomad job failed: %v", err))
	}
}

func testJob(jobID, dockerImage string) *nomad.Job {
	count := 1
	name := "job"
	taskName := "test"
	jobType := "service"
	region := "region1"
	return &nomad.Job{
		ID:          &jobID,
		Name:        &name,
		Type:        &jobType,
		Datacenters: []string{"dc-1"},
		Region:      &region,
		TaskGroups: []*nomad.TaskGroup{
			{
				Name:  &taskName,
				Count: &count,
				Tasks: []*nomad.Task{
					{
						Name:   "test",
						Driver: "docker",
						Config: map[string]interface{}{
							"image": dockerImage,
						},
					},
				},
			},
		},
	}

}

func (te *testEngine) TearDown() {
	te.Clients.NomadServer.Stop()
	for _, containerID := range te.containerIDs {
		if err := te.helper.RemoveContainer(containerID); err != nil {
			log.LogAppErr("Couldn't remove container", err)
		}
	}
	te.Ts.Close()
}

// named tag is what it will appear as in the docker registryDomain
// actual tag is what the tag is based on (this is for testing purposes only)
func (te *testEngine) PushNewTag(namedTag, actualTag string) {
	_, registryDomain, registryPrefix, _ := utils.ExtractRegistryInfo(te.Conf, te.TestRepoName)
	mockImageName := utils.ConstructImageName(registryDomain, registryPrefix, te.TestRepoName, namedTag)
	publicImageName := fmt.Sprintf("%s:%s", te.Conf.GetString("base_public_image"), actualTag)
	err := te.helper.AddImageToRegistry(publicImageName, mockImageName)

	imageDigest := testutils.RandSeq(10)
	te.ImageTagMap[imageDigest] = []string{namedTag, "true"}

	// make is_current false for images holding the same tag
	for image, tag := range te.ImageTagMap {
		if tag[0] == namedTag && image != imageDigest {
			tag[1] = "false"
		}
	}

	if err != nil {
		panic(fmt.Errorf("couldn't add image to registry: %v", err))
	}
}

func (te *testEngine) UpdatePinnedTag(newTag string) {
	err := te.Clients.PostgresClient.UpdatePinnedTag(te.TestRepoName, newTag)
	if err != nil {
		panic(fmt.Errorf("couldn't update postgres client pinned_tag: %v", err))
	}
}
