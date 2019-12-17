package jenkins

import (
	"testing"
)

func TestNewJenkinsClient(t *testing.T) {
	v, repoProperties := GetConfig()
	NewJenkinsClient(v, repoProperties)
}

func TestConfigureJobs(t *testing.T) {
	repoName := "portal"
	sha := "13456745678945678"
	branch := "brainfuck"
	v, repoProperties := GetConfig()
	jenkinsclient := NewJenkinsClient(v, repoProperties)
	jobsParams := &JobsParameters{
		CommitPerProject: CICommitSpec{
			repoName: Commit{Sha: sha, Branch: branch}},
	}
	jenkinsclient.ConfigureJobs(jobsParams, "ms-fuffa2")
}

func TestGetJobStatus(t *testing.T) {
	v, repoProperties := GetConfig()
	jenkinsclient := NewJenkinsClient(v, repoProperties)
	jenkinsclient.GetJobStatus("ms-2329ab58")
}

func TestCreateFolder(t *testing.T) {
	v, repoProperties := GetConfig()
	jenkinsclient := NewJenkinsClient(v, repoProperties)
	jenkinsclient.createFolder("ms-template", "ms-2329ab58")
}

func TestDeleteFolder(t *testing.T) {
	v, repoProperties := GetConfig()
	jenkinsclient := NewJenkinsClient(v, repoProperties)
	jenkinsclient.DeleteFolder("ms-fuffa2")
}
