package commands

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
)

func UploadParameterValue(ssmSvc *ssm.SSM, ssmKey string, filePath string) error {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("Error reading the file:", err)
		return err
	}

	// Create or update the SSM parameter with the new value
	_, err = ssmSvc.PutParameter(&ssm.PutParameterInput{
		Name:      aws.String(ssmKey),
		Value:     aws.String(string(fileContent)),
		Type:      aws.String("String"),
		Overwrite: aws.Bool(true),
	})
	if err != nil {
		return err
	}
	return nil
}
