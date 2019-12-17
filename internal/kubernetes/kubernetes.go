package kubernetes

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/lzecca78/one/internal/config"
	"github.com/lzecca78/one/internal/jenkins"
	"github.com/lzecca78/one/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	batchV1 "k8s.io/api/batch/v1"
	v1BatchB1 "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	v1b1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	JobsLabelConfigmap = "jobs"
	DeleteSecret       = "delete_secret"
)

// KubernetesClient is a struct that inherits all the capabilities of a needed kubernetes datas
type Client struct {
	clientSet               *kubernetes.Clientset
	srcNamespace            string
	namespaceValidator      func(string) bool
	maxUniverseNumber       int
	maxStableUniverseNumber int
	seppukuSecret           string
	url                     string
}

// DefaultNamespaceValidator is a function that change  the namespace adding a prefix
func DefaultNamespaceValidator(namespace string) bool {
	return strings.Contains(namespace, "ms-")
}

// NewKubernetesClient initialize the Client struct
func NewKubernetesClient(v *viper.Viper, checkNamespaces ...func(string) bool) *Client {
	checkNamespace := DefaultNamespaceValidator
	if len(checkNamespaces) > 0 {
		checkNamespace = checkNamespaces[0]
	}
	//not using viper to avoid prefix
	kubeconfig := os.Getenv("KUBECONFIG")
	var cfg *rest.Config
	var err error
	if kubeconfig != "" {
		log.Println("using kubeconfig from (first config specified in KUBECONFIG env var) ", kubeconfig)
		kubeconfig = strings.Split(kubeconfig, ":")[0]
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		log.Println("using inCluster configuration")
		cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		log.Fatal("failed to load kubeconfig: ", err)
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatal("failed creating client from config", err)
	}
	srcNamespace := config.CheckAndGetString(v, "K8S_SRCNAMESPACE")
	maxNamespace := config.CheckAndGetString(v, "MAX_UNIVERSE")
	maxStableNamespace := config.CheckAndGetString(v, "MAX_STABLE_UNIVERSE")
	//TODO get url from ingress one(you need at least the ingress name)
	myURL := config.CheckAndGetString(v, "MY_URL")
	maxNumNs, err := strconv.Atoi(maxNamespace)
	maxNumStableNs, err := strconv.Atoi(maxStableNamespace)
	if err != nil {
		log.Fatal("failed converting to int:", err)
	}
	return &Client{
		clientSet:               clientset,
		srcNamespace:            srcNamespace,
		namespaceValidator:      checkNamespace,
		maxUniverseNumber:       maxNumNs,
		maxStableUniverseNumber: maxNumStableNs,
		url:                     myURL,
	}
}

// MyNameSpace describe the namespace informations needed
type MyNameSpace struct {
	Name   string `json:"name" yaml:"name"`
	Stable bool   `json:"stable" yaml:"stable"`
	Status string `json:"status" yaml:"status"`
}

// NamespaceManagedList will list all kubernetes namespace handled by one
func (k *Client) NamespaceManagedList() (nslist []MyNameSpace, err error) {
	labelselector := "scope=multistaging"
	namespaces, err := k.clientSet.CoreV1().Namespaces().List(
		metav1.ListOptions{
			LabelSelector: labelselector,
		},
	)
	if err != nil {
		log.Printf("error while getting ns list: %s", err)
		return nil, err
	}
	for _, namespace := range namespaces.Items {
		stableBool, err := strconv.ParseBool(namespace.Labels["stable"])
		if err != nil {
			fmt.Printf("error while converting string to boolean : %v", err)
			return nil, err
		}

		nslist = append(nslist, MyNameSpace{
			Name:   namespace.ObjectMeta.Name,
			Stable: stableBool,
			Status: fmt.Sprintf("%v", namespace.Status.Phase),
		})
	}
	return nslist, nil
}

// NsConstraintReqs check the ns constraints
func (k *Client) NsConstraintReqs() error {
	maxStableNs := k.maxStableUniverseNumber
	maxNs := k.maxUniverseNumber
	if maxStableNs > maxNs {
		return errors.Errorf("Stable Namespaces number are greater than Total Namespaces")
	}
	return nil
}

