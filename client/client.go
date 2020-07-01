package client

import (
	"fmt"
	"reflect"
	"sort"
	"sync"
	"testing"

	"github.com/dsaidgovsg/registrywatcher/log"
	"github.com/dsaidgovsg/registrywatcher/utils"
	nomad "github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/testutil"
	"github.com/spf13/viper"
)

type Clients struct {
	NomadClient          *NomadClient
	DockerRegistryClient *DockerRegistryClient
	PostgresClient       *PostgresClient
	DockerTags           sync.Map
	DigestMap            sync.Map

	// for test usage only
	NomadServer *testutil.TestServer
}

func SetUpClients(conf *viper.Viper) *Clients {
	postgresClient, err := InitializePostgresClient(conf)
	if err != nil {
		panic(fmt.Errorf("starting postgres client failed: %v", err))
	}
	dockerClient := InitializeDockerRegistryClient(conf)

	// caching fields
	dockerTags := sync.Map{}
	digestMap := sync.Map{}
	for _, repoName := range conf.GetStringSlice("watched_repositories") {
		dockerTags.Store(repoName, []string{})
		digestMap.Store(repoName, "")
	}
	clients := Clients{
		NomadClient:          InitializeNomadClient(conf),
		PostgresClient:       postgresClient,
		DockerRegistryClient: dockerClient,
		DockerTags:           dockerTags,
		DigestMap:            digestMap,
	}
	return &clients
}

func SetUpTestClients(t *testing.T, conf *viper.Viper) *Clients {
	postgresClient, err := InitializePostgresClient(conf)
	if err != nil {
		panic(fmt.Errorf("starting postgres client failed: %v", err))

	}

	// Create server
	config := nomad.DefaultConfig()
	ns := testutil.NewTestServer(t, nil)
	config.Address = "http://" + ns.HTTPAddr

	// Create client
	client, err := nomad.NewClient(config)
	if err != nil {
		panic(fmt.Errorf("starting nomad client failed: %v", err))
	}
	nc := NomadClient{
		nc:   client,
		conf: conf,
	}

	// caching fields
	dockerTags := sync.Map{}
	digestMap := sync.Map{}
	for _, repoName := range conf.GetStringSlice("watched_repositories") {
		dockerTags.Store(repoName, []string{})
		digestMap.Store(repoName, "")
	}

	clients := Clients{
		NomadClient:          &nc,
		NomadServer:          ns,
		PostgresClient:       postgresClient,
		DockerRegistryClient: InitializeDockerRegistryClient(conf),
		DockerTags:           dockerTags,
		DigestMap:            digestMap,
	}
	return &clients
}

func (client *Clients) GetCachedTags(repoName string) ([]string, error) {
	rtn, ok := client.DockerTags.Load(repoName)
	if !ok {
		return []string{}, fmt.Errorf("Couldn't read tags from cache error")
	}
	rtnSlice, ok := rtn.([]string)

	if !ok {
		return []string{}, fmt.Errorf(
			"Unsupported type %v, type stored in tags cache should be string slice", rtn)
	} else {
		return rtnSlice, nil
	}
}

func (client *Clients) GetCachedTagDigest(repoName string) (string, error) {
	rtn, ok := client.DigestMap.Load(repoName)
	if !ok {
		return "", fmt.Errorf("Couldn't read tag digest from cache error")
	}
	rtnString, ok := rtn.(string)

	if !ok {
		return "", fmt.Errorf(
			"Unsupported type %v, type stored in digest cache should be string", rtn)
	} else {
		return rtnString, nil
	}
}

// fetches the CACHED pinned tag
func (client *Clients) GetFormattedPinnedTag(repoName string) (string, error) {
	pinnedTag, err := client.PostgresClient.GetPinnedTag(repoName)
	if err != nil {
		return "", err
	}
	if pinnedTag == "" {
		tags, err := client.GetCachedTags(repoName)
		if err != nil {
			return "", err
		}
		pinnedTag, err = utils.GetLatestReleaseTag(tags)
	}
	return pinnedTag, err
}

func (client *Clients) DeployPinnedTag(conf *viper.Viper, repoName string) {
	pinnedTag, err := client.GetFormattedPinnedTag(repoName)
	if err != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't fetch pinned tag while deploying pinned tag for %s", repoName), err)
		return
	}
	jobID := utils.GetRepoNomadJob(conf, repoName)
	taskName := utils.GetRepoNomadTaskName(conf, repoName)
	client.NomadClient.UpdateNomadJobTag(jobID, repoName, taskName, pinnedTag)
}

func (client *Clients) PopulateCaches(repoName string) {
	// populate tags
	tags, err := client.getSHATags(repoName)
	if err != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't fetch docker tags from registry while populating cache for %s", repoName), err)
		return
	}
	validTags := utils.FilterSHATags(tags)
	client.updateTagsCache(repoName, validTags)

	// populate digest
	pinnedTag, err := client.GetFormattedPinnedTag(repoName)
	if err != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't fetch pinned tag while populating cache for %s", repoName), err)
	}
	tagDigest, err := client.DockerRegistryClient.GetTagDigest(repoName, pinnedTag)
	client.updateDigestCache(repoName, tagDigest)
	if err != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't fetch tag digest from registry while populating cache for %s", repoName), err)
	}
}

