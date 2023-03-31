package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dsaidgovsg/registrywatcher/client"
	"github.com/dsaidgovsg/registrywatcher/config"
	"github.com/dsaidgovsg/registrywatcher/log"
	"github.com/dsaidgovsg/registrywatcher/utils"
	"github.com/dsaidgovsg/registrywatcher/worker"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func main() {
	conf := config.SetUpConfig("staging")

	clients := client.SetUpClients(conf)

	SetUpWorkers(conf, clients)

	log.SetUpLogger()

	r := SetUpRouter(conf, clients)

	r.Run(conf.GetString("server_listening_address"))
}

func SetUpWorkers(conf *viper.Viper, clients *client.Clients) {
	for _, repoName := range conf.GetStringSlice("watched_repositories") {
		pollInterval, err := time.ParseDuration(conf.GetString("poll_interval"))
		if err != nil {
			panic(fmt.Errorf("starting worker for %s failed: %v", repoName, err))
		}
		ww := worker.InitializeWatcherWorker(conf, pollInterval, repoName, clients)
		go ww.Run()
	}
}

type Config struct {
	CORSAllowOrigin      string `mapstructure:"cors_allow_origin"`
	CORSAllowCredentials string `mapstructure:"cors_allow_credentials"`
	CORSAllowHeaders     string `mapstructure:"cors_allow_headers"`
	CORSAllowMethods     string `mapstructure:"cors_allow_methods"`
}

func SetUpRouter(conf *viper.Viper, clients *client.Clients) *gin.Engine {
	r := gin.Default()
	handler := Handler{
		clients: clients,
		conf:    conf,
	}

	routerConf := Config{
		CORSAllowOrigin:      "*",
		CORSAllowCredentials: "true",
		CORSAllowHeaders:     "pragma,content-type,content-length,accept-encoding,x-csrf-token,authorization,accept,origin,x-requested-with",
		CORSAllowMethods:     "GET,POST,PUT,DELETE",
	}

	r.Use(corsMiddleware(&routerConf))

	r.GET("/ping", HealthCheckHandler)
	r.POST("/tags/:repo_name/reset", handler.ResetTagHandler)
	r.POST("/tags/:repo_name", handler.DeployTagHandler)
	r.GET("/tags/:repo_name", handler.GetTagHandler)
	r.GET("/repos", handler.RepoSummaryHandler)
	r.GET("/debug/caches", handler.CacheSummaryHandler)

	return r
}

func corsMiddleware(conf *Config) gin.HandlerFunc {
	allowCreds, err := strconv.ParseBool(conf.CORSAllowCredentials)

	if err != nil {
		// Default to false if cannot parse Allow-Credentials
		allowCreds = false
	}

	return cors.New(cors.Config{
		AllowOrigins:     strings.Split(conf.CORSAllowOrigin, ","),
		AllowMethods:     strings.Split(conf.CORSAllowMethods, ","),
		AllowHeaders:     strings.Split(conf.CORSAllowHeaders, ","),
		AllowCredentials: allowCreds,
		ExposeHeaders:    []string{"Content-Length"},
		MaxAge:           12 * time.Hour,
	})
}

func HealthCheckHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

type Handler struct {
	clients *client.Clients
	conf    *viper.Viper
}

type deployBody struct {
	PinnedTag  *string `json:"pinned_tag",omitempty`
	AutoDeploy *bool   `json:"auto_deploy",omitempty`
}

