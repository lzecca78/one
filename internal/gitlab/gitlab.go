package gitlab

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/lzecca78/one/internal/config"
	"github.com/spf13/viper"
	glab "github.com/xanzy/go-gitlab"
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
type GClient struct {
	client *glab.Client
	ctx    context.Context
	config *viper.Viper
}

// BranchSHAComment s a struct that combines Branch with latest sha and related comment
type BranchSHAComment struct {
	BranchName string     `json:"branch"`
	Sha        string     `json:"sha"`
	Comment    string     `json:"comment"`
	CreatedAt  *time.Time `json:"createdAt"`
	Author     string     `json:"author"`
}

type BranchesWithError struct {
	Repo     string
	Branches []BranchSHAComment
	Error    error
}

//NewGitClient is a static function needed for tests
func NewGitlabClient(v *viper.Viper) (*GClient, error) {
	token := config.CheckAndGetString(v, "GITLAB_TOKEN")
	myURL := config.CheckAndGetString(v, "GITLAB_URL")
	ctx := context.Background()
	fmt.Printf("GITLAB_URL: %s\n", myURL)

	client, err := glab.NewClient(token, glab.WithBaseURL(myURL))
	if err != nil {
		return new(GClient), fmt.Errorf("error initializing Gitlab Client: %v", err)

	}
	return &GClient{client, ctx, v}, nil
}

func (g *GClient) GetProjectFromRepoName(repo string) (*glab.Project, error) {
	//set hardcoded pagination because is expected to return at least 1 record
	//the function return for  this reason 1 only Project
	projects, _, err := g.client.Search.Projects(
		fmt.Sprintf("%s", repo),
		&glab.SearchOptions{Page: 1, PerPage: 10},
	)
	if err != nil {
		return nil, fmt.Errorf("error while getting search response: %v", err)
	}

	if len(projects) == 0 {
		return nil, fmt.Errorf("no project found matching %s", repo)
	}

	if len(projects) > 1 {
		return nil, fmt.Errorf("too many project related to the query string %s: %v", repo, err)
	}

	return projects[0], nil

}

func (g *GClient) GetPidFromProject(repo string) (pid interface{}, err error) {
	project, err := g.GetProjectFromRepoName(repo)
	if err != nil {
		return nil, fmt.Errorf("error while gettting project : %v", err)
	}
	return project.PathWithNamespace, nil
}

//ListBranchesByRepo is a function that implements the interface GitClient with a repo param that return a list of branches and an error interface
func (g *GClient) ListBranchesByProject(repo, pattern string) ([]*glab.Branch, error) {
	pid, err := g.GetPidFromProject(repo)
	searchString := fmt.Sprintf("^%s", pattern)
	options := glab.ListBranchesOptions{
		ListOptions: glab.ListOptions{Page: 1, PerPage: 10},
		Search:      &searchString,
	}
	branches, _, err := g.client.Branches.ListBranches(pid, &options)
	if err != nil {
		return nil, fmt.Errorf("error getting branch list: %v", err)
	}
	return branches, nil
}

func (g *GClient) GetCommit(sha, repo string) (*glab.Commit, error) {
	pid, _ := g.GetPidFromProject(repo)
	commit, _, err := g.client.Commits.GetCommit(pid, sha)
	if err != nil {
		return new(glab.Commit), fmt.Errorf("error while getting commit: %v", err)
	}
	return commit, nil
}

func (g *GClient) ListBranchesByRepoWithComment(repo, pattern string) BranchesWithError {
	branches, err := g.ListBranchesByProject(repo, pattern)
	if err != nil {
		return BranchesWithError{repo, nil, err}
	}
	result := []BranchSHAComment{}
	for _, branch := range branches {
		commit, err := g.GetCommit(branch.Commit.ID, repo)
		if err != nil {
			return BranchesWithError{repo, nil, err}
		}
		result = append(result, BranchSHAComment{
			BranchName: branch.Name,
			Sha:        branch.Commit.ID,
			Comment:    commit.Message,
			CreatedAt:  commit.CreatedAt,
			Author:     commit.AuthorName})
	}
	return BranchesWithError{repo, result, nil}
}

type RepoFilterBranch struct {
	repo   string
	filter string
}

func (g *GClient) worker(wg *sync.WaitGroup, done chan bool, branchesChannel chan BranchesWithError, jobs chan RepoFilterBranch, i int) {
	defer wg.Done()
	for bf := range jobs {
		//log.Printf("worker %d received job for repo %s", i, repo)
		select {
		case branchesChannel <- g.ListBranchesByRepoWithComment(bf.repo, bf.filter):
			//log.Printf("worker %d finished job for repo %s", i, repo)
		case <-done:
			//log.Printf("worker %d done", i)
			break
		}
	}
}

//GetRepos is a function that implements the interface GitClient with a RepositoriesProperties param that return a list of of BranchSHAComment and an error interface
func (g *GClient) GetRepos(reposConf []RepoFilterBranch) (map[string][]BranchSHAComment, error) {
	wg := &sync.WaitGroup{}
	// workPoolSize set to +2 w.r.t. number of cpu due to network bounded nature of the task
	workPoolSize := len(reposConf)
	// returned structure
	result := map[string][]BranchSHAComment{}
	// channel used by workers to return results
	branchesChannel := make(chan BranchesWithError, len(reposConf))
	jobs := make(chan RepoFilterBranch, len(reposConf))
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
