package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/lzecca78/one/internal"
	"github.com/gin-gonic/gin"
	"gopkg.in/go-playground/assert.v1"
)

var router *gin.Engine

func setupRouterOnce() {
	if router == nil {
		log.SetFlags(log.Lshortfile)
		v, repoProperties := internal.GetConfig()
		client := internal.NewGitClient(v)
		jenkinsClient := internal.NewJenkinsClient(v, repoProperties)
		kubernetesClient := internal.NewKubernetesClient(v)
		r53cli := internal.NewRoute53Client(v)
		globalLocks = internal.NewLocks()

		allClients := clients{
			config:           repoProperties,
			gitClient:        client,
			jenkinsClient:    jenkinsClient,
			kubernetesClient: kubernetesClient,
			r53client:        r53cli,
		}
		router = setupRouter(&allClients)
	}
}

func TestApiReposRoute(t *testing.T) {
	setupRouterOnce()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/repos", nil)
	router.ServeHTTP(w, req)
	log.Println(w.Body)

	assert.Equal(t, 200, w.Code)
}

func TestPostApiStagingsRouteBrainfuckBranch(t *testing.T) {
	setupRouterOnce()

	w := httptest.NewRecorder()
	repoName := "portal"
	sha := "123456789"
	branch := "brainfuck"
	contentData := &internal.JobsParameters{
		CommitPerProject: internal.CICommitSpec{
			repoName: internal.Commit{
				Sha:    sha,
				Branch: branch},
		},
	}
	requestBody, err := json.Marshal(contentData)
	if err != nil {
		log.Fatal("there was an error while json marshal: ", err)
	}
	req, _ := http.NewRequest("POST", "/api/stagings", bytes.NewBuffer(requestBody))
	router.ServeHTTP(w, req)
	log.Println(w.Body)

	assert.Equal(t, 201, w.Code)
}