// UnderMaxNsLimit  check the ns number limits
func (k *Client) UnderMaxNsLimit() (bool, error) {
	err := k.NsConstraintReqs()
	if err != nil {
		log.Printf("constraint of namespaces number limits broken: %s", err)
		return false, err
	}
	myNamespaces, err := k.NamespaceManagedList()
	if err != nil {
		log.Printf("error while getting list of namespace : %v", err)
	}
	var activeNs int
	for _, namespace := range myNamespaces {
		if namespace.Status == "Active" {
			activeNs = activeNs + 1
		}

	}
	if activeNs >= k.maxUniverseNumber {
		return false, errors.Errorf("total number of namespace reached: %v", activeNs)
	}
	return true, nil
}

// UnderMaxStableNsLimit check the number of stable ns
func (k *Client) UnderMaxStableNsLimit() (bool, error) {
	err := k.NsConstraintReqs()
	if err != nil {
		log.Printf("constraint of namespaces number limits broken: %s", err)
		return false, err
	}
	myNamespaces, err := k.NamespaceManagedList()
	if err != nil {
		log.Printf("error while getting list of namespace : %v", err)
	}
	var activeStableNs int
	for _, namespace := range myNamespaces {
		if namespace.Status == "Active" && namespace.Stable {
			activeStableNs = activeStableNs + 1
		}
	}
	if activeStableNs >= k.maxStableUniverseNumber {
		return false, errors.Errorf("total number of stable namespace reached: %v", activeStableNs)
	}
	return true, nil
}

// CreateNamespace will create a kubernetes namespace adding stable labels
func (k *Client) CreateNamespace(namespace string, stable bool) error {
	if !k.namespaceValidator(namespace) {
		return errors.Errorf("namespace %s does not satisfy given namespaceValidator function", namespace)
	}
	namespaces := k.clientSet.CoreV1().Namespaces()
	namespaceResource := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"scope":  "multistaging",
				"stable": fmt.Sprintf("%v", stable),
			},
		},
	}
	_, err := namespaces.Create(namespaceResource)
	if err != nil {
		log.Printf("error creating namespace %s: %v\n", namespace, err)
		_, err := namespaces.Update(namespaceResource)
		if err != nil {
			return err
		}
	}
	return nil
}

// NamespaceAlreadyCreated check if the namespace is already created or not
func (k *Client) NamespaceAlreadyCreated(namespace string) bool {
	_, err := k.clientSet.CoreV1().Namespaces().Get(
		namespace,
		metav1.GetOptions{},
	)
	if err != nil {
		return false
	}
	return true
}

//DeleteNamespace if a function useful to delete namespace
func (k *Client) DeleteNamespace(namespace string) error {
	if !k.namespaceValidator(namespace) {
		return errors.Errorf("namespace %s does not satisfy given namespaceValidator function", namespace)
	}

	return k.clientSet.CoreV1().Namespaces().Delete(namespace, nil)
}

