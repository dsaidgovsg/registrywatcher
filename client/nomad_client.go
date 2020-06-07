package client

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dsaidgovsg/registrywatcher/log"
	"github.com/dsaidgovsg/registrywatcher/utils"
	nomad "github.com/hashicorp/nomad/api"
	"github.com/spf13/viper"
)

type NomadClient struct {
	nc   *nomad.Client
	conf *viper.Viper
}

func InitializeNomadClient(conf *viper.Viper) *NomadClient {
	rtn := NomadClient{}
	config := nomad.DefaultConfig()
	client, err := nomad.NewClient(config)

	if err != nil {
		panic(fmt.Errorf("starting nomad client failed: %v", err))
	}

	rtn.nc = client
	rtn.conf = conf
	return &rtn
}

func getNomadJobTagFromTask(task *nomad.Task) string {
	rtnArray := strings.Split(task.Config["image"].(string), ":")
	return rtnArray[1]
}

func getNomadJobImageFromTask(task *nomad.Task) string {
	rtnArray := strings.Split(task.Config["image"].(string), ":")
	return rtnArray[0]
}

func (client *NomadClient) getNomadJob(jobID string) (nomad.Job, error) {
	job, _, err := client.nc.Jobs().Info(jobID, nil)
	if err != nil {
		return nomad.Job{}, err
	}
	return *job, nil
}

func (client *NomadClient) GetNomadJobTag(jobID, imageName string) (string, error) {
	job, err := client.getNomadJob(jobID)
	if err != nil {
		return "", err
	}
	var tag string
	for i, taskGroup := range job.TaskGroups {
		for j, task := range taskGroup.Tasks {
			fullImageName := getNomadJobImageFromTask(task)
			arr := strings.Split(fullImageName, "/")
			taskImageName := arr[len(arr)-1]
			if taskImageName == imageName {
				tag = getNomadJobTagFromTask(job.TaskGroups[i].Tasks[j])
			}
		}
	}
	return tag, nil
}

// Updates one image in a Nomad job, unless the Nomad jobspec is registrywatcher itself.
// Since the registrywatcher Nomad jobspec contains 2 images (UI and backend), it will update
// both images before it restarts itself.
func (client *NomadClient) UpdateNomadJobTag(jobID, imageName, desiredTag string) {
	_, registryDomain, registryPrefix, _ := utils.ExtractRegistryInfo(client.conf, imageName)
	desiredFullImageName := utils.ConstructImageName(registryDomain, registryPrefix, imageName, desiredTag)
	job, err := client.getNomadJob(jobID)
	if err != nil {
		log.LogAppErr(fmt.Sprintf("Couldn't find jobID %s", jobID), err)
		return
	}
	for i, taskGroup := range job.TaskGroups {
		for j, task := range taskGroup.Tasks {
			fullImageName := getNomadJobImageFromTask(task)
			arr := strings.Split(fullImageName, "/")
			taskImageName := arr[len(arr)-1]
			if taskImageName == imageName {
				// to avoid unexpected issues we use the prefix from config
				// its possible that what is deployed may be from a different registry
				job.TaskGroups[i].Tasks[j].Config["image"] = desiredFullImageName
				// nomad jobs with this config set to false may end up
				// with this config as true in the process of calling this endpoint
				job.TaskGroups[i].Tasks[j].Config["force_pull"] = true
			}
			// bootstrapping. the nomad job "registrywatcher" also contains
			// a task/container of the ui image.
			if taskImageName == "registrywatcher-ui" {
				uiFullImageName := utils.ConstructImageName(
					registryDomain, registryPrefix, "registrywatcher-ui", desiredTag)
				job.TaskGroups[i].Tasks[j].Config["image"] = uiFullImageName
				job.TaskGroups[i].Tasks[j].Config["force_pull"] = true
			}
		}
	}

	if value, ok := os.LookupEnv("VAULT_TOKEN"); ok {
		vaultToken := value
		job.VaultToken = &vaultToken
	}

	go client.RestartNomadJob(&job, desiredTag)
}

// Modify a flag that should not affect the operation of a Nomad jobspec
func (client *NomadClient) flipJobMeta(job *nomad.Job) {
	if _, ok := job.Meta["restart"]; ok {
		if job.Meta["restart"] == "foo" {
			job.Meta["restart"] = "bar"
		} else {
			job.Meta["restart"] = "foo"
		}
	} else {
		job.Meta = map[string]string{
			"restart": "bar",
		}
	}
}

// There is no way to restart a job through the API currently
// https://github.com/hashicorp/nomad/issues/698
func (client *NomadClient) RestartNomadJob(job *nomad.Job, desiredTag string) {
	jobID := *job.ID

	// stupid hack to force a restart when registering a job
	client.flipJobMeta(job)

	utils.PostSlackUpdate(client.conf, fmt.Sprintf("Update: deploying job `%s` to tag `%s`", jobID, desiredTag))
	resp, _, err := client.nc.Jobs().RegisterOpts(job, nil, nil)
	if err != nil {
		log.LogAppErr(fmt.Sprintf("Failed to restart job %s", jobID), err)
		utils.PostSlackError(client.conf, fmt.Sprintf("Error: failed to force redeploy job `%s` for tag `%s`", jobID, desiredTag))
	} else {
		client.MonitorNomadJob(resp.EvalID, jobID, desiredTag)
	}
}

// Monitor the progress of a Nomad job deployment
// and posts a slack update on the outcome
func (client *NomadClient) MonitorNomadJob(evalID, jobID, desiredTag string) {
	evalStatusDesc := ""
	deploymentStatus := "running"
	evalDeploymentID := ""

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	done := make(chan bool)
	go func() {
		// monitor for max 20 mins
		time.Sleep(1200 * time.Second)
		done <- true
	}()
	for {
		select {
		case <-done:
			return
		case _ = <-ticker.C:
			for evalStatusDesc != "complete" {
				eval, _, err := client.nc.Evaluations().Info(evalID, nil)
				if err != nil {
					continue
				}
				evalDeploymentID = eval.DeploymentID
				evalStatusDesc = eval.StatusDescription
				if evalStatusDesc == "" {
					evalStatusDesc = eval.Status
				}
			}

			for deploymentStatus == "running" {
				d, _, err := client.nc.Deployments().Info(evalDeploymentID, nil)
				if err != nil {
					continue
				}
				deploymentStatus = d.Status
			}

			if deploymentStatus == "successful" {
				utils.PostSlackSuccess(client.conf, fmt.Sprintf("Success: Nomad deployment for job `%s` succeeded for tag `%s`", jobID, desiredTag))
			} else if deploymentStatus == "failed" {
				utils.PostSlackError(client.conf, fmt.Sprintf("Error: Nomad deployment job `%s` failed for tag `%s`, nomad server will roll back to last working version if possible", jobID, desiredTag))
			} else {
				utils.PostSlackUpdate(client.conf, fmt.Sprintf("Update: Nomad deployment status is `%s` for job `%s` for tag `%s`. Monitoring timeout", deploymentStatus, jobID, desiredTag))
			}
			done <- true
		}
	}
}
