package commands

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
)

func DeleteParameter(ssmSvc *ssm.SSM, ssmKey string) error {
	_, err := ssmSvc.DeleteParameter(&ssm.DeleteParameterInput{
		Name: aws.String(ssmKey),
	})
	return err
}
