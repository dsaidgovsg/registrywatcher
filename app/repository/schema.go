package repository

type GetReposiroriesRequest struct {
	RecipientID uint `form:"recipient_id" binding:"required"`
}

type GetRepositoriesResponse struct {
	RepositoryMap map[string]RepositoryData
}

type RepositoryData struct {
	AutoDeploy     bool     `json:"auto_deploy"`
	PinnedTag      string   `json:"pinned_tag"`
	PinnedTagValue string   `json:"pinned_tag_value"`
	Tags           []string `json:"tags"`
}