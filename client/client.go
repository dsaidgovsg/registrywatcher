package client

import (
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/dsaidgovsg/registrywatcher/log"
	"github.com/dsaidgovsg/registrywatcher/utils"
	"github.com/spf13/viper"
)

type Clients struct {
	NomadClient          *NomadClient
	DockerRegistryClient *DockerRegistryClient
	PostgresClient       *PostgresClient
	DockerhubApi         *DockerhubApi
	DockerTags           sync.Map
	DigestMap            sync.Map
}

func SetUpClients(conf *viper.Viper) *Clients {
	postgresClient, err := InitializePostgresClient(conf)
	if err != nil {
		panic(fmt.Errorf("starting postgres client failed: %v", err))
	}
	dockerClient := InitializeDockerRegistryClient(conf)
	dockerhubApi, err := InitializeDockerhubApi(conf)
	if err != nil {
		log.LogAppErr("error initializing dockerhub API client", err)
	}

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
		DockerhubApi:         dockerhubApi,
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
	// update after deploying new sha, so it will not trigger autodeployment
	client.updateCaches(repoName)
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
	tagDigest, err := client.DockerhubApi.GetTagDigestFromApi(repoName, pinnedTag)
	if err != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't fetch tag digest from Dockerhub while populating cache for %s", repoName), err)
		return
	}
	client.updateDigestCache(repoName, *tagDigest)
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
	cachedTagDigest, err := client.GetCachedTagDigest(repoName)
	if err != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't fetch tag digest from cache while checking if it was changed for %s", repoName), err)
		return false, err
	}
	digestIsCurrent, err := client.DockerhubApi.CheckImageIsCurrent(repoName, cachedTagDigest, pinnedTag)
	if err != nil {
		log.LogAppErr("Couldn't check if tag currently points to cached image digest", err)
		return false, err
	}

	return !*digestIsCurrent, nil
}

func (client *Clients) updateCaches(repoName string) {
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
		tagDigest, err := client.DockerhubApi.GetTagDigestFromApi(repoName, pinnedTag)
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
		log.LogAppInfo(fmt.Sprintf("cached digest: %s, new digest: %s", cachedTagDigest, *tagDigest))

		client.updateDigestCache(repoName, *tagDigest)
	}
}

// this function compares cached values with the actual values,
// so only update the cache before returning non-error cases
func (client *Clients) ShouldDeploy(repoName string) (bool, error) {
	autoDeploy, err := client.PostgresClient.GetAutoDeployFlag(repoName)
	if err != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't fetch whether to deploy flag while checking whether to deploy for %s", repoName), err)
		return false, err
	}
	if !autoDeploy {
		client.updateCaches(repoName)
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
		client.updateCaches(repoName)
		return true, nil
	}
	client.updateCaches(repoName)
	return false, nil
}
