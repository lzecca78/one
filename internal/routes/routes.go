package routes

import (
	"log"
	"net/http"

	"github.com/lzecca78/one/internal/git"
	"github.com/lzecca78/one/internal/jenkins"
	"github.com/lzecca78/one/internal/kubernetes"
	"github.com/lzecca78/one/internal/route53"
	"github.com/lzecca78/one/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// Clients abastracts all clients needed to setup all the implementation neeeded by api
type Clients struct {
	ViperEnvConfig   *viper.Viper
	GitClient        *git.Client
	JenkinsClient    *jenkins.JenkinsClient
	KubernetesClient *kubernetes.Client
	R53client        *route53.RClient
}

// Router abstracts  the router needs
type Router struct {
	*Clients
	*utils.Locks
}

// NewRouter  setup the Router struct
func NewRouter(clients *Clients, locks *utils.Locks) *Router {
	return &Router{
		Clients: clients,
		Locks:   locks,
	}
}

// CheckNamespaceSecret is a specific auth function for cronjob
func (router *Router) CheckNamespaceSecret(f gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		namespace := c.Param("namespace")
		_, projectJobMap, err := router.KubernetesClient.GetConfigMap(namespace, namespace)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		queryDeleteSecret, ok := c.GetQuery(kubernetes.DeleteSecret)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "delete_secret query parameter not found"})
			return
		}
		if mapDeleteSecret, ok := projectJobMap[kubernetes.DeleteSecret]; ok {
			if queryDeleteSecret == mapDeleteSecret {
				f(c)
			} else {
				log.Printf("%s query param %s not matching %s", kubernetes.DeleteSecret, queryDeleteSecret, mapDeleteSecret)
				c.JSON(http.StatusUnauthorized, gin.H{"error": "delete_secret not matching"})
				return
			}

		}
	}

}

// DeleteNamespace will delete the namespace passed as an api field
func (router *Router) DeleteNamespace() gin.HandlerFunc {
	return func(c *gin.Context) {
		namespace := c.Param("namespace")
		//get lock for for chosen namespace
		router.LoadOrStoreLock(namespace)
		defer router.Unlock(namespace)
		//delete records in route53
		cfgMapData, projectJobMap, err := router.KubernetesClient.GetConfigMap(namespace, namespace)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.Printf("will delete jobs %v", projectJobMap)
		err = router.JenkinsClient.DeleteFolder(namespace)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		for project, records := range cfgMapData.ProjectsWithDetails {
			for _, record := range records.Ingresses {
				log.Printf("deleting record %s for project %s in namespace %s", record, project, namespace)
				router.R53client.DeleteRecordSet(record)
			}
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		//delete namespace
		err = router.KubernetesClient.DeleteNamespace(namespace)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNoContent, nil)
	}
}
