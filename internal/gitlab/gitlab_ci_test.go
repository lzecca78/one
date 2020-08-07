package gitlab

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
)

func NewTrigger() *Trigger {
	client := getTestGitlabClient()
	os.Setenv("ONE_GITLAB_TOKEN", "nJRj3NNqxLpHaFXHu8ok")
	os.Setenv("ONE_GITLAB_URL", "https://git.sighup.io/api/v4")
	trigger := client.NewTrigger("lzecca/test-go")
	return trigger
}

func TestProjectFromRepoNameFacile(t *testing.T) {

	// os.Setenv("ONE_GITLAB_TOKEN", "zZ34ssiYGgM2Uc2e45ti")
	// os.Setenv("ONE_GITLAB_URL", "https://gitlab.facile.it/api/v4")

	os.Setenv("ONE_GITLAB_TOKEN", "nJRj3NNqxLpHaFXHu8ok")
	os.Setenv("ONE_GITLAB_URL", "https://git.sighup.io/api/v4")
	client := getTestGitlabClient()
	repo := "lzecca/test-go"
	project, err := client.GetProjectFromRepoName(repo)
	if err != nil {
		t.Fatal(err)
	}
	log.Printf("project is %v", project.NameWithNamespace)
}

func TestRunPipeline(t *testing.T) {
	// os.Setenv("ONE_GITLAB_TOKEN", "zZ34ssiYGgM2Uc2e45ti")
	// os.Setenv("ONE_GITLAB_URL", "https://gitlab.facile.it/api/v4")
	os.Setenv("ONE_GITLAB_TOKEN", "bbQTqsx8hB52J5QtY9FQ")
	//os.Setenv("ONE_GITLAB_TOKEN", "nJRj3NNqxLpHaFXHu8ok")
	os.Setenv("ONE_GITLAB_URL", "https://git.sighup.io/api/v4")
	os.Setenv("ONE_GITLAB_PIPELINE_TIMEOUT", "30m")
	repo := "lzecca/test-go"
	trigger := NewTrigger()

	timeString := "30m"

	timeout, err := time.ParseDuration(timeString)
	if err != nil {
		log.Fatalf("time conversion error: %v", err)
	}

	pipeline := gitlabPipelineReq{
		Ref:     "master",
		Timeout: timeout,
	}
	pline := trigger.NewPipeline(repo, &pipeline)
	spew.Dump(pline)

}
