package worker

import (
	"fmt"
	"os"
	"time"

	"github.com/dsaidgovsg/registrywatcher/client"
	"github.com/dsaidgovsg/registrywatcher/log"
	"github.com/dsaidgovsg/registrywatcher/utils"
	"github.com/spf13/viper"
)

type WatcherWorker struct {
	conf         *viper.Viper
	pollInterval time.Duration
	repoName     string
	clients      *client.Clients
}

func InitializeWatcherWorker(conf *viper.Viper, pollInterval time.Duration,
	repoName string, clients *client.Clients) *WatcherWorker {
	ww := WatcherWorker{
		pollInterval: pollInterval,
		conf:         conf,
		repoName:     repoName,
		clients:      clients,
	}
	return &ww
}

func (ww *WatcherWorker) Run() {
	ww.initialize()
	for {
		ww.runOnce()
		time.Sleep(ww.pollInterval)
	}
}

func (ww *WatcherWorker) initialize() {
	ww.clients.PopulateCaches(ww.repoName)
}

func (ww *WatcherWorker) runOnce() {
	shouldDeploy, err := ww.clients.ShouldDeploy(ww.repoName)
	if err != nil {
		return
	}
	originalTag, err := ww.clients.GetFormattedPinnedTag(ww.repoName)
	if err != nil || !shouldDeploy {
		return
	}

	// quick way to determine if SHA changes prompted the auto deployment
	// proper way is to compare the docker content digest inside ./clients/nomad_client.go
	tagToDeploy, err := ww.clients.GetFormattedPinnedTag(ww.repoName)
	if err != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't fetch formatted pinned tag to post slack update for %s", ww.repoName), err)
		return
	} else if tagToDeploy == originalTag {
		utils.PostSlackUpdate(ww.conf, fmt.Sprintf("Update: the SHA of tag `%s` in `%s` changed. Auto deployment will happen shortly.", tagToDeploy, ww.repoName))
	}

	log.LogAppInfo(fmt.Sprintf("Auto deploying tag %s for repo %s", tagToDeploy, ww.repoName))
	if _, ok := os.LookupEnv("DEBUG"); !ok {
		ww.clients.DeployPinnedTag(ww.conf, ww.repoName)
	}
}