func (h *Handler) ResetTagHandler(c *gin.Context) {

	pinnedTag := ""

	// check if repoName is valid
	repoName := c.Param("repo_name")
	validName := false
	for _, repo := range h.conf.GetStringSlice("watched_repositories") {
		if repoName == repo {
			validName = true
			break
		}
	}
	if !validName {
		_, registryDomain, _, _ := utils.ExtractRegistryInfo(h.conf, repoName)
		c.JSON(400, gin.H{
			"message": fmt.Sprintf("Error: The specified repo name %s is not inside the docker repository registry %s", repoName, registryDomain),
		})
		return
	}

	// if originalTag == pinnedTag, just terminate early
	originalTag, err := h.clients.PostgresClient.GetPinnedTag(repoName)
	if originalTag == pinnedTag {
		c.JSON(200, gin.H{
			"message": fmt.Sprintf("pinned_tag is already %s", pinnedTag),
		})
		return
	}

	// update auto deployment
	_ = h.clients.PostgresClient.UpdateAutoDeployFlag(repoName, true)

	// update tag
	err = h.clients.PostgresClient.UpdatePinnedTag(repoName, pinnedTag)

	if err != nil {
		_ = h.clients.PostgresClient.UpdatePinnedTag(repoName, originalTag)
		c.JSON(400, gin.H{
			"message": fmt.Sprintf("Error: Failed to update pinned tag, %s", err),
		})
	} else {
		log.LogAppInfo(fmt.Sprintf("Updated pinned_tag for repo %s from %s to %s succesfully, deployment of pinned_tag will happen shortly", repoName, originalTag, pinnedTag))
		h.clients.DeployPinnedTag(h.conf, repoName)
		c.JSON(200, gin.H{
			"message": fmt.Sprintf("Deploying to %s", pinnedTag),
		})
	}
}

func (h *Handler) DeployTagHandler(c *gin.Context) {

	// check if repoName is valid
	repoName := c.Param("repo_name")
	validName := false
	for _, repo := range h.conf.GetStringSlice("watched_repositories") {
		if repoName == repo {
			validName = true
			break
		}
	}
	if !validName {
		_, registryDomain, _, _ := utils.ExtractRegistryInfo(h.conf, repoName)
		c.JSON(400, gin.H{
			"message": fmt.Sprintf("Error: The specified repo name %s is not inside the docker repository registry %s", repoName, registryDomain),
		})
		return
	}

	// check that either pinnedTag or autoDeploy is given
	var deployBody deployBody
	err := c.BindJSON(&deployBody)
	if err != nil {
		c.JSON(400, gin.H{
			"message": fmt.Sprintf("Error: %s", err),
		})
		return
	} else if deployBody.PinnedTag == nil && deployBody.AutoDeploy == nil {
		c.JSON(400, gin.H{
			"message": "Either pinned_tag or auto_deploy must be specified.",
		})
		return
	}

	var newAutoDeployFlag bool
	// set autoDeploy if its present
	if deployBody.AutoDeploy != nil {
		newAutoDeployFlag = *deployBody.AutoDeploy
		currentAutoDeployFlag, _ := h.clients.PostgresClient.GetAutoDeployFlag(repoName)
		if newAutoDeployFlag != currentAutoDeployFlag {
			_ = h.clients.PostgresClient.UpdateAutoDeployFlag(repoName, newAutoDeployFlag)
			var msg string
			if newAutoDeployFlag {
				msg = fmt.Sprintf("Turned on auto deployment for repo `%s`", repoName)
			} else {
				msg = fmt.Sprintf("Turned off auto deployment for repo `%s`", repoName)
			}
			utils.PostSlackUpdate(h.conf, msg)
			log.LogAppInfo(msg)
		} else {
			log.LogAppInfo(fmt.Sprintf("Auto deployment is already set to %s", strconv.FormatBool(newAutoDeployFlag)))
		}
	}

	// exit if only autoDeploy in body
	if deployBody.PinnedTag == nil {
		c.JSON(200, gin.H{
			"message": fmt.Sprintf("Auto deployment set to %s", strconv.FormatBool(newAutoDeployFlag)),
		})
		return
	}


	// check if tag is valid
	pinnedTag := *deployBody.PinnedTag
	tags, err := h.clients.DockerRegistryClient.GetAllTags(repoName)
	if err != nil || !utils.IsTagDeployable(pinnedTag, tags) {
		c.JSON(400, gin.H{
			"message": fmt.Sprintf("Error: The specified pinned_tag %s is not inside the docker repository registry %s", pinnedTag, repoName),
		})
		return
	}

	// can terminate early if originalTag == pinnedTag
	originalTag, err := h.clients.PostgresClient.GetPinnedTag(repoName)
	if originalTag == pinnedTag {
		h.clients.DeployPinnedTag(h.conf, repoName)
		c.JSON(200, gin.H{
			"message": fmt.Sprintf("Deploying to %s", pinnedTag),
		})
		return
	}

	// update tag
	err = h.clients.PostgresClient.UpdatePinnedTag(repoName, pinnedTag)

	if err != nil {
		_ = h.clients.PostgresClient.UpdatePinnedTag(repoName, originalTag)
		c.JSON(400, gin.H{
			"message": fmt.Sprintf("Error: Failed to update pinned tag, %s", err),
		})
	} else {
		log.LogAppInfo(fmt.Sprintf("Updated pinned_tag for repo %s from %s to %s succesfully, deployment of pinned_tag will happen shortly", repoName, originalTag, pinnedTag))
		h.clients.DeployPinnedTag(h.conf, repoName)
		c.JSON(200, gin.H{
			"message": fmt.Sprintf("Deploying to %s", pinnedTag),
		})
	}
}

