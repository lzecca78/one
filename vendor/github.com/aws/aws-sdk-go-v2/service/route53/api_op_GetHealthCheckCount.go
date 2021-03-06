// Code generated by private/model/cli/gen-api/main.go. DO NOT EDIT.

package route53

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/internal/awsutil"
	"github.com/aws/aws-sdk-go-v2/private/protocol"
)

// A request for the number of health checks that are associated with the current
// AWS account.
// Please also see https://docs.aws.amazon.com/goto/WebAPI/route53-2013-04-01/GetHealthCheckCountRequest
type GetHealthCheckCountInput struct {
	_ struct{} `type:"structure"`
}

// String returns the string representation
func (s GetHealthCheckCountInput) String() string {
	return awsutil.Prettify(s)
}

// MarshalFields encodes the AWS API shape using the passed in protocol encoder.
func (s GetHealthCheckCountInput) MarshalFields(e protocol.FieldEncoder) error {

	return nil
}

// A complex type that contains the response to a GetHealthCheckCount request.
// Please also see https://docs.aws.amazon.com/goto/WebAPI/route53-2013-04-01/GetHealthCheckCountResponse
type GetHealthCheckCountOutput struct {
	_ struct{} `type:"structure"`

	// The number of health checks associated with the current AWS account.
	//
	// HealthCheckCount is a required field
	HealthCheckCount *int64 `type:"long" required:"true"`
}

// String returns the string representation
func (s GetHealthCheckCountOutput) String() string {
	return awsutil.Prettify(s)
}

// MarshalFields encodes the AWS API shape using the passed in protocol encoder.
func (s GetHealthCheckCountOutput) MarshalFields(e protocol.FieldEncoder) error {
	if s.HealthCheckCount != nil {
		v := *s.HealthCheckCount

		metadata := protocol.Metadata{}
		e.SetValue(protocol.BodyTarget, "HealthCheckCount", protocol.Int64Value(v), metadata)
	}
	return nil
}

const opGetHealthCheckCount = "GetHealthCheckCount"

// GetHealthCheckCountRequest returns a request value for making API operation for
// Amazon Route 53.
//
// Retrieves the number of health checks that are associated with the current
// AWS account.
//
//    // Example sending a request using GetHealthCheckCountRequest.
//    req := client.GetHealthCheckCountRequest(params)
//    resp, err := req.Send(context.TODO())
//    if err == nil {
//        fmt.Println(resp)
//    }
//
// Please also see https://docs.aws.amazon.com/goto/WebAPI/route53-2013-04-01/GetHealthCheckCount
func (c *Client) GetHealthCheckCountRequest(input *GetHealthCheckCountInput) GetHealthCheckCountRequest {
	op := &aws.Operation{
		Name:       opGetHealthCheckCount,
		HTTPMethod: "GET",
		HTTPPath:   "/2013-04-01/healthcheckcount",
	}

	if input == nil {
		input = &GetHealthCheckCountInput{}
	}

	req := c.newRequest(op, input, &GetHealthCheckCountOutput{})
	return GetHealthCheckCountRequest{Request: req, Input: input, Copy: c.GetHealthCheckCountRequest}
}

// GetHealthCheckCountRequest is the request type for the
// GetHealthCheckCount API operation.
type GetHealthCheckCountRequest struct {
	*aws.Request
	Input *GetHealthCheckCountInput
	Copy  func(*GetHealthCheckCountInput) GetHealthCheckCountRequest
}

// Send marshals and sends the GetHealthCheckCount API request.
func (r GetHealthCheckCountRequest) Send(ctx context.Context) (*GetHealthCheckCountResponse, error) {
	r.Request.SetContext(ctx)
	err := r.Request.Send()
	if err != nil {
		return nil, err
	}

	resp := &GetHealthCheckCountResponse{
		GetHealthCheckCountOutput: r.Request.Data.(*GetHealthCheckCountOutput),
		response:                  &aws.Response{Request: r.Request},
	}

	return resp, nil
}

// GetHealthCheckCountResponse is the response type for the
// GetHealthCheckCount API operation.
type GetHealthCheckCountResponse struct {
	*GetHealthCheckCountOutput

	response *aws.Response
}

// SDKResponseMetdata returns the response metadata for the
// GetHealthCheckCount request.
func (r *GetHealthCheckCountResponse) SDKResponseMetdata() *aws.Response {
	return r.response
}
