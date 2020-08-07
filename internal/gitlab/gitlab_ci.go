package gitlab

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	glab "github.com/xanzy/go-gitlab"
)

type JobsParameters struct {
	Stable           bool
	CommitPerProject CommitSpec
}

type gitlabPipelineReq struct {
	Ref     string
	Timeout time.Duration
}

type TriggerConfig struct {
	PipelineToken string
	Identifier    int
}

type Trigger struct {
	GClient   *GClient
	Conf      TriggerConfig
	ProjectID *interface{}
}

type PipelineSpec struct {
	Pipeline *glab.Pipeline
	gitlabPipelineReq
	Trigger
}

func (g *GClient) NewTrigger(repo string) *Trigger {

	pid, err := g.GetPidFromProject(repo)
	if err != nil {
		log.Fatal(err)
	}
	trigger, _ := g.manageTrigger(&pid)
	log.Printf("trigger is %+v", trigger)

	if err != nil {
		log.Fatalf("time conversion error: %v", err)
	}

	return &Trigger{
		GClient:   g,
		ProjectID: &pid,
		Conf: TriggerConfig{
			PipelineToken: trigger.Token,
			Identifier:    trigger.ID,
		},
	}
}

func (g *GClient) manageTrigger(pid *interface{}) (finalTrigger *glab.PipelineTrigger, err error) {
	ok, err := g.TriggerAlreadyExists(pid)
	if err != nil {
		return new(glab.PipelineTrigger), err
	}
	if !ok {
		log.Println("no trigger found, creating new one")
		finalTrigger, err = g.CreateTriggerPipeline(pid)
		if err != nil {
			return new(glab.PipelineTrigger), err
		}
		return finalTrigger, nil
	}
	log.Println("trigger found! using existing one")
	finalTrigger, err = g.GetExistingTrigger(pid)
	if err != nil {
		return new(glab.PipelineTrigger), err
	}
	return finalTrigger, nil
}

func (g *GClient) CreateTriggerPipeline(pid *interface{}) (*glab.PipelineTrigger, error) {
	triggerOptions := glab.AddPipelineTriggerOptions{
		Description: glab.String(strings.Replace((*pid).(string), "/", "-", -1)),
	}

	trigger, resp, err := g.client.PipelineTriggers.AddPipelineTrigger(*pid, &triggerOptions)
	if err != nil {
		return new(glab.PipelineTrigger), fmt.Errorf("error while create a trigger: %v, response is %v", err, resp)
	}
	return trigger, nil
}

func (g *GClient) TriggerAlreadyExists(pid *interface{}) (bool, error) {
	opts := glab.ListPipelineTriggersOptions{
		Page:    1,
		PerPage: 10,
	}
	triggers, _, err := g.client.PipelineTriggers.ListPipelineTriggers(*pid, &opts)
	if err != nil {
		log.Printf("error while fetching triggers : %v", err)
		return false, err
	}
	if len(triggers) == 0 {
		return false, nil
	}
	return true, nil
}

func (g *GClient) GetExistingTrigger(pid *interface{}) (*glab.PipelineTrigger, error) {
	opts := glab.ListPipelineTriggersOptions{
		Page:    1,
		PerPage: 10,
	}
	triggers, _, err := g.client.PipelineTriggers.ListPipelineTriggers(*pid, &opts)

	if err != nil {
		return new(glab.PipelineTrigger), err

	}

	return triggers[0], nil
}

func (t *Trigger) runPipeline(repo string, plineReq *gitlabPipelineReq) (*PipelineSpec, error) {

	options := glab.RunPipelineTriggerOptions{
		Ref:       glab.String(plineReq.Ref),
		Variables: map[string]string{"TEST": "PROVA"},
		Token:     glab.String(t.Conf.PipelineToken),
	}

	pid, err := t.GClient.GetPidFromProject(repo)
	if err != nil {
		log.Fatal(err)
	}

	resp, _, err := t.GClient.client.PipelineTriggers.RunPipelineTrigger(pid, &options)
	if err != nil {
		return new(PipelineSpec), fmt.Errorf("%v", err)
	}

	return &PipelineSpec{Pipeline: resp, gitlabPipelineReq: *plineReq, Trigger: *t}, nil
}

func (t *Trigger) NewPipeline(repo string, plineReq *gitlabPipelineReq) *PipelineSpec {
	pipeline, err := t.runPipeline(repo, plineReq)
	if err != nil {
		log.Fatal(err)
	}
	return pipeline
}

func (p *PipelineSpec) waitForPipelineToFinish() error {
	timeout := time.After(p.Timeout)
	tick := time.Tick(2 * time.Second)
	for {
		select {
		case <-timeout:
			return errors.New("timed out")
		case <-tick:
			finished, err := p.isPipelineFinished()
			if err != nil {
				return fmt.Errorf("failed to check if pipline is finished. %w", err)
			} else if finished {
				fmt.Println()
				return nil
			} else {
				fmt.Print(".")
			}
		}
	}
}

func (p *PipelineSpec) isPipelineFinished() (bool, error) {
	pl, _, err := p.Trigger.GClient.client.Pipelines.GetPipeline(p.Trigger.ProjectID, p.Pipeline.ID)
	if err != nil {
		return false, fmt.Errorf("failed to get pipeline info. %w", err)
	}
	finishedStatuses := []string{"failed", "manual", "canceled", "success", "skipped"}
	for _, status := range finishedStatuses {
		if pl.Status == status {
			return true, nil
		}
	}
	return false, nil
}

func (p *PipelineSpec) isPipelineFailed() (bool, error) {
	pl, _, err := p.Trigger.GClient.client.Pipelines.GetPipeline(p.Trigger.ProjectID, p.Pipeline.ID)
	if err != nil {
		return false, fmt.Errorf("failed to get pipeline info. %w", err)
	}
	successStatuses := []string{"manual", "success"}
	for _, status := range successStatuses {
		if pl.Status == status {
			return false, nil
		}
	}
	return true, nil
}

func (g *GClient) CheckIfPipelineisUpToDateWithlastCommitSha(commit *Commit, pline *glab.Pipeline) bool {

	if commit.Sha != pline.SHA {
		return false
	}

	return true
}
