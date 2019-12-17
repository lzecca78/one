package git

import (
	"encoding/json"
	"log"
	"testing"

	"github.com/lzecca78/one/internal/config"
	"github.com/lzecca78/one/internal/jenkins"
)

//const token = "token"
//const owner = "owner"

func getTestGitClient() (*GitClient, *config.RepositoriesProperties) {
	v, reposConfig := config.GetConfig()
	return NewGitClient(v), reposConfig
}

func TestNewClient(t *testing.T) {
	getTestGitClient()
}

func TestPortalListBranches(t *testing.T) {
	client, _ := getTestGitClient()

	commits, err := client.ListBranchesByRepo("portal")
	if err != nil || len(commits) == 0 {
		t.Error("no commit found in portal: ", err)
	}
	sha := *commits[0].Commit.SHA
	commit, err := client.GetCommit(sha, "portal")
	if err != nil {
		t.Error("error getting commit ", sha, "in repo portal", err)
	}
	log.Println(*commit.Commit.Message)

}

func TestPortalListBranchesWithComment(t *testing.T) {
	client, _ := getTestGitClient()

	client.ListBranchesByRepoWithComment("portal")

}

func TestGetRepos(t *testing.T) {
	client, _ := getTestGitClient()

	reposProp := &config.RepositoriesProperties{Conf: map[string]jenkins.JenkinsJobConfig{"portal": {"asasasas", "343243423"}}}
	repos, err := client.GetRepos(reposProp)
	if err != nil {
		t.Error(err)
	}
	//log.Println(repos)
	jsonRepos, _ := json.MarshalIndent(repos, "", " ")
	log.Println(string(jsonRepos))
}
