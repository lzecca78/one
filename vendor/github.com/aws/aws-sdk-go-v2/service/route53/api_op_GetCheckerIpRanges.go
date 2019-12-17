// Code generated by private/model/cli/gen-api/main.go. DO NOT EDIT.

package route53

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/internal/awsutil"
	"github.com/aws/aws-sdk-go-v2/private/protocol"
)

// Empty request.
// Please also see https://docs.aws.amazon.com/goto/WebAPI/route53-2013-04-01/GetCheckerIpRangesRequest
type GetCheckerIpRangesInput struct {
	_ struct{} `type:"structure"`
}

// String returns the string representation
func (s GetCheckerIpRangesInput) String() string {
	return awsutil.Prettify(s)
}

// MarshalFields encodes the AWS API shape using the passed in protocol encoder.
func (s GetCheckerIpRangesInput) MarshalFields(e protocol.FieldEncoder) error {

	return nil
}

// A complex type that contains the CheckerIpRanges element.
// Please also see https://docs.aws.amazon.com/goto/WebAPI/route53-2013-04-01/GetCheckerIpRangesResponse
type GetCheckerIpRangesOutput struct {
	_ struct{} `type:"structure"`

	// A complex type that contains sorted list of IP ranges in CIDR format for
	// Amazon Route 53 health checkers.
	//
	// CheckerIpRanges is a required field
	CheckerIpRanges []string `type:"list" required:"true"`
}

// String returns the string representation
func (s GetCheckerIpRangesOutput) String() string {
	return awsutil.Prettify(s)
}

// MarshalFields encodes the AWS API shape using the passed in protocol encoder.
func (s GetCheckerIpRangesOutput) MarshalFields(e protocol.FieldEncoder) error {
	if len(s.CheckerIpRanges) > 0 {
		v := s.CheckerIpRanges

		metadata := protocol.Metadata{}
		ls0 := e.List(protocol.BodyTarget, "CheckerIpRanges", metadata)
		ls0.Start()
		for _, v1 := range v {
			ls0.ListAddValue(protocol.StringValue(v1))
		}
		ls0.End()

	}
	return nil
}

const opGetCheckerIpRanges = "GetCheckerIpRanges"

// GetCheckerIpRangesRequest returns a request value for making API operation for
// Amazon Route 53.
//
//
// GetCheckerIpRanges still works, but we recommend that you download ip-ranges.json,
// which includes IP address ranges for all AWS services. For more information,
// see IP Address Ranges of Amazon Route 53 Servers (http://docs.aws.amazon.com/Route53/latest/DeveloperGuide/route-53-ip-addresses.html)
// in the Amazon Route 53 Developer Guide.
//
//    // Example sending a request using GetCheckerIpRangesRequest.
//    req := client.GetCheckerIpRangesRequest(params)
//    resp, err := req.Send(context.TODO())
//    if err == nil {
//        fmt.Println(resp)
//    }
//
// Please also see https://docs.aws.amazon.com/goto/WebAPI/route53-2013-04-01/GetCheckerIpRanges
func (c *Client) GetCheckerIpRangesRequest(input *GetCheckerIpRangesInput) GetCheckerIpRangesRequest {
	op := &aws.Operation{
		Name:       opGetCheckerIpRanges,
		HTTPMethod: "GET",
		HTTPPath:   "/2013-04-01/checkeripranges",
	}

	if input == nil {
		input = &GetCheckerIpRangesInput{}
	}

	req := c.newRequest(op, input, &GetCheckerIpRangesOutput{})
	return GetCheckerIpRangesRequest{Request: req, Input: input, Copy: c.GetCheckerIpRangesRequest}
}

// GetCheckerIpRangesRequest is the request type for the
// GetCheckerIpRanges API operation.
type GetCheckerIpRangesRequest struct {
	*aws.Request
	Input *GetCheckerIpRangesInput
	Copy  func(*GetCheckerIpRangesInput) GetCheckerIpRangesRequest
}

// Send marshals and sends the GetCheckerIpRanges API request.
func (r GetCheckerIpRangesRequest) Send(ctx context.Context) (*GetCheckerIpRangesResponse, error) {
	r.Request.SetContext(ctx)
	err := r.Request.Send()
	if err != nil {
		return nil, err
	}

	resp := &GetCheckerIpRangesResponse{
		GetCheckerIpRangesOutput: r.Request.Data.(*GetCheckerIpRangesOutput),
		response:                 &aws.Response{Request: r.Request},
	}

	return resp, nil
}

// GetCheckerIpRangesResponse is the response type for the
// GetCheckerIpRanges API operation.
type GetCheckerIpRangesResponse struct {
	*GetCheckerIpRangesOutput

	response *aws.Response
}

// SDKResponseMetdata returns the response metadata for the
// GetCheckerIpRanges request.
func (r *GetCheckerIpRangesResponse) SDKResponseMetdata() *aws.Response {
	return r.response
}