//CloneIngresses will clone ingresses from a source namespace
func (k *Client) CloneIngresses(dstNamespace string, projects []string) (*CloneIngressResponse, error) {
	srcIngresses := k.clientSet.ExtensionsV1beta1().Ingresses(k.srcNamespace)
	dstIngresses := k.clientSet.ExtensionsV1beta1().Ingresses(dstNamespace)
	hosts := utils.StatusPerProject{}
	for _, project := range projects {
		currentIngressWithStatus, ok := hosts[project]
		if !ok {
			//TODO add JobName in IngressesWithStatus
			currentIngressWithStatus = &utils.MultistagingSpecs{}
			hosts[project] = currentIngressWithStatus
		}
		currentIngressWithStatus.Ingresses = []string{}
		newJobName := jenkins.GetNewJobName(project, dstNamespace)
		currentIngressWithStatus.JobName = newJobName
		ingressList, err := srcIngresses.List(metav1.ListOptions{
			LabelSelector: labels.Set(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"project": project,
				},
			}.MatchLabels).String(),
			Limit: 100,
		})
		if err != nil {
			log.Fatalf("error getting ingresses for %s in namespace %s", project, k.srcNamespace)
		}
		for _, ingress := range ingressList.Items {
			ingress.ObjectMeta = metav1.ObjectMeta{
				Namespace:   dstNamespace,
				Name:        ingress.ObjectMeta.Name,
				Annotations: ingress.ObjectMeta.Annotations}
			ingress.Status = v1b1.IngressStatus{}
			currentIngressWithStatus, ok := hosts[project]
			if !ok {
				//TODO add JobName in IngressesWithStatus
				currentIngressWithStatus = &utils.MultistagingSpecs{}
				hosts[project] = currentIngressWithStatus
			}
			for idx, currentRule := range ingress.Spec.Rules {
				host := currentRule.Host
				splitHost := strings.Split(host, ".")
				domain := strings.Join(splitHost[1:], ".")
				newHost := fmt.Sprintf("%s-%s.%s", dstNamespace, splitHost[0], domain)
				log.Printf("currentIngressWithStatus is %v", currentIngressWithStatus)
				currentIngressWithStatus.Ingresses = append(currentIngressWithStatus.Ingresses, newHost)
				currentIngressWithStatus.Ingresses = utils.RemoveDuplicatesFromSlice(currentIngressWithStatus.Ingresses)
				ingress.Spec.Rules[idx].Host = newHost
			}
			_, err := dstIngresses.Create(&ingress)
			if err != nil {
				log.Printf("error while creating ingress for project %s with error %s", project, err)
				log.Println("will try updating")
				_, err := dstIngresses.Update(&ingress)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	result := &CloneIngressResponse{
		NamespaceCreated:    dstNamespace,
		ProjectsWithDetails: hosts,
	}
	return result, nil
}

//CreateConfigMap is a function that allow to create a configMap in a namespace with a specific content
func (k *Client) CreateConfigMap(data *CloneIngressResponse, projectJobMap map[string]string, deleteSecret string) error {
	cfgMap := k.clientSet.CoreV1().ConfigMaps(data.NamespaceCreated)
	pcfgMap := persistentCfgMap(data, projectJobMap, deleteSecret)
	_, err := cfgMap.Create(pcfgMap)
	if err != nil {
		log.Printf("error while creating configmap %s for project %s with error %v", pcfgMap.ObjectMeta.Name, pcfgMap.ObjectMeta.Namespace, err)
		log.Println("will try updating")
		_, err := cfgMap.Update(pcfgMap)
		if err != nil {
			return err
		}
	}
	log.Printf("created configmap: %v", *data)
	return nil
}

//GetConfigMap is a function that allow to fetch the data in a configmap, with error
func (k *Client) GetConfigMap(namespace, cmName string) (*CloneIngressResponse, map[string]string, error) {
	cfgMap := k.clientSet.CoreV1().ConfigMaps(namespace)
	cm, err := cfgMap.Get(cmName, metav1.GetOptions{})

	data, ok := cm.Data[namespace]
	if !ok {
		log.Printf("not found %s value in configmap %s: %v ", namespace, cmName, cm)
		return nil, nil, errors.Errorf("not found %s value in configmap %s: %v ", namespace, cmName, cm)
	}
	var datas *CloneIngressResponse
	json.Unmarshal([]byte(data), &datas)

	jobs, ok := cm.Data[JobsLabelConfigmap]
	if !ok {
		log.Printf("not found %s value in configmap %s: %v ", JobsLabelConfigmap, cmName, cm)
		return nil, nil, errors.Errorf("not found %s value in configmap %s: %v ", JobsLabelConfigmap, cmName, cm)
	}
	var projectJobMap map[string]string
	json.Unmarshal([]byte(jobs), &projectJobMap)

	return datas, projectJobMap, err
}

func persistentCfgMap(data *CloneIngressResponse, projectJobMap map[string]string, deleteSecret string) *v1.ConfigMap {
	content, err := json.Marshal(data)
	if err != nil {
		log.Printf("error while converting to json: %s", err)
	}
	jobMapJSON, err := json.Marshal(projectJobMap)
	if err != nil {
		log.Printf("error while converting to json: %s", err)
	}
	cm := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: data.NamespaceCreated,
		},
		Data: map[string]string{
			data.NamespaceCreated: string(content),
			JobsLabelConfigmap:    string(jobMapJSON),
			DeleteSecret:          deleteSecret,
		},
	}
	return &cm
}

