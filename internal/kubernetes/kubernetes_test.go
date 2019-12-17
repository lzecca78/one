package kubernetes

import (
	"log"
	"testing"
)

func TestNewKubernetesClient(t *testing.T) {
	v, _ := GetConfig()
	NewKubernetesClient(v)
}

func TestCreateNamespace(t *testing.T) {
	ns := "ms-fuffa1"
	v, _ := GetConfig()
	kubernetesClient := NewKubernetesClient(v)
	defer cleanupNs(ns, kubernetesClient)
	err := kubernetesClient.CreateNamespace(ns, false)
	if err != nil {
		t.Fatalf("unable to create namespace %s", ns)
	}
}

func TestFailingCreateNamespace(t *testing.T) {
	ns := "fuffa"
	v, _ := GetConfig()
	kubernetesClient := NewKubernetesClient(v)
	err := kubernetesClient.CreateNamespace(ns, false)
	if err == nil {
		t.Fatalf("unable to create namespace %s", ns)
	}
}

func TestCloneIngresses(t *testing.T) {
	dstNamespace := "ms-fuffa2"
	ingressDetails := []string{
		"portal-ui",
		"cards",
		"portal",
		"msrvz",
	}
	v, _ := GetConfig()
	kubernetesClient := NewKubernetesClient(v)
	kubernetesClient.CreateNamespace(dstNamespace, false)
	defer cleanupNs(dstNamespace, kubernetesClient)
	_, err := kubernetesClient.CloneIngresses(dstNamespace, ingressDetails)
	if err != nil {
		t.Fatal()
	}
}

func TestCreateConfigMap(t *testing.T) {
	fixturesCm := CloneIngressResponse{
		NamespaceCreated:    "ms-test-cm1",
		ProjectsWithDetails: &StatusPerProject{"portal": &MultistagingSpecs{Ingresses: []string{"pippo.pluto.it", "ciao.ciao.it"}}},
	}
	v, _ := GetConfig()
	kubernetesClient := NewKubernetesClient(v)
	defer cleanupNs("ms-test-cm1", kubernetesClient)
	kubernetesClient.CreateNamespace("ms-test-cm1", false)
	err := kubernetesClient.CreateConfigMap(&fixturesCm, map[string]string{})
	if err != nil {
		t.Fatal()
	}

}

func TestGetConfigMap(t *testing.T) {
	fixturesCm := CloneIngressResponse{
		NamespaceCreated:    "ms-test-cm2",
		ProjectsWithDetails: &StatusPerProject{"portal": &MultistagingSpecs{Ingresses: []string{"pippo.pluto.it", "ciao.ciao.it"}}},
	}
	v, _ := GetConfig()
	kubernetesClient := NewKubernetesClient(v)
	defer cleanupNs("ms-test-cm2", kubernetesClient)
	kubernetesClient.CreateNamespace("ms-test-cm2", false)
	err := kubernetesClient.CreateConfigMap(&fixturesCm, map[string]string{})
	resp, jobs, err := kubernetesClient.GetConfigMap(fixturesCm.NamespaceCreated, fixturesCm.NamespaceCreated)
	if err != nil {
		t.Fatal()
	}
	log.Printf("RESP: %v", resp)
	log.Printf("JOBS: %v", jobs)
}

func cleanupNs(ns string, kubernetesClient *KubernetesClient) {
	err := kubernetesClient.DeleteNamespace(ns)
	if err != nil {
		log.Fatalf("unable to delete namespace %s", ns)
	}
}
