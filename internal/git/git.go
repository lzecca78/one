package git

import (
	"context"
	"log"
	"sync"

	"github.com/lzecca78/one/internal/config"
	"github.com/google/go-github/v26/github"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

// CommitSpec is a type def as map of string and Commit
type CommitSpec map[string]Commit

//Commit is a struct needed to define an object `commit` defined with message and sha
type Commit struct {
	Message string `json:"message,omitempty" yaml:"message,omitempty"`
	Branch  string `json:"branch" yaml:"branch"`
	Sha     string `json:"sha" yaml:"sha"`
}

//BranchesWithCommits is a map with list of commits for each branch
type BranchesWithCommits map[string][]Commit

//RepositoriesResponse is a map with name of the repo and BranchesWithCommits
type RepositoriesResponse map[string]BranchesWithCommits

// Client is a struct that describe necessary fields for describing git with client, context and owner
type Client struct {
	client *github.Client
	ctx    context.Context
	owner  string
}

// BranchSHAComment s a struct that combines Branch with latest sha and related comment
type BranchSHAComment struct {
	BranchName string `json:"branch"`
	Sha        string `json:"sha"`
	Comment    string `json:"comment"`
}

//NewGitClient is a static function needed for tests
func NewGitClient(v *viper.Viper) *Client {
	token := config.CheckAndGetString(v, "GITHUB_TOKEN")
	owner := config.CheckAndGetString(v, "GITHUB_OWNER")

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	return &Client{client, ctx, owner}
}

//ListBranchesByRepo is a function that implements the interface GitClient with a repo param that return a list of branches and an error interface
func (g *Client) ListBranchesByRepo(repo string) ([]*github.Branch, error) {
	branches, _, err := g.client.Repositories.ListBranches(g.ctx, g.owner, repo, nil)
	return branches, err
}

//GetCommit is a function that implements the interface GitClient with a sha and repo param that return a github.RepositoryCommit and an error interface
func (g *Client) GetCommit(sha, repo string) (*github.RepositoryCommit, error) {
	commit, _, err := g.client.Repositories.GetCommit(g.ctx, g.owner, repo, sha)
	return commit, err
}

//ListBranchesByRepoWithComment is a function that implements the interface GitClient with a repo param that return a list of of BranchSHAComment and an error interface
func (g *Client) ListBranchesByRepoWithComment(repo string) BranchesWithError {
	branches, _, err := g.client.Repositories.ListBranches(g.ctx, g.owner, repo, nil)
	if err != nil {
		return BranchesWithError{repo, nil, err}
	}
	result := []BranchSHAComment{}
	for _, branch := range branches {
		commit, err := g.GetCommit(*branch.Commit.SHA, repo)
		if err != nil {
			return BranchesWithError{repo, nil, err}
		}
		result = append(result, BranchSHAComment{*branch.Name, *branch.Commit.SHA, *commit.Commit.Message})
	}
	return BranchesWithError{repo, result, nil}
}

type BranchesWithError struct {
	Repo     string
	Branches []BranchSHAComment
	Error    error
}

func (g *Client) worker(wg *sync.WaitGroup, done chan bool, branchesChannel chan BranchesWithError, jobs chan string, i int) {
	defer wg.Done()
	for repo := range jobs {
		//log.Printf("worker %d received job for repo %s", i, repo)
		select {
		case branchesChannel <- g.ListBranchesByRepoWithComment(repo):
			//log.Printf("worker %d finished job for repo %s", i, repo)
		case <-done:
			//log.Printf("worker %d done", i)
			break
		}
	}
}

//GetRepos is a function that implements the interface GitClient with a RepositoriesProperties param that return a list of of BranchSHAComment and an error interface
func (g *Client) GetRepos(reposConf []string) (map[string][]BranchSHAComment, error) {
	wg := &sync.WaitGroup{}
	// workPoolSize set to +2 w.r.t. number of cpu due to network bounded nature of the task
	workPoolSize := len(reposConf)
	// returned structure
	result := map[string][]BranchSHAComment{}
	// channel used by workers to return results
	branchesChannel := make(chan BranchesWithError, len(reposConf))
	jobs := make(chan string, len(reposConf))
	done := make(chan bool, workPoolSize)
	defer close(branchesChannel)
	defer close(done)
	// put repos in the jobs channel
	for _, repo := range reposConf {
		jobs <- repo
	}
	// close jobs channel to communicate workers when jobs can be considered closed
	close(jobs)
	//log.Printf("populated jobs queue with %v", reposConf)
	// spawn workers
	log.Printf("spawning %d workers", workPoolSize)
	for i := 0; i < workPoolSize; i++ {
		go g.worker(wg, done, branchesChannel, jobs, i)
		//log.Printf("spawned worker %d", i)
		// increase wait group counter
		wg.Add(1)
	}
	log.Printf("waiting for %d responses from workers", len(reposConf))
	var err error
	// wait for all jobs to be finished
	for i := 0; i < len(reposConf); i++ {
		//log.Printf("waiting for response %d to be received", i)
		response := <-branchesChannel
		//log.Printf("received response %d/%d", i, len(reposConf))
		// if one worker return an error, communicate done to all workers and exit from loop
		err = response.Error
		if err != nil {
			log.Printf("error getting info for repo %s: %v", response.Repo, err)
			for i := 0; i < workPoolSize; i++ {
				//log.Printf("error: communicating done %d to workers", i)
				done <- true
			}
			break
		}
		result[response.Repo] = response.Branches
	}
	//log.Println("waiting for workers to finish")
	// wait for all workers to exit (gracefully or not)
	wg.Wait()
	//log.Println("communicating done to workers")
	if err == nil {
		for i := 0; i < workPoolSize; i++ {
			//log.Printf("communicating done %d to workers", i)
			done <- true
		}
	}
	return result, err
}
