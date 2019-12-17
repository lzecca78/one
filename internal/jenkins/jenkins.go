package jenkins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/lzecca78/one/internal/config"
	"github.com/lzecca78/one/internal/git"
	"github.com/bndr/gojenkins"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/jbowtie/gokogiri"
	"github.com/jbowtie/gokogiri/xml"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	verbPost          = "POST"
	verbGet           = "GET"
	triggersXpath     = "/flow-definition/properties/org.jenkinsci.plugins.workflow.job.properties.PipelineTriggersJobProperty/triggers"
	envsXpath         = "/flow-definition/properties/hudson.model.ParametersDefinitionProperty/parameterDefinitions/hudson.model.StringParameterDefinition"
	cvsXpath          = "/flow-definition/definition/scm/branches/hudson.plugins.git.BranchSpec"
	k8sEnvAttr        = "K8S_NAMESPACE"
	gitBranchEnvAttr  = "GIT_BRANCH"
	nameXpath         = "name"
	defaultValueXpath = "defaultValue"
)

//JenkinsClient struct with config inherited from JenkinsClientConfig and native client of gojenkins
type JenkinsClient struct {
	Config *JenkinsClientConfig
	Client *gojenkins.Jenkins
}

//JenkinsJobConfig is a struct that describe the needed element for implementing the jenkins api calls
type JenkinsJobConfig struct {
	JenkinsJob   string `json:"jenkinsJob,omitempty" yaml:"jenkinsJob,omitempty" mapstructure:"jenkinsJob,omitempty"`
	JenkinsToken string `json:"jenkinsToken,omitempty" yaml:"jenkinsToken,omitempty" mapstructure:"jenkinsToken,omitempty"`
}

//JobsParameters is the struct needed by a continuous integration service to configure parametrized jobs
type JobsParameters struct {
	Stable           bool
	CommitPerProject git.CommitSpec
}

type JobsStatuses map[string]string

type ContinousIntegration interface {
	User() string
	Password() string
	Url() string
	JobsWithParameters() *JobsParameters
}

type JenkinsItem struct {
	Folder bool
	Job    bool
}

//RepositoriesProperties is needed for the output of the api `/api/repos` in the format:
//{
// "portal": [
//  {
//   "branch": "REF-1415-put-me-beneficiaries",
//   "sha": "89b49d796e4028e078d0b743bbcc0e6f4641334d",
//   "comment": "Add fake response from beneficiaryService. Testing"
//  },
//  {
//   "branch": "add-validity-date-to-employee-order-api",
//   "sha": "2dd8e54b12e386b3e4649e9545b125b05ff70393",
//   "comment": "Present also employeeOrderValidityDate in /employee-orders/{id} api"
//  }
// ]
//}
//RepositoriesProperties is a struct of an hash composed by string and JenkinsConfig struct inherited
type RepositoriesProperties struct {
	Conf map[string]JenkinsJobConfig `mapstructure:"conf"`
}

//JenkinsClientConfig struct that describe
type JenkinsClientConfig struct {
	URI                    string `json:"jenkins_uri" yaml:"jenkins_uri"`
	Username               string `json:"jenkins_username" yaml:"jenkins_username"`
	Password               string `json:"jenkins_password" yaml:"jenkins_password"`
	FolderTemplate         string `json:"folder_template" yaml:"folder_template"`
	RepositoriesProperties `json:"repositoriesProperties" yaml:"repositoriesProperties"`
	repos                  []string
}

//NewJenkinsClient initialize the client jenkins
func NewJenkinsClient(v *viper.Viper) *JenkinsClient {

	repositoriesProperties := RepositoriesProperties{}
	err := v.Unmarshal(&repositoriesProperties)
	if err != nil {
		log.Fatalf("unable to unmarshal repoProperties: %v", err)
	}

	uri := config.CheckAndGetString(v, "JENKINS_URI")
	username := config.CheckAndGetString(v, "JENKINS_USERNAME")
	password := config.CheckAndGetString(v, "JENKINS_PASSWORD")
	folderTemplate := config.CheckAndGetString(v, "JENKINS_FOLDER_TEMPLATE")

	jenkins, err := gojenkins.CreateJenkins(nil, uri, username, password).Init()
	if err != nil {
		log.Fatal("jenkins client failed with ", err)
	}
	repos := []string{}
	for repo := range repositoriesProperties.Conf {
		repos = append(repos, repo)
	}

	jenkinsConfig := &JenkinsClientConfig{URI: uri, Username: username, Password: password, FolderTemplate: folderTemplate, RepositoriesProperties: repositoriesProperties, repos: repos}
	return &JenkinsClient{jenkinsConfig, jenkins}
}

