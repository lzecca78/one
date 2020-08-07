package gitlab

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/lzecca78/one/internal/config"
	"github.com/spf13/viper"
)

func getTestGitlabClient(options ...map[string]string) *GClient {
	v := config.GetConfig()
	for _, i := range options {
		for k, v := range i {
			viper.Set(k, v)
		}
	}
	client, _ := NewGitlabClient(v)
	return client
}

func TestNewClient(t *testing.T) {
	getTestGitlabClient()
}

func TestProjectFromRepoName(t *testing.T) {
	os.Setenv("ONE_GITLAB_TOKEN", "nJRj3NNqxLpHaFXHu8ok")
	os.Setenv("ONE_GITLAB_URL", "https://git.sighup.io/api/v4")
	client := getTestGitlabClient()
	repo := "sisal-op/elot/infra"
	project, err := client.GetProjectFromRepoName(repo)
	if err != nil {
		t.Errorf("error while fetching project from %s: %v", repo, err)
	}
	log.Printf("project is %v", project.NameWithNamespace)
}

func TestSisalRepoListBranches(t *testing.T) {
	client := getTestGitlabClient()
	repo := "sisal-op/pgt-tr/infra"
	filter := "dynatrace"
	branchesWithError := client.ListBranchesByRepoWithComment(repo, filter)
	if len(branchesWithError.Branches) == 0 {
		t.Error("no branches found in: ", repo)
	}
	sha := branchesWithError.Branches[0].Sha
	commit, err := client.GetCommit(sha, repo)
	if err != nil {
		t.Error("error getting commit ", sha, "in repo portal", err)
	}
	log.Println(commit.Message)

}

func TestPortalListBranchesWithComment(t *testing.T) {
	client := getTestGitlabClient()

	repo := "sisal-op/pgt-tr/infra"
	filter := "dynatrace"
	s := client.ListBranchesByRepoWithComment(repo, filter)
	fmt.Printf("branches are %+v", s.Branches)

}

func TestGetRepos(t *testing.T) {
	client := getTestGitlabClient()
	p := []RepoFilterBranch{
		{
			repo:   "sisal-op/pgt-tr/infra",
			filter: "dynatrace",
		},
		{
			repo:   "casavo/infra",
			filter: "iam-group",
		},
		{
			repo:   "theoutplay/infra",
			filter: "new-prometheus",
		},
		{
			repo:   "easywelfare/infra",
			filter: "update_monit",
		},
		{
			repo:   "unicredit/corp-pipeline",
			filter: "notary",
		},
	}
	s, err := client.GetRepos(p)
	if err != nil {
		t.Errorf("error while fetching function: %v \n", err)
	}
	log.Printf("response is %+v", s)
	spew.Dump(s)
}
