// +build integration

package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// test ShouldDeploy for a multitude of conditions
func TestAutoDeployLatestTag(t *testing.T) {
	te := SetUpClientTest(t)
	defer te.TearDown()

	// populate with new tags
	tags := []string{"test", "v0.1.0"}
	for _, tag := range tags {
		te.PushNewTag(tag, "latest")
	}

	// update the cache with digests
	te.Clients.PopulateCaches(te.TestRepoName)

	/*
		resolves correctly for autodeployment of latest
	*/

	// push the new tag
	newTag := "v0.0.2"
	te.PushNewTag(newTag, "latest")

	shouldDeploy, _ := te.Clients.ShouldDeploy(te.TestRepoName)
	te.Clients.UpdateCaches(te.TestRepoName)
	tagToDeploy, _ := te.Clients.GetFormattedPinnedTag(te.TestRepoName)
	// v0.0.2 is not a new release version
	assert.False(t, shouldDeploy)
	assert.Equal(t, tagToDeploy, "v0.1.0")

	// push the new tag
	newTag = "v0.2.0"
	te.PushNewTag(newTag, "latest")

	shouldDeploy, _ = te.Clients.ShouldDeploy(te.TestRepoName)
	te.Clients.UpdateCaches(te.TestRepoName)
	tagToDeploy, _ = te.Clients.GetFormattedPinnedTag(te.TestRepoName)
	// v0.2.0 is a new release version,
	assert.True(t, shouldDeploy)
	assert.Equal(t, tagToDeploy, newTag)

	// push the new tag
	newTag = "v0.0.9"
	te.PushNewTag(newTag, "latest")

	shouldDeploy, _ = te.Clients.ShouldDeploy(te.TestRepoName)
	te.Clients.UpdateCaches(te.TestRepoName)
	tagToDeploy, _ = te.Clients.GetFormattedPinnedTag(te.TestRepoName)
	// v0.0.9 is not a new release version
	assert.False(t, shouldDeploy)
	assert.Equal(t, tagToDeploy, "v0.2.0")

	/*
		resolves correctly for autodeployment of custom tags
	*/

	// set to "test:latest"
	newTag = "test"
	te.UpdatePinnedTag(newTag)

	shouldDeploy, _ = te.Clients.ShouldDeploy(te.TestRepoName)
	te.Clients.UpdateCaches(te.TestRepoName)
	tagToDeploy, _ = te.Clients.GetFormattedPinnedTag(te.TestRepoName)
	assert.False(t, shouldDeploy)
	assert.Equal(t, "test", tagToDeploy)

	// "test" is now based on "alpine", rather than the original "latest"
	te.PushNewTag(newTag, "alpine")

	shouldDeploy, _ = te.Clients.ShouldDeploy(te.TestRepoName)
	te.Clients.UpdateCaches(te.TestRepoName)
	tagToDeploy, _ = te.Clients.GetFormattedPinnedTag(te.TestRepoName)
	assert.True(t, shouldDeploy)
	assert.Equal(t, "test", tagToDeploy)

	// "test" is back to "latest", but autoDeploy is off
	_ = te.Clients.PostgresClient.UpdateAutoDeployFlag(te.TestRepoName, false)
	te.PushNewTag(newTag, "alpine")

	shouldDeploy, _ = te.Clients.ShouldDeploy(te.TestRepoName)
	autoDeploy, _ := te.Clients.PostgresClient.GetAutoDeployFlag(te.TestRepoName)
	assert.False(t, autoDeploy)
	assert.False(t, shouldDeploy)
	assert.Equal(t, "test", tagToDeploy)
}
