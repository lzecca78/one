package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"

	"github.com/lzecca78/one/internal/auth"
	"github.com/lzecca78/one/internal/config"
	"github.com/lzecca78/one/internal/git"
	"github.com/lzecca78/one/internal/jenkins"
	"github.com/lzecca78/one/internal/kubernetes"
	"github.com/lzecca78/one/internal/route53"
	"github.com/lzecca78/one/internal/routes"
	"github.com/lzecca78/one/internal/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var globalLocks *utils.Locks

func main() {

	log.SetFlags(log.Lshortfile)
	v := config.GetConfig()
	client := git.NewGitClient(v)
	jenkinsClient := jenkins.NewJenkinsClient(v)
	kubernetesClient := kubernetes.NewKubernetesClient(v)
	r53cli := route53.NewRoute53Client(v)

	allClients := routes.Clients{
		ViperEnvConfig:   v,
		GitClient:        client,
		JenkinsClient:    jenkinsClient,
		KubernetesClient: kubernetesClient,
		R53client:        r53cli,
	}
	globalLocks = utils.NewLocks()
	router := routes.NewRouter(&allClients, globalLocks)
	r := setup(router)
	r.Run(":8080") // listen and serve on 0.0.0.0:8080
}

func setup(router *routes.Router) *gin.Engine {

	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowCredentials = true
	config.AllowAllOrigins = true
	r.Use(cors.New(config))
	r.Use(func(c *gin.Context) {
		log.Printf("received: %+v", *c.Request)
	})
	api := r.Group("/api")
	auth, err := auth.AdapterSet(router.ViperEnvConfig, api, "/private")
	if err != nil {
		log.Fatalf("error while setting the auth adapter %v", err)
	}
	auth.GET("/config", func(c *gin.Context) {
		c.JSON(http.StatusOK, router.JenkinsClient.Config)
	})
	auth.GET("/repos", func(c *gin.Context) {
		repos, err := router.GitClient.GetRepos(router.JenkinsClient.GetRepos())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, repos)
	})
	auth.POST("/stagings", func(c *gin.Context) {
		var jobsParams jenkins.JobsParameters
		err := c.BindJSON(&jobsParams)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		//create unique namespace name
		namespace := kubernetes.NsNameGen(jobsParams)
		//get lock for for chosen namespace
		globalLocks.LoadOrStoreLock(namespace)
		defer globalLocks.Unlock(namespace)
		//if the namespace already exists, i give as a response a redirect to the already existing namespace
		if router.KubernetesClient.NamespaceAlreadyCreated(namespace) {
			//redirectUrl := fmt.Sprintf("%s/api/stagings/%s", c.Request.Header.Get("HOST"), namespace)
			c.JSON(http.StatusConflict, "resource already exists")
			return
		}
		//check if a new namespace can be created or not
		check, err := PreconditionCheck(router.KubernetesClient, &jobsParams)
		if err != nil {
			c.JSON(http.StatusPreconditionFailed, gin.H{"error": err.Error()})
			return
		}
		if !check {
			message := "maximum number of universes already reached"
			c.JSON(http.StatusPreconditionFailed, message)
			return
		}
		CreateStagingEntity(jobsParams, namespace, router, c)
	})
	auth.DELETE("/stagings/:namespace", router.DeleteNamespace())
	api.DELETE("/stagings/:namespace", router.CheckNamespaceSecret(router.DeleteNamespace()))
	auth.GET("/stagings", func(c *gin.Context) {
		listNs, err := router.KubernetesClient.NamespaceManagedList()
		log.Printf("listNs is: %v", listNs)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, listNs)
	})
	auth.GET("/stagings/:namespace", func(c *gin.Context) {
		namespace := c.Param("namespace")
		//get lock for for chosen namespace
		globalLocks.LoadOrStoreLock(namespace)
		defer globalLocks.Unlock(namespace)
		var data *kubernetes.CloneIngressResponse
		data, _, err := router.KubernetesClient.GetConfigMap(namespace, namespace)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		//resp, err := json.Marshal(data)
		c.JSON(http.StatusOK, data)
	})
	auth.GET("/stagings/:namespace/pipelines/status", func(c *gin.Context) {
		namespace := c.Param("namespace")
		globalLocks.LoadOrStoreLock(namespace)
		defer globalLocks.Unlock(namespace)
		var data jenkins.JobsStatuses
		data, err := router.JenkinsClient.GetJobStatus(namespace)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, data)

	})
	auth.POST("/stagings/:namespace/pipelines/:repo", func(c *gin.Context) {
		namespace := c.Param("namespace")
		repo := c.Param("repo")
		log.Printf("repo is %s", repo)
		globalLocks.LoadOrStoreLock(namespace)
		defer globalLocks.Unlock(namespace)
		data, _, err := router.KubernetesClient.GetConfigMap(namespace, namespace)
		log.Printf("configMap is %+v", data)
		branch := data.ProjectsWithDetails[repo].CVSRefs.Branch
		log.Printf("branch is %s", branch)
		err = router.JenkinsClient.ReplayJob(repo, repo, branch, namespace)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, fmt.Sprintf("re-playing the pipeline in namespace %s for job %s", namespace, repo))
	})
	return r
}

