package route53

import (
	"log"
	"testing"
)

func TestNewRoute53Client(t *testing.T) {
	v, _ := GetConfig()
	NewRoute53Client(v)
}

func TestCreateRecordSet(t *testing.T) {
	record := "test2.example.com"
	v, _ := GetConfig()
	r53 := NewRoute53Client(v)
	createRecord, err := r53.CreateRecordSet(record)
	if err != nil {
		t.Fatalf("unable to create records %s: %v", record, err)
	}
	log.Println(createRecord)
}

func TestDeleteRecordSet(t *testing.T) {
	record := "test2.example.com"
	v, _ := GetConfig()
	r53 := NewRoute53Client(v)
	createRecord, err := r53.DeleteRecordSet(record)
	if err != nil {
		t.Fatalf("unable to delete records %s: %v", record, err)
	}
	log.Println(createRecord)
}