func (h *Handler) GetTagHandler(c *gin.Context) {

	repoName := c.Param("repo_name")

	invalidRepoName := true
	for _, repo := range h.conf.GetStringSlice("watched_repositories") {
		if repo == repoName {
			invalidRepoName = false
			break
		}
	}
	if invalidRepoName {
		c.JSON(400, gin.H{
			"message": fmt.Sprintf("Error: Repo %s is not being watched", repoName),
		})
		return
	}

	tag, err := h.clients.PostgresClient.GetPinnedTag(repoName)
	var rtn string
	if tag == "" {
		tags, err := h.clients.DockerRegistryClient.GetAllTags(repoName)
		rtn, err = utils.GetLatestReleaseTag(tags)
		if err != nil {
			c.JSON(400, gin.H{
				"message": fmt.Sprintf("No valid tags %s for repo %s, err: %s", tags, repoName, err),
			})
		}
	} else {
		rtn = tag
	}

	if err != nil {
		c.JSON(400, gin.H{
			"message": fmt.Sprintf("Unable to fetch repo %s tag, err: %s", repoName, err),
		})
	} else {
		c.JSON(200, gin.H{
			"repo_tag": rtn,
		})
	}
}

func (h *Handler) RepoSummaryHandler(c *gin.Context) {

	rtn := map[string]map[string]interface{}{}

	tagMap, err := h.clients.PostgresClient.GetAllTags()
	for _, repoName := range h.conf.GetStringSlice("watched_repositories") {
		var tag string
		if _, ok := tagMap[repoName]; ok {
			tag = tagMap[repoName]
		} else {
			log.LogAppErr(fmt.Sprintf("Couldn't fetch tag from database for endpoint summary handler for repo %s", repoName), err)
			continue
		}
		tags, err := h.clients.GetCachedTags(repoName)
		if err != nil {
			log.LogAppErr(fmt.Sprintf("Couldn't fetch tags from cache for endpoint summary handler for repo %s", repoName), err)
			continue
		}
		tagValue, err := h.clients.GetFormattedPinnedTag(repoName)
		if err != nil {
			log.LogAppErr(fmt.Sprintf("Couldn't fetch pinned tag for endpoint summary handler for repo %s", repoName), err)
			continue
		}
		autoDeployFlag, err := h.clients.PostgresClient.GetAutoDeployFlag(repoName)
		if err != nil {
			log.LogAppErr(fmt.Sprintf("Couldn't fetch auto deploy flag for endpoint summary handler for repo %s", repoName), err)
			continue
		}
		rtn[repoName] = map[string]interface{}{
			"pinned_tag":       tag,
			"pinned_tag_value": tagValue,
			"tags":             tags,
			"auto_deploy":      autoDeployFlag,
		}
	}

	c.JSON(200, rtn)
}

func (h *Handler) CacheSummaryHandler(c *gin.Context) {

	rtn := map[string]map[string]interface{}{}

	for _, repoName := range h.conf.GetStringSlice("watched_repositories") {
		cachedTagDigest, _ := h.clients.GetCachedTagDigest(repoName)
		tags, _ := h.clients.GetCachedTags(repoName)
		rtn[repoName] = map[string]interface{}{
			"cached_tags":   tags,
			"cached_digest": cachedTagDigest,
		}
	}

	c.JSON(200, rtn)
}