func TestDeleteApiStagingRouteBrainFuckBranch(t *testing.T) {
	setupRouterOnce()
	repoName := "portal"
	sha := "123456789"
	branch := "brainfuck"

	contentData := internal.JobsParameters{
		CommitPerProject: internal.CICommitSpec{
			repoName: internal.Commit{
				Sha:    sha,
				Branch: branch},
		},
	}

	w := httptest.NewRecorder()
	namespace := NsNameGen(contentData)
	deleteContext := fmt.Sprintf("/api/stagings/%s", namespace)
	req, _ := http.NewRequest("DELETE", deleteContext, nil)
	router.ServeHTTP(w, req)
	log.Println(w.Body)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestPostApiStagingsRouteTestMultistagingBranch(t *testing.T) {
	setupRouterOnce()

	w := httptest.NewRecorder()
	repoName := "portal"
	sha := "12345678900"
	branch := "test_multistaging"
	contentData := &internal.JobsParameters{
		CommitPerProject: internal.CICommitSpec{
			repoName: internal.Commit{
				Sha:    sha,
				Branch: branch},
		},
	}
	requestBody, err := json.Marshal(contentData)
	if err != nil {
		log.Fatal("there was an error while json marshal: ", err)
	}
	req, _ := http.NewRequest("POST", "/api/stagings", bytes.NewBuffer(requestBody))
	router.ServeHTTP(w, req)
	log.Println(w.Body)

	assert.Equal(t, 201, w.Code)
}

func TestDeleteApiStagingRouteTestMultistagingBranch(t *testing.T) {
	setupRouterOnce()

	repoName := "portal"
	sha := "12345678900"
	branch := "test_multistaging"

	contentData := internal.JobsParameters{
		CommitPerProject: internal.CICommitSpec{
			repoName: internal.Commit{
				Sha:    sha,
				Branch: branch},
		},
	}

	w := httptest.NewRecorder()
	namespace := NsNameGen(contentData)
	deleteContext := fmt.Sprintf("/api/stagings/%s", namespace)
	req, _ := http.NewRequest("DELETE", deleteContext, nil)
	router.ServeHTTP(w, req)
	log.Println(w.Body)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestConcurrentCreate(t *testing.T) {
	setupRouterOnce()

	repoName := "portal"
	sha := "12345678900"
	branch := "test_multistaging"
	contentData := &internal.JobsParameters{
		CommitPerProject: internal.CICommitSpec{
			repoName: internal.Commit{
				Sha:    sha,
				Branch: branch},
		},
	}
	responseChan := make(chan int)

	requester := func(responseChan chan int, id int) {
		w := httptest.NewRecorder()
		requestBody, err := json.Marshal(contentData)
		if err != nil {
			log.Fatal("there was an error while json marshal: ", err)
		}
		req, _ := http.NewRequest("POST", "/api/stagings", bytes.NewBuffer(requestBody))
		router.ServeHTTP(w, req)
		log.Printf("response for %d = %+v", id, w)
		responseChan <- w.Code
	}

	defer func(namespace string) {
		deleteContext := fmt.Sprintf("/api/stagings/%s", namespace)
		req, _ := http.NewRequest("DELETE", deleteContext, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}(NsNameGen(*contentData))

	go requester(responseChan, 0)
	go requester(responseChan, 1)
	codes := sort.IntSlice{}
	codes = append(codes, <-responseChan)
	codes = append(codes, <-responseChan)
	sort.Sort(codes)

	//invoking also get /api/stagings for checking list of Namespaces
	w := httptest.NewRecorder()
	reqStagings, _ := http.NewRequest("GET", "/api/stagings", nil)
	router.ServeHTTP(w, reqStagings)
	var listNs []internal.MyNameSpace
	log.Printf("body is : %v", w)

	if w.Body == nil {
		t.Fatal("the body is nil")
	}
	err := json.NewDecoder(w.Body).Decode(&listNs)
	if err != nil {
		t.Fatalf("unable to unmarshal json: %v", err)
	}
	var activeNs int
	for _, namespace := range listNs {
		if namespace.Status == "Active" {
			activeNs = activeNs + 1
		}
	}
	assert.Equal(t, http.StatusCreated, codes[0])
	assert.NotEqual(t, http.StatusCreated, codes[1])
	assert.Equal(t, 1, activeNs)
}

func TestPostApiStagingsRouteMultipleProjects(t *testing.T) {
	setupRouterOnce()

	w := httptest.NewRecorder()
	contentData := &internal.JobsParameters{
		CommitPerProject: internal.CICommitSpec{
			"portal": internal.Commit{
				Branch: "master"},
			"docker-mongo": internal.Commit{
				Branch: "docker-mongo"},
			"docker-mysql": internal.Commit{
				Branch: "master"},
			"cards": internal.Commit{
				Branch: "master"},
			"mvrs-servizi-muoversiservizi": internal.Commit{
				Branch: "master",
				Sha:    "asdasdas",
			},
			"portal-ui": internal.Commit{
				Branch: "portal-ui"},
		},
		Stable: false,
	}
	requestBody, err := json.Marshal(contentData)
	if err != nil {
		log.Fatal("there was an error while json marshal: ", err)
	}
	log.Println(bytes.NewBuffer(requestBody))
	req, _ := http.NewRequest("POST", "/api/stagings", bytes.NewBuffer(requestBody))
	router.ServeHTTP(w, req)
	log.Println(w.Body)

	assert.Equal(t, 201, w.Code)
}

func TestDeleteApiStagingsRouteMultipleProjects(t *testing.T) {
	setupRouterOnce()

	w := httptest.NewRecorder()
	contentData := internal.JobsParameters{
		CommitPerProject: internal.CICommitSpec{
			"portal": internal.Commit{
				Branch: "master"},
			"docker-mongo": internal.Commit{
				Branch: "master"},
			"docker-mysql": internal.Commit{
				Branch: "master"},
			"cards": internal.Commit{
				Branch: "master"},
			"mvrs-servizi-muoversiservizi": internal.Commit{
				Branch: "master"},
			"portal-ui": internal.Commit{
				Branch: "master"},
		},
	}

	namespace := NsNameGen(contentData)
	deleteContext := fmt.Sprintf("/api/stagings/%s", namespace)
	req, _ := http.NewRequest("DELETE", deleteContext, nil)
	router.ServeHTTP(w, req)
	log.Println(w.Body)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestNsNameGen(t *testing.T) {

	contentData := internal.JobsParameters{
		CommitPerProject: internal.CICommitSpec{
			"portal": internal.Commit{
				Branch: "master"},
			"docker-mongo": internal.Commit{
				Branch: "master"},
			"docker-mysql": internal.Commit{
				Branch: "master"},
			"cards": internal.Commit{
				Branch: "master"},
			"mvrs-servizi-muoversiservizi": internal.Commit{
				Branch: "master"},
			"portal-ui": internal.Commit{
				Branch: "master"},
		},
	}

	namespace := NsNameGen(contentData)
	log.Println(namespace)
}

//func TestEnrichCloneIngressResp(t *testing.T) {
//	contentData := internal.JobsParameters{
//		CommitPerProject: internal.CICommitSpec{
//			"portal": internal.Commit{
//				Branch: "master"},
//			"docker-mongo": internal.Commit{
//				Branch: "master"},
//			"docker-mysql": internal.Commit{
//				Branch: "master"},
//			"cards": internal.Commit{
//				Branch: "master"},
//			"mvrs-servizi-muoversiservizi": internal.Commit{
//				Branch: "master"},
//			"portal-ui": internal.Commit{
//				Branch: "master"},
//		},
//	}
//
//}