// PreconditionCheck will check if the total number multistaging stable and not are under the params passed from env var
func PreconditionCheck(client *kubernetes.Client, jobsParams *jenkins.JobsParameters) (res bool, err error) {
	res, err = client.UnderMaxNsLimit()
	if err != nil {
		return false, err
	}
	if jobsParams.Stable {
		maxStableLimit, err := client.UnderMaxStableNsLimit()
		res = res && maxStableLimit
		if err != nil {
			return false, err
		}
	}
	return res, nil
}

// CreateStagingEntity is the logic called from the POST api /stagings
func CreateStagingEntity(jobsParams jenkins.JobsParameters, namespace string, router *routes.Router, c *gin.Context) {
	err := router.KubernetesClient.CreateNamespace(namespace, jobsParams.Stable)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	keys := make([]string, 0, len(router.JenkinsClient.Config.RepositoriesProperties.Conf))
	for k := range router.JenkinsClient.Config.RepositoriesProperties.Conf {
		keys = append(keys, k)
	}
	//clone ingress from source namespace (staging) with custom new values
	kresp, err := router.KubernetesClient.CloneIngresses(namespace, keys)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var deleteSecret [32]byte
	_, err = rand.Read(deleteSecret[:])
	if err != nil {
		log.Printf("error reading random secret: %s", err)
	}
	// adding creation of random string for cli authentication on delete api
	deleteSecretString := base64.StdEncoding.EncodeToString(deleteSecret[:])
	//create cronjob that will destroy everything calling the route with DELETE
	err = router.KubernetesClient.CreateCronjob(namespace, jobsParams.Stable, deleteSecretString)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for _, ingressList := range kresp.ProjectsWithDetails {
		for _, ingressHost := range ingressList.Ingresses {
			//create record for each ingress created previously
			resp, err := router.R53client.CreateRecordSet(ingressHost)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			log.Println(resp)
		}
	}
	//initialization of all pipelines of all projects describe in the main config file with custom parameters(branch, namespace and commit)
	projectJobMap, err := router.JenkinsClient.ConfigureJobs(&jobsParams, namespace)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// enrich with jenkins job status
	js, err := router.JenkinsClient.GetJobStatus(namespace)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	newCloneIngResp := kubernetes.EnrichCloneIngressResp(js, kresp, &jobsParams)
	//create cm as persistence layer with CloneIngressResponse struct inside
	err = router.KubernetesClient.CreateConfigMap(newCloneIngResp, projectJobMap, deleteSecretString)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, kresp)
}
