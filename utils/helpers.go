package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dsaidgovsg/registrywatcher/log"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"
)

// update README link if moving this definition
var re = regexp.MustCompile(`^v([0-9]{1,3})\.([0-9]{1,3})\.([0-9]{1,3})$`)
var findSHA = regexp.MustCompile(`[A-Fa-f0-9]{40}`)

func IsTagSHAFormat(tag string) bool {
	return findSHA.MatchString(tag)
}

func IsTagReleaseFormat(tag string) bool {
	return re.MatchString(tag)
}

func FilterReleaseTags(tags []string) []string {
	rtn := []string{}
	for _, tag := range tags {
		if IsTagReleaseFormat(tag) {
			rtn = append(rtn, tag)
		}
	}
	return rtn
}

func FilterSHATags(tags []string) []string {
	rtn := []string{}
	for _, tag := range tags {
		if !IsTagSHAFormat(tag) {
			rtn = append(rtn, tag)
		}
	}
	return rtn
}

func TagToNumber(tag string) int {
	arr := strings.Split(tag, ".")
	major, _ := strconv.Atoi(string(arr[0][1:]))
	major = major * 1000000
	minor, _ := strconv.Atoi(string(arr[1]))
	minor = minor * 1000
	patch, _ := strconv.Atoi(string(arr[2]))
	return major + minor + patch
}

// latest with respect to versioned tags
func IsTagLatest(tag string, tags []string) bool {
	if tag == "" {
		return true
	}
	// tags to numbers
	values := []int{}
	for _, tag := range tags {
		values = append(values, TagToNumber(tag))
	}
	sort.Ints(values)
	rtn := TagToNumber(tag) == values[len(values)-1]
	return rtn
}

func IsTagDeployable(checkedTag string, availableDockerTags []string) bool {
	if checkedTag == "" {
		return true
	}
	for _, tag := range availableDockerTags {
		if tag == checkedTag {
			return true
		}
	}
	return false
}

func GetLatestReleaseTag(tags []string) (string, error) {
	tags = FilterReleaseTags(tags)
	tagValueMap := map[int]string{}
	values := []int{}
	for _, tag := range tags {
		tagValueMap[TagToNumber(tag)] = tag
		values = append(values, TagToNumber(tag))
	}
	sort.Ints(values)
	if len(values) == 0 {
		return "", fmt.Errorf("No valid tag")
	}
	biggestValue := values[len(values)-1]
	return tagValueMap[biggestValue], nil
}

func CastMapOfMaps(mapOfMap interface{}) map[string]map[string]string {
	rtn := map[string]map[string]string{}
	for k1, nestedMap := range mapOfMap.(map[string]interface{}) {
		rtn[k1] = map[string]string{}
		for k2, v2 := range nestedMap.(map[string]interface{}) {
			rtn[k1][k2] = v2.(string)
		}
	}
	return rtn
}

func extractRegistryInfo(conf *viper.Viper, repoName, keyName string) string {
	repoMap := CastMapOfMaps(conf.Get("repo_map"))
	registryMap := CastMapOfMaps(conf.Get("registry_map"))
	registryName := repoMap[repoName]["registry_name"]
	repoRegistryMap := registryMap[registryName]
	return repoRegistryMap[keyName]
}

// return the scheme, domain and prefix in that order
func ExtractRegistryInfo(conf *viper.Viper, repoName string) (string, string, string, string) {
	registryScheme := extractRegistryInfo(conf, repoName, "registry_scheme")
	registryDomain := extractRegistryInfo(conf, repoName, "registry_domain")
	registryPrefix := extractRegistryInfo(conf, repoName, "registry_prefix")
	registryAuth := extractRegistryInfo(conf, repoName, "registry_auth")
	return registryScheme, registryDomain, registryPrefix, registryAuth
}

func DecodeAuthString(encoded string) (string, string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", fmt.Errorf("docker auth string not valid: %v", err)
	}
	decoded := string(data)
	arr := strings.Split(decoded, ":")
	if len(arr) != 2 {
		return "", "", fmt.Errorf("docker auth string not valid: %v", err)
	}
	username := arr[0]
	password := arr[1]
	return username, password, nil
}

func ConstructImageName(domain, prefix, repoName, tag string) string {
	return fmt.Sprintf("%s/%s/%s:%s",
		domain,
		prefix,
		repoName,
		tag,
	)
}

func GetJobOfRepo(conf *viper.Viper, repoName string) string {
	repoMap := conf.Get("repo_map").(map[string]interface{})
	jobID := repoMap[repoName].(map[string]interface{})["nomad_job_name"].(string)
	return jobID
}

const (
	green  = "#00FF00"
	red    = "#FF0000"
	orange = "#FFA500"
	yarly  = "https://i.imgur.com/LWRp6ZT.png"
	nowai  = "https://i.imgur.com/MJ5Qx8f.jpg"
)

func PostSlackUpdate(conf *viper.Viper, text string) {
	attachment := slack.Attachment{
		Color: orange,
		Text:  text,
		Ts:    json.Number(strconv.FormatInt(time.Now().Unix(), 10)),
	}
	postSlackMessage(conf, attachment)
}

func PostSlackError(conf *viper.Viper, text string) {
	attachment := slack.Attachment{
		Color:    red,
		Text:     text,
		Ts:       json.Number(strconv.FormatInt(time.Now().Unix(), 10)),
		ThumbURL: nowai,
	}
	postSlackMessage(conf, attachment)
}

func PostSlackSuccess(conf *viper.Viper, text string) {
	attachment := slack.Attachment{
		Color:    green,
		Text:     text,
		Ts:       json.Number(strconv.FormatInt(time.Now().Unix(), 10)),
		ThumbURL: yarly,
	}
	postSlackMessage(conf, attachment)
}

func postSlackMessage(conf *viper.Viper, attachment slack.Attachment) {
	if _, ok := os.LookupEnv("DEBUG"); !ok {
		msg := slack.WebhookMessage{
			Attachments: []slack.Attachment{attachment},
		}

		err := slack.PostWebhook(conf.GetString("webhook_url"), &msg)
		if err != nil {
			log.LogAppErr(fmt.Sprintf("Cannot post to slack webhook_url %s", conf.GetString("webhook_url")), err)
		}
	}
}
