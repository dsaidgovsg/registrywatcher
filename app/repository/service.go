package repository

import (
	"net/http"
	"registrywatcher/app/models"
	"registrywatcher/utilities/config"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RepositoryService struct {
	DB     *gorm.DB
	Config *config.Config
	// Sender external.Sender
}

// GetRepositories is a handler that returns a list of repositories
func (m RepositoryService) GetRepositories(c *gin.Context) {

	var repositories []models.Repository
	if err := m.DB.Preload("Tags").Find(&repositories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := map[string]RepositoryData{}

	for _, repo := range repositories {
		tags := []string{}
		for _, tag := range repo.Tags {
			tags = append(tags, tag.TagName)
		}

		response[repo.RepositoryName] = RepositoryData{
			AutoDeploy:     repo.AutoDeploy,
			PinnedTag:      repo.PinnedTag,
			PinnedTagValue: repo.PinnedTag,
			Tags:           tags,
		}
	}
	c.JSON(http.StatusOK, response)
}