func (c *JenkinsClient) GetRepos() []string {
	return c.Config.repos
}

func (c *JenkinsClient) GetJobStatus(namespace string) (JobsStatuses, error) {
	statuses := JobsStatuses{}
	repoProp := c.Config.RepositoriesProperties

	for job := range repoProp.Conf {
		newJobName := GetNewJobName(job, namespace)
		folderSubPath := fmt.Sprintf("job/%s/job", namespace)
		getURIPath := filepath.Join(folderSubPath, newJobName, "lastBuild", "api", "json")
		getResponse, err := c.httpJenkinsClient(newJobName, getURIPath, verbGet, nil, nil)
		log.Printf("the response body for job %s is %v", job, getResponse)
		if err != nil {
			log.Printf("error while getting response : %s", err)
			return nil, err
		}
		var jsonResp map[string]interface{}
		respRead, err := ioutil.ReadAll(getResponse.Body)
		if err != nil {
			log.Printf("error while reading response :%s", err)
			return nil, err
		}
		err = json.Unmarshal(respRead, &jsonResp)
		if err != nil {
			log.Printf("error while unmarshaling json :%v", err)
		}
		for k, v := range jsonResp {
			if k == "result" {
				var status string
				if v == nil {
					status = "RUNNING"
				} else {
					status = v.(string)
				}
				statuses[newJobName] = status
			}
		}
	}
	log.Printf("statuses is %v", statuses)
	return statuses, nil
}

func (c *JenkinsClient) createFolder(cloneFolder, namespace string, jobSpec *JobsParameters) (string, error) {
	folder := namespace
	jItem := JenkinsItem{
		Folder: true,
		Job:    false,
	}
	bytesXML, err := c.getItemFromJenkins(namespace, cloneFolder, jobSpec, jItem)
	if err != nil {
		log.Printf("error while parsing xml: %v", err)
		return "", errors.Errorf("error while parsing xml: %v", err)
	}
	postContextPath := filepath.Join("createItem")
	postResponse, err := c.httpJenkinsClient(folder, postContextPath, verbPost, bytes.NewBuffer(bytesXML), map[string]string{"name": folder})
	if postResponse.StatusCode != 200 {
		r, _ := ioutil.ReadAll(postResponse.Body)
		log.Printf("the post of new job was not good : %v, %v", postResponse.StatusCode, string(r))
		return "", errors.Errorf("the response was not ok! : %v", postResponse.StatusCode)
	}
	if err != nil {
		log.Printf("error in post:%s", err)
		return "", err
	}
	return folder, nil
}

func (c *JenkinsClient) ReplayJob(job, repo, branch, namespace string) error {
	log.Printf("entering in the function ReplayJob")
	jobToken := c.Config.RepositoriesProperties.Conf[repo].JenkinsToken
	log.Printf("executed JenkinsToken")
	newJobName := GetNewJobName(job, namespace)
	log.Printf("newJobName is %s", newJobName)
	err := c.executeJob(repo, namespace, jobToken, newJobName, branch)
	log.Printf("executeJob is executed")
	if err != nil {
		log.Printf("error executing job : %s for repo : %s with ", namespace, err)
		return err
	}
	return nil
}

func (c *JenkinsClient) createJob(job, namespace string, jobSpec *JobsParameters) (string, error) {
	jItem := JenkinsItem{
		Folder: false,
		Job:    true,
	}
	bytesXML, err := c.getItemFromJenkins(namespace, job, jobSpec, jItem)
	if err != nil {
		log.Printf("error while parsing xml: %v", err)
		return "", errors.Errorf("error while parsing xml: %v", err)
	}
	newNameJob := GetNewJobName(job, namespace)
	folderSubPath := fmt.Sprintf("job/%s", namespace)
	postContextPath := filepath.Join(folderSubPath, "createItem")
	postResponse, err := c.httpJenkinsClient(job, postContextPath, verbPost, bytes.NewBuffer(bytesXML), map[string]string{"name": newNameJob})
	if postResponse.StatusCode != 200 {
		r, _ := ioutil.ReadAll(postResponse.Body)
		log.Printf("the post of new job was not good : %v, %v", postResponse.StatusCode, string(r))
		return "", errors.Errorf("the response was not ok! : %v", postResponse.StatusCode)
	}
	if err != nil {
		log.Printf("error in post:%s", err)
		return "", err
	}
	return newNameJob, nil
}

