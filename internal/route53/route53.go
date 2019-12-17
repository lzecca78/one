package route53

import (
	"context"
	"fmt"
	"log"

	"github.com/lzecca78/one/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/spf13/viper"
)

// AliasRecord gives alias abstraction with zoneid and lb cname
type AliasRecord struct {
	ZoneID  string
	LbCname string
}

// RClient is a client r53 with necessary private and public records
type RClient struct {
	R53Client     *route53.Client
	PublicRecord  AliasRecord
	PrivateRecord AliasRecord
}

//NewRoute53Client initialize a r53 client
func NewRoute53Client(v *viper.Viper) *RClient {
	privateZoneID := config.CheckAndGetString(v, "R53_PVT_ZID")
	publicZoneID := config.CheckAndGetString(v, "R53_PUB_ZID")
	publicLbCname := config.CheckAndGetString(v, "LB_PUBLIC_CNAME")
	privateLbCname := config.CheckAndGetString(v, "LB_PRIVATE_CNAME")

	config, err := external.LoadDefaultAWSConfig()
	if err != nil {
		log.Fatal("unable to load aws config:", err)
	}

	pubRecord := AliasRecord{
		ZoneID:  privateZoneID,
		LbCname: privateLbCname,
	}
	privRecord := AliasRecord{
		ZoneID:  publicZoneID,
		LbCname: publicLbCname,
	}

	return &RClient{
		route53.New(config),
		pubRecord,
		privRecord,
	}
}

// CreateRecordSet will create a record on r53
func (r *RClient) CreateRecordSet(record string) ([]*route53.ChangeResourceRecordSetsResponse, error) {
	return r.r53Action("UPSERT", record)
}

// DeleteRecordSet will delete a record on r53
func (r *RClient) DeleteRecordSet(record string) ([]*route53.ChangeResourceRecordSetsResponse, error) {
	return r.r53Action("DELETE", record)
}

func (r *RClient) r53Action(action, record string) ([]*route53.ChangeResourceRecordSetsResponse, error) {
	listRecords := []AliasRecord{
		r.PrivateRecord,
		r.PublicRecord,
	}
	var err error
	var responses []*route53.ChangeResourceRecordSetsResponse
	for _, recSet := range listRecords {
		message := fmt.Sprintf("%s record %s in zone %s to target %s", action, record, recSet.ZoneID, recSet.LbCname)
		input := recordSetInput(message, record, route53.ChangeAction(action), recSet)
		req := r.R53Client.ChangeResourceRecordSetsRequest(input)
		resp, err := req.Send(context.TODO())
		if err != nil {
			log.Printf("there was en error: %s: %s", resp, err)
		}
		responses = append(responses, resp)
	}
	return responses, err
}

func recordSetInput(message, record string, action route53.ChangeAction, aliasSet AliasRecord) *route53.ChangeResourceRecordSetsInput {
	changeResourceInput := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []route53.Change{
				{
					Action: action,
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(record),
						Type: "CNAME",
						TTL:  aws.Int64(30),
						ResourceRecords: []route53.ResourceRecord{
							{Value: aws.String(aliasSet.LbCname)},
						},
					},
				},
			},
			Comment: aws.String(message),
		},
		HostedZoneId: aws.String(aliasSet.ZoneID),
	}
	return changeResourceInput
}
