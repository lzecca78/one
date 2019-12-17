// Code generated by private/model/cli/gen-api/main.go. DO NOT EDIT.

package route53

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/internal/awsutil"
	"github.com/aws/aws-sdk-go-v2/private/protocol"
)

// A complex type that contains information about the health checks or hosted
// zones for which you want to list tags.
// Please also see https://docs.aws.amazon.com/goto/WebAPI/route53-2013-04-01/ListTagsForResourcesRequest
type ListTagsForResourcesInput struct {
	_ struct{} `locationName:"ListTagsForResourcesRequest" type:"structure" xmlURI:"https://route53.amazonaws.com/doc/2013-04-01/"`

	// A complex type that contains the ResourceId element for each resource for
	// which you want to get a list of tags.
	//
	// ResourceIds is a required field
	ResourceIds []string `locationNameList:"ResourceId" min:"1" type:"list" required:"true"`

	// The type of the resources.
	//
	//    * The resource type for health checks is healthcheck.
	//
	//    * The resource type for hosted zones is hostedzone.
	//
	// ResourceType is a required field
	ResourceType TagResourceType `location:"uri" locationName:"ResourceType" type:"string" required:"true" enum:"true"`
}

// String returns the string representation
func (s ListTagsForResourcesInput) String() string {
	return awsutil.Prettify(s)
}

// Validate inspects the fields of the type to determine if they are valid.
func (s *ListTagsForResourcesInput) Validate() error {
	invalidParams := aws.ErrInvalidParams{Context: "ListTagsForResourcesInput"}

	if s.ResourceIds == nil {
		invalidParams.Add(aws.NewErrParamRequired("ResourceIds"))
	}
	if s.ResourceIds != nil && len(s.ResourceIds) < 1 {
		invalidParams.Add(aws.NewErrParamMinLen("ResourceIds", 1))
	}
	if len(s.ResourceType) == 0 {
		invalidParams.Add(aws.NewErrParamRequired("ResourceType"))
	}

	if invalidParams.Len() > 0 {
		return invalidParams
	}
	return nil
}

// MarshalFields encodes the AWS API shape using the passed in protocol encoder.
func (s ListTagsForResourcesInput) MarshalFields(e protocol.FieldEncoder) error {

	e.SetFields(protocol.BodyTarget, "ListTagsForResourcesRequest", protocol.FieldMarshalerFunc(func(e protocol.FieldEncoder) error {
		if len(s.ResourceIds) > 0 {
			v := s.ResourceIds

			metadata := protocol.Metadata{ListLocationName: "ResourceId"}
			ls0 := e.List(protocol.BodyTarget, "ResourceIds", metadata)
			ls0.Start()
			for _, v1 := range v {
				ls0.ListAddValue(protocol.StringValue(v1))
			}
			ls0.End()

		}
		return nil
	}), protocol.Metadata{XMLNamespaceURI: "https://route53.amazonaws.com/doc/2013-04-01/"})
	if len(s.ResourceType) > 0 {
		v := s.ResourceType

		metadata := protocol.Metadata{}
		e.SetValue(protocol.PathTarget, "ResourceType", v, metadata)
	}
	return nil
}

// A complex type containing tags for the specified resources.
// Please also see https://docs.aws.amazon.com/goto/WebAPI/route53-2013-04-01/ListTagsForResourcesResponse
type ListTagsForResourcesOutput struct {
	_ struct{} `type:"structure"`

	// A list of ResourceTagSets containing tags associated with the specified resources.
	//
	// ResourceTagSets is a required field
	ResourceTagSets []ResourceTagSet `locationNameList:"ResourceTagSet" type:"list" required:"true"`
}

// String returns the string representation
func (s ListTagsForResourcesOutput) String() string {
	return awsutil.Prettify(s)
}

// MarshalFields encodes the AWS API shape using the passed in protocol encoder.
func (s ListTagsForResourcesOutput) MarshalFields(e protocol.FieldEncoder) error {
	if len(s.ResourceTagSets) > 0 {
		v := s.ResourceTagSets

		metadata := protocol.Metadata{ListLocationName: "ResourceTagSet"}
		ls0 := e.List(protocol.BodyTarget, "ResourceTagSets", metadata)
		ls0.Start()
		for _, v1 := range v {
			ls0.ListAddFields(v1)
		}
		ls0.End()

	}
	return nil
}

const opListTagsForResources = "ListTagsForResources"

// ListTagsForResourcesRequest returns a request value for making API operation for
// Amazon Route 53.
//
// Lists tags for up to 10 health checks or hosted zones.
//
// For information about using tags for cost allocation, see Using Cost Allocation
// Tags (https://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/cost-alloc-tags.html)
// in the AWS Billing and Cost Management User Guide.
//
//    // Example sending a request using ListTagsForResourcesRequest.
//    req := client.ListTagsForResourcesRequest(params)
//    resp, err := req.Send(context.TODO())
//    if err == nil {
//        fmt.Println(resp)
//    }
//
// Please also see https://docs.aws.amazon.com/goto/WebAPI/route53-2013-04-01/ListTagsForResources
func (c *Client) ListTagsForResourcesRequest(input *ListTagsForResourcesInput) ListTagsForResourcesRequest {
	op := &aws.Operation{
		Name:       opListTagsForResources,
		HTTPMethod: "POST",
		HTTPPath:   "/2013-04-01/tags/{ResourceType}",
	}

	if input == nil {
		input = &ListTagsForResourcesInput{}
	}

	req := c.newRequest(op, input, &ListTagsForResourcesOutput{})
	return ListTagsForResourcesRequest{Request: req, Input: input, Copy: c.ListTagsForResourcesRequest}
}

// ListTagsForResourcesRequest is the request type for the
// ListTagsForResources API operation.
type ListTagsForResourcesRequest struct {
	*aws.Request
	Input *ListTagsForResourcesInput
	Copy  func(*ListTagsForResourcesInput) ListTagsForResourcesRequest
}

// Send marshals and sends the ListTagsForResources API request.
func (r ListTagsForResourcesRequest) Send(ctx context.Context) (*ListTagsForResourcesResponse, error) {
	r.Request.SetContext(ctx)
	err := r.Request.Send()
	if err != nil {
		return nil, err
	}

	resp := &ListTagsForResourcesResponse{
		ListTagsForResourcesOutput: r.Request.Data.(*ListTagsForResourcesOutput),
		response:                   &aws.Response{Request: r.Request},
	}

	return resp, nil
}

// ListTagsForResourcesResponse is the response type for the
// ListTagsForResources API operation.
type ListTagsForResourcesResponse struct {
	*ListTagsForResourcesOutput

	response *aws.Response
}

// SDKResponseMetdata returns the response metadata for the
// ListTagsForResources request.
func (r *ListTagsForResourcesResponse) SDKResponseMetdata() *aws.Response {
	return r.response
}
