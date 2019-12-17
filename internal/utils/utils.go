package utils

import (
	"log"

	"github.com/lzecca78/one/internal/git"
)

//StatusPerProject type-def with map of string and IngressesWithStatus
type StatusPerProject map[string]*MultistagingSpecs

// MultistagingSpecs is a complex struct that describe the multistaging environment
type MultistagingSpecs struct {
	Ingresses []string   `json:"ingresses"`
	Status    string     `json:"status"`
	JobName   string     `json:"job_name"`
	CVSRefs   git.Commit `json:"cvs_refs"`
}

// RemoveDuplicatesFromSlice remove duplicate item from a slice
func RemoveDuplicatesFromSlice(s []string) []string {
	m := make(map[string]bool)
	for _, item := range s {
		if _, ok := m[item]; ok {
			// duplicate item
			log.Println(item, "is a duplicate")
		} else {
			m[item] = true
		}
	}

	var result []string
	for item := range m {
		result = append(result, item)
	}
	return result
}
