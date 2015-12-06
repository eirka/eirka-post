package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
)

// the format for the lambda thumbnail
type LambdaThumbnail struct {
	Bucket    string `json:"bucket"`
	Filename  string `json:"filename"`
	Thumbnail string `json:"thumbnail"`
	MaxWidth  int    `json:"max_width"`
	MaxHeight int    `json:"max_height"`
}

// the format of the lambda context response
type LambdaResponse struct {
	Success string `json:"successMessage"`
	Error   string `json:"errorMessage"`
	Width   int    `json:"thumbWidth"`
	Height  int    `json:"thumbHeight"`
}

// posts to our api endpoint
func (t *LambdaThumbnail) Execute() (width, height int, err error) {

	// Marshal the structs into JSON
	output, err := json.Marshal(t)
	if err != nil {
		return
	}

	session, err := AWSSession()
	if err != nil {
		return
	}

	// params for lambda invocation
	params := &lambda.InvokeInput{
		FunctionName:   aws.String("resize_image"),
		InvocationType: lambda.InvocationTypeRequestResponse,
		Payload:        output,
	}

	// invoke lambda function
	resp, err := session.Invoke(params)
	if err != nil {
		return
	}

	if resp.FunctionError != "" {
		err = errors.New(fmt.Sprintf("Error creating thumbnail: %s", resp.FunctionError))
		return
	}

	response := LambdaResponse{}

	err = json.Unmarshal(resp.Payload, &response)
	if err != nil {
		return
	}

	// return our values
	width = response.Width
	height = response.Height

	return

}
