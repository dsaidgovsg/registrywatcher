package app_tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"registrywatcher/app/models"
	testutils "registrywatcher/tests"

	"github.com/stretchr/testify/suite"
)

type RepositoriesTestSuite struct {
	testutils.TestSuite
}

func TestRunMessagesTestSuite(t *testing.T) {
	suite.Run(t, &RepositoriesTestSuite{
		TestSuite: testutils.TestSuite{
			Data: &testutils.DataSeed{
				Repos: []models.Repository{
					{
						RepositoryName: "repo_1",
						PinnedTag:      "repo_1_tag_1",
						AutoDeploy:     true,
					},
					{
						RepositoryName: "repo_2",
						PinnedTag:      "repo_2_tag_1",
						AutoDeploy:     true,
					},
				},
				Tags: []models.Tag{
					{
						TagName:      "repo_1_tag_1",
						Digest:       "sha256:1234",
						RepositoryID: "repo_1",
					},
					{
						TagName:      "repo_2_tag_1",
						Digest:       "sha256:5678",
						RepositoryID: "repo_2",
					},
					{
						TagName:      "repo_1_tag_2",
						Digest:       "sha256:2345",
						RepositoryID: "repo_1",
					},
					{
						TagName:      "repo_2_tag_2",
						Digest:       "sha256:6789",
						RepositoryID: "repo_2",
					},
				},
			},
		},
	})
}

func (suite *RepositoriesTestSuite) TestRepositoryService_GetRepositories() {
	// Define response structure
	type repo struct {
		AutoDeploy     bool     `json:"auto_deploy"`
		PinnedTag      string   `json:"pinned_tag"`
		PinnedTagValue string   `json:"pinned_tag_value"`
		Tags           []string `json:"tags"`
	}

	tests := []struct {
		name     string
		response map[string]repo
	}{
		{
			name: "GetRepositories",
			response: map[string]repo{
				"repo_1": {
					AutoDeploy:     true,
					PinnedTag:      "repo_1_tag_1",
					PinnedTagValue: "repo_1_tag_1",
					Tags:           []string{"repo_1_tag_1", "repo_1_tag_2"},
				},
				"repo_2": {
					AutoDeploy:     true,
					PinnedTag:      "repo_2_tag_1",
					PinnedTagValue: "repo_2_tag_1",
					Tags:           []string{"repo_2_tag_1", "repo_2_tag_2"},
				},
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Send request
			resp := suite.SendRequest(http.MethodGet, "/repos", nil, nil)
			suite.Equal(http.StatusOK, resp.Code)

			// Deserialize response
			deserializedResp := map[string]repo{}
			fmt.Println(resp.Body.String())
			err := json.Unmarshal(resp.Body.Bytes(), &deserializedResp)
			suite.NoError(err)

			// Compare actual and expected response
			suite.Equal(deserializedResp, tt.response)
		})
	}
}