// CreateCronjob will create the seppuku cronjob for self-killing task
func (k *Client) CreateCronjob(namespace string, stable bool, deleteSecret string) error {
	cj := &v1BatchB1.CronJob{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("seppuku-%s", namespace),
		},
		Spec: v1BatchB1.CronJobSpec{
			Schedule: "0 20 * * *",
			Suspend:  &stable,
			JobTemplate: v1BatchB1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("job-seppuku-%s", namespace),
				},
				Spec: *createJob(namespace, k.url, deleteSecret),
			},
		},
	}
	_, err := k.clientSet.BatchV1beta1().CronJobs(namespace).Create(cj)
	if err != nil {
		log.Printf("error while create cronjob in namespace: %s with error: %v", namespace, err)
	}
	return err
}

func createJob(namespace, url, secret string) *batchV1.JobSpec {
	deleteURL := fmt.Sprintf("%s/api/stagings/%s?%s=%s", url, namespace, DeleteSecret, secret)
	job := &batchV1.JobSpec{
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: v1.PodSpec{
				RestartPolicy: v1.RestartPolicyNever,
				Containers: []v1.Container{
					v1.Container{
						Name:  "seppuku",
						Image: "ubuntu:18.04",
						Command: []string{
							"/bin/bash",
							"-c",
						},
						Args: []string{
							"apt update && apt install -y curl && curl -XDELETE " + deleteURL,
						},
						Resources: v1.ResourceRequirements{
							Limits: v1.ResourceList{
								v1.ResourceCPU:    resource.MustParse("250m"),
								v1.ResourceMemory: resource.MustParse("150Mi"),
							},
							Requests: v1.ResourceList{
								v1.ResourceCPU:    resource.MustParse("75m"),
								v1.ResourceMemory: resource.MustParse("40Mi"),
							},
						},
					},
				},
			},
		},
	}
	return job
}

//ClientRequest is the atom of the structure of the response of the client once it envelop the choose step
type projectGitProperties struct {
	project string
	sha     string
	branch  string
}

//NsNameGen is the function that return the unique namespace
func NsNameGen(jobs jenkins.JobsParameters) string {
	cleanList := []projectGitProperties{}
	for project, commit := range jobs.CommitPerProject {
		cleanList = append(cleanList, projectGitProperties{
			project: project,
			sha:     commit.Sha,
			branch:  commit.Branch,
		})
	}
	sort.Slice(cleanList, func(i, j int) bool {
		return strings.Compare(cleanList[i].project, cleanList[j].project) > 0
	})
	var concatList string
	for _, element := range cleanList {
		concatList = concatList + element.project + element.sha + element.branch
	}
	//convert to sha512 and taking first 8 char
	toByte := []byte(concatList)
	shaValue := sha512.Sum512(toByte)
	encodedSha := hex.EncodeToString(shaValue[:])[:8]

	return fmt.Sprintf("ms-%v", encodedSha)
}

// CloneIngressResponse  give the response from cloneIngress
type CloneIngressResponse struct {
	ProjectsWithDetails utils.StatusPerProject `json:"projects_with_details" yaml:"projects_with_details"`
	NamespaceCreated    string                 `json:"namespace_created" yaml:"namespace_created"`
}

// jobStatuses :: {jobName: status}
// jobParams :: { stable? : ... , CommitPerProject : { projectName: { ...Commit ... }}}
// cloneIngressResp :: {projects_with_details : { projectName: { jobName : ... , CSVRefs : ... } } }
func EnrichCloneIngressResp(jobStatus jenkins.JobsStatuses, cloneIngressResp *CloneIngressResponse, jobParams *jenkins.JobsParameters) *CloneIngressResponse {
	for projectName, commitspec := range jobParams.CommitPerProject {
		projectDetails, ok := cloneIngressResp.ProjectsWithDetails[projectName]
		if !ok {
			// assuming cloneIngressResp to contain all projects
			log.Fatalf("not found %s in %v", projectName, cloneIngressResp)
		}
		status, ok := jobStatus[projectDetails.JobName]
		if !ok {
			// assuming jobStatus to contain all jobs
			log.Fatalf("not found %s in %v", projectDetails.JobName, jobStatus)
		}
		projectDetails.Status = status
		projectDetails.CVSRefs = commitspec
	}
	log.Printf("enriched CloneIngressResponse: %v", &cloneIngressResp)
	return cloneIngressResp
}
