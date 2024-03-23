package models

type Repository struct {
	RepositoryName string `gorm:"not null;unique;primaryKey" binding:"required"`
	PinnedTag      string `gorm:"not null;unique" binding:"required,email"`
	AutoDeploy     bool   `gorm:"not null" binding:"required"`
	Tags           []Tag  `gorm:"foreignKey:RepositoryID"`
}

func (r Repository) TableName() string {
	return "deployed_repository_version"
}

type Tag struct {
	ID           uint   `gorm:"primaryKey"`
	TagName      string `gorm:"not null" binding:"required"`
	Digest       string `gorm:"not null" binding:"required"`
	RepositoryID string
}