func (client *Clients) isNewReleaseTagAvailable(repoName string) bool {
	registryTags, err := client.getSHATags(repoName)
	if err != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't fetch docker tags from registry checking if new release available for for %s", repoName), err)
		return false
	}
	if len(registryTags) == 0 {
		return false
	}

	cachedTags, err := client.GetCachedTags(repoName)
	if err != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't fetch tags from cache while checking if new release available for for %s", repoName), err)
		return false
	}

	if reflect.DeepEqual(registryTags, cachedTags) {
		return false
	}

	// a new versioned tag doesn't necessarily mean it's the latest
	latestTagOld, err1 := utils.GetLatestReleaseTag(cachedTags)
	if err1 != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't fetch latest tag from registry while checking if new release available for %s", repoName), err)
		return false
	}

	latestTagNew, err2 := utils.GetLatestReleaseTag(registryTags)
	if err2 != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't fetch latest tag from cache while checking if new release available for for %s", repoName), err)
		return false
	}

	if utils.TagToNumber(latestTagNew) > utils.TagToNumber(latestTagOld) {
		return true
	}

	return false
}

// fetches from docker registry
func (client *Clients) getSHATags(repoName string) ([]string, error) {
	tags, err := client.DockerRegistryClient.GetAllTags(repoName)
	if len(tags) == 0 {
		return []string{}, err
	}
	validTags := utils.FilterSHATags(tags)
	sort.Strings(validTags)
	return validTags, nil
}

func (client *Clients) updateTagsCache(repoName string, tags []string) {
	client.DockerTags.Store(repoName, tags)
}

func (client *Clients) updateDigestCache(repoName string, digest string) {
	client.DigestMap.Store(repoName, digest)
}

func (client *Clients) isPinnedTagDeployed(conf *viper.Viper, repoName string) (bool, error) {
	jobID := utils.GetRepoNomadJob(conf, repoName)
	deployedTag, err := client.NomadClient.GetNomadJobTag(jobID, repoName)
	if err != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't fetch nomad job tag while checking deployed tag for %s", repoName), err)
		return false, err
	}
	pinnedTag, err := client.GetFormattedPinnedTag(repoName)
	if err != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't fetch pinned tag while checking deployed tag for %s", repoName), err)
		return false, err
	}
	return deployedTag == pinnedTag, nil
}

func (client *Clients) isTagDigestChanged(repoName string) (bool, error) {
	pinnedTag, err := client.GetFormattedPinnedTag(repoName)
	if err != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't fetch pinned tag while checking if it was changed for %s", repoName), err)
		return false, err
	}
	tagDigest, err := client.DockerRegistryClient.GetTagDigest(repoName, pinnedTag)
	if err != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't fetch tag digest from registry while checking if it was changed for %s", repoName), err)
		return false, err
	}
	cachedTagDigest, err := client.GetCachedTagDigest(repoName)
	if err != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't fetch tag digest from cache while checking if it was changed for %s", repoName), err)
		return false, err
	}

	return cachedTagDigest != tagDigest, nil
}

func (client *Clients) UpdateCaches(repoName string) {
	validTags, err := client.getSHATags(repoName)
	if err != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't fetch tags from registry while updating cache for %s", repoName), err)
		return
	}
	client.updateTagsCache(repoName, validTags)

	isDigestChanged, err := client.isTagDigestChanged(repoName)
	if err != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't check tag digest changed while updating cache for %s", repoName), err)
		return
	}
	if isDigestChanged {
		pinnedTag, err := client.GetFormattedPinnedTag(repoName)
		if err != nil {
			log.LogAppErr(fmt.Sprintf("Couldn't fetch pinned tag while updating cache for %s", repoName), err)
			return
		}
		tagDigest, err := client.DockerRegistryClient.GetTagDigest(repoName, pinnedTag)
		if err != nil {
			log.LogAppErr(fmt.Sprintf("Couldn't fetch tag digest from registry while updating cache for %s", repoName), err)
			return
		}

		// log update
		cachedTagDigest, err := client.GetCachedTagDigest(repoName)
		if err != nil {
			log.LogAppErr(fmt.Sprintf("Couldn't fetch tag digest from cache while updating cache for %s", repoName), err)
			return
		}
		log.LogAppInfo(fmt.Sprintf("cached digest: %s, new digest: %s", cachedTagDigest, tagDigest))

		client.updateDigestCache(repoName, tagDigest)
	}
}

// this function compares cached values with the actual values,
// so only update the cache after calling, not before
func (client *Clients) ShouldDeploy(repoName string) (bool, error) {
	autoDeploy, err := client.PostgresClient.GetAutoDeployFlag(repoName)
	if err != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't fetch whether to deploy flag while checking whether to deploy for %s", repoName), err)
		return false, err
	}
	if !autoDeploy {
		return false, nil
	}

	pinnedTag, err := client.PostgresClient.GetPinnedTag(repoName)
	if err != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't fetch pinned tag while checking whether to deploy for %s", repoName), err)
		return false, err
	}
	isDigestChanged, err := client.isTagDigestChanged(repoName)
	if err != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't check tag digest changed while checking whether to deploy for %s", repoName), err)
		return false, err
	}

	if (pinnedTag == "" && client.isNewReleaseTagAvailable(repoName)) || isDigestChanged {
		return true, nil
	}

	return false, nil
}
