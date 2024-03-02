package client

import (
	"fmt"

	"github.com/dsaidgovsg/registrywatcher/registry"
	"github.com/dsaidgovsg/registrywatcher/utils"
	"github.com/spf13/viper"
)

type DockerRegistryClient struct {
	Hubs map[string]registry.Registry
	conf *viper.Viper
}

func InitializeDockerRegistryClient(conf *viper.Viper) *DockerRegistryClient {
	watchedRepositories := conf.GetStringSlice("watched_repositories")
	drc := DockerRegistryClient{
		Hubs: make(map[string]registry.Registry, len(watchedRepositories)),
	}
	for _, repoName := range watchedRepositories {
		registryScheme, registryDomain, registryPrefix, registryAuth := utils.ExtractRegistryInfo(conf, repoName)
		registryUrl := fmt.Sprintf("%s://%s", registryScheme, registryDomain)
		username, password, err := utils.DecodeAuthString(registryAuth)
		if err != nil {
			panic(fmt.Errorf("docker auth string not valid: %v", err))
		}
		scope := fmt.Sprintf("repository:%s/%s:pull,push", registryPrefix, repoName)
		var hub *registry.Registry
		hub, err = registry.New(registryUrl, scope, username, password)
		if err != nil {
			panic(fmt.Errorf("starting docker registry client failed: %v", err))
		}
		drc.Hubs[repoName] = *hub
	}

	drc.conf = conf
	return &drc
}

func (e *DockerRegistryClient) GetAllTags(repoName string) ([]string, error) {
	_, _, registryPrefix, _ := utils.ExtractRegistryInfo(e.conf, repoName)
	repoRegistry := e.Hubs[repoName]
	tags, err := repoRegistry.Tags(fmt.Sprintf("%s/%s", registryPrefix, repoName))
	return tags, err
}
