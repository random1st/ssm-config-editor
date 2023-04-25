package commands

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
)

// ParameterMetadata contains metadata about an SSM parameter
type ParameterMetadata struct {
	Name             string
	Version          int64
	LastModifiedDate string
}

// ListParameters retrieves and returns a list of SSM parameters
func ListParameters(ssmSvc *ssm.SSM, prefix string) ([]ParameterMetadata, error) {
	input := &ssm.DescribeParametersInput{}
	if prefix != "" {
		input.SetParameterFilters([]*ssm.ParameterStringFilter{
			{
				Key:    aws.String("Name"),
				Option: aws.String("BeginsWith"),
				Values: []*string{aws.String(prefix)},
			},
		})
	}

	var parameters []*ssm.ParameterMetadata
	err := ssmSvc.DescribeParametersPages(input,
		func(page *ssm.DescribeParametersOutput, lastPage bool) bool {
			parameters = append(parameters, page.Parameters...)
			return !lastPage
		})
	if err != nil {
		return nil, err
	}

	var result []ParameterMetadata
	for _, param := range parameters {
		lastModified := param.LastModifiedDate.Format("2006-01-02 15:04:05")
		result = append(result, ParameterMetadata{
			Name:             *param.Name,
			Version:          *param.Version,
			LastModifiedDate: lastModified,
		})
	}

	return result, nil
}