func (c *JenkinsClient) getItemFromJenkins(namespace, projectScope string, jobSpec *JobsParameters, jitem JenkinsItem) ([]byte, error) {
	getURIPath := filepath.Join("job", projectScope, "config.xml")
	getResponse, err := c.httpJenkinsClient(projectScope, getURIPath, verbGet, nil, nil)
	if err != nil {
		log.Printf("error while getting response : %s", err)
		return nil, err
	}
	if getResponse.StatusCode != 200 {
		log.Printf("response status conewJobName is not 200 : %v", getResponse.StatusCode)
		return nil, errors.Errorf("the response was not ok! : %v", getResponse.StatusCode)
	}
	return parseXMLBody(namespace, projectScope, getResponse, jobSpec, jitem)
}

func parseXMLBody(namespace, project string, response *http.Response, jobSpec *JobsParameters, item JenkinsItem) ([]byte, error) {
	var resp []byte
	xmlResponse, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	if err != nil {
		log.Printf("error reading get response body: %v ", err)
		return resp, err
	}
	if len(xmlResponse) <= 0 {
		log.Print("resposnse body was empty")
		return resp, errors.Errorf("response body was empty")
	}
	parsedXML, err := gokogiri.ParseXml(xmlResponse)
	defer parsedXML.Free()
	if err != nil {
		log.Printf("error parsing response as xml: %v", err)
		return resp, err
	}
	if (!jobSpec.Stable) && (item.Job) {
		k8sEnvCurrentValue := namespace
		gitBranchCurrentValue := jobSpec.CommitPerProject[project].Branch
		envs, err := parsedXML.Root().Search(envsXpath)
		if err != nil {
			log.Printf("error searching for triggers in %s: %v", triggersXpath, err)
			return resp, err
		}
		for _, env := range envs {
			err := changeDefaultValueForSpecificName(env, k8sEnvAttr, k8sEnvCurrentValue, nameXpath, defaultValueXpath)
			if err != nil {
				log.Printf("error while changing value %s for key %s: %v", k8sEnvCurrentValue, k8sEnvAttr, err)
			}
			err = changeDefaultValueForSpecificName(env, gitBranchEnvAttr, gitBranchCurrentValue, nameXpath, defaultValueXpath)
			if err != nil {
				log.Printf("error while changing value %s for key %s: %v", gitBranchCurrentValue, gitBranchEnvAttr, err)
			}
		}
		cvsBlocks, err := parsedXML.Root().Search(cvsXpath)
		if err != nil {
			log.Printf("error searching for envsBlock in %s: %v", envsXpath, err)
		}
		for _, cvs := range cvsBlocks {
			err := replaceBranchSpec(cvs, "name", gitBranchCurrentValue, nameXpath)
			if err != nil {
				log.Printf("error while changing value for name %s: %v", gitBranchCurrentValue, err)
			}
		}
	}

	if jobSpec.Stable {
		triggers, err := parsedXML.Root().Search(triggersXpath)
		if err != nil {
			log.Printf("error searching for triggers in %s: %v", triggersXpath, err)
			return resp, err
		}
		for _, trigger := range triggers {
			trigger.Remove()
		}
	}
	bytesXML := []byte(fmt.Sprintf("%v", parsedXML))
	return bytesXML, nil
}

//GetNewJobName return the name of the job with the namespace prefix
func GetNewJobName(job, namespace string) string {
	return fmt.Sprintf("%s-%s", namespace, job)
}

//ConfigureJobs is a function that implements JenkinsClient, takes JobsParameters and return ....
func (c *JenkinsClient) ConfigureJobs(j *JobsParameters, namespace string) (map[string]string, error) {
	response := map[string]string{}
	folderTemplateName := c.Config.FolderTemplate
	_, err := c.createFolder(folderTemplateName, namespace, j)
	if err != nil {
		log.Printf("error creating folder with name %s: %v", namespace, err)
		return nil, err
	}
	for repo, commit := range j.CommitPerProject {
		jobName := c.Config.RepositoriesProperties.Conf[repo].JenkinsJob
		jobToken := c.Config.RepositoriesProperties.Conf[repo].JenkinsToken
		log.Println("jobName: ", jobName, "jobToken: ", jobToken, "commit: ", commit)
		newJobName, err := c.createJob(jobName, namespace, j)
		response[repo] = newJobName
		if err != nil {
			log.Printf("error creating job %s from %s : %v", newJobName, jobName, err)
			return nil, err
		}
		response[repo] = newJobName
		log.Println("jobName: ", jobName, "jobToken: ", jobToken, "commit: ", commit)
		branch := commit.Branch
		err = c.executeJob(repo, namespace, jobToken, newJobName, branch)
		if err != nil {
			log.Printf("error executing job : %s for repo : %s with ", namespace, err)
		}

	}
	return response, nil
}

func (c *JenkinsClient) executeJob(repo, namespace, token, job, branch string) error {
	parameters := map[string]string{
		"GIT_BRANCH":    "refs/heads/" + branch,
		"K8S_NAMESPACE": namespace,
		"token":         token,
		"cause":         "build by one, deploying to ns: " + namespace,
	}
	err := c.jenkinsRequest(parameters, job, namespace)
	if err != nil {
		return err
	}
	return nil
}

//DeleteFolder is a function that takes folderName as parameter and delete the specified jenkins folder with jobs inside
func (c *JenkinsClient) DeleteFolder(folderName string) error {
	log.Printf("deleting job %v", folderName)
	uriPath := filepath.Join("job", folderName, "doDelete")
	response, err := c.httpJenkinsClient(folderName, uriPath, verbPost, nil, nil)
	log.Println(response)
	return err
}

//jenkinsRequest is a wrapper for an httpClient that send a post to Jenkins with the job params in the payload
func (c *JenkinsClient) jenkinsRequest(parameters map[string]string, jobName, namespace string) error {
	folderSubPath := fmt.Sprintf("job/%s/job", namespace)
	uriPath := filepath.Join(folderSubPath, jobName, "buildWithParameters")
	response, err := c.httpJenkinsClient(jobName, uriPath, verbPost, nil, parameters)
	if err != nil {
		log.Printf("there was en error in the response for uri %s, verb %s: %v", uriPath, verbPost, err)
	}
	log.Println(response)
	return err
}

func (c *JenkinsClient) httpJenkinsClient(jobName, uri, verb string, payload io.Reader, qs map[string]string) (*http.Response, error) {
	rootURI, err := url.Parse(c.Config.URI)
	if err != nil {
		log.Printf("error while parsing url : %v", err)
		return &http.Response{}, err
	}
	rootURI.Path = uri
	if qs != nil {
		v := url.Values{}
		for key, value := range qs {
			v.Add(key, value)
		}
		rootURI.RawQuery = v.Encode()
	}
	log.Println(rootURI)
	retryClient := retryablehttp.NewClient()
	retryClient.CheckRetry = func(_ context.Context, resp *http.Response, err error) (bool, error) {
		if resp == nil || resp.StatusCode == 401 {
			return true, err
		}
		return retryablehttp.DefaultRetryPolicy(context.TODO(), resp, err)
	}
	//request, _ := http.NewRequest(verb, rootURI.String(), payload)
	request, _ := retryablehttp.NewRequest(verb, rootURI.String(), payload)
	request.SetBasicAuth(c.Config.Username, c.Config.Password)
	request.Header.Add("Content-Type", "text/xml")
	response, err := retryClient.Do(request)
	if err != nil {
		log.Printf("error while sending http %s : %v", verb, err)
		return &http.Response{}, err
	}
	return response, nil
}

func changeDefaultValueForSpecificName(xnode xml.Node, name, value, xpathName, xpathValue string) error {
	xname, err := xnode.Search(xpathName)
	if err != nil {
		log.Printf("error searching for env in %s: %v", xpathName, err)
		return err
	}
	if len(xname) == 1 {
		defValue, err := xnode.Search(xpathValue)
		if err != nil {
			log.Printf("error searching for defValue in %s: %v", xpathValue, err)
			return err
		}
		if (len(defValue) == 1) && (xname[0].Content() == name) {
			newDefaultValue := fmt.Sprintf("<defaultValue>%s</defaultValue>", value)
			defValue[0].Replace(newDefaultValue)
		}
	}
	return nil
}

func replaceBranchSpec(xnode xml.Node, name, value, xpathName string) error {
	xname, err := xnode.Search(xpathName)
	if err != nil {
		log.Printf("error searching for env in %s: %v", xpathName, err)
		return err
	}
	if len(xname) == 1 {
		newNameValue := fmt.Sprintf("<name>%s</name>", value)
		xname[0].Replace(newNameValue)
	}
	return nil
}
