package commands

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"

	"github.com/random1st/ssm-config-editor/internal/ssmutil"
)

func EditParameter(ssmSvc *ssm.SSM, ssmKey, format string) error {
	// Fetch the value of the SSM key
	param, err := ssmSvc.GetParameter(&ssm.GetParameterInput{
		Name:           aws.String(ssmKey),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return err
	}
	if format == "" {
		format = ssmutil.DetectFormat([]byte(*param.Parameter.Value))
	}

	for {
		// Create a temporary file and write the fetched value to it
		tempFile, err := os.CreateTemp("", "ssm-edit-*")
		if err != nil {
			return err
		}
		defer os.Remove(tempFile.Name())
		_, err = tempFile.WriteString(*param.Parameter.Value)
		if err != nil {
			return err
		}
		tempFile.Close()

		// Open the temporary file using the system editor
		editor := getEditor()
		err = executeEditorCommand(editor, tempFile.Name())
		if err != nil {
			return err
		}

		// Read the updated content of the temporary file after the editor is closed
		updatedValue, err := os.ReadFile(tempFile.Name())
		if err != nil {
			return err
		}

		// Validate the updated content based on the specified format
		if err := ssmutil.ValidateFormat(updatedValue, format); err != nil {
			fmt.Println("Error validating updated file:", err)
			fmt.Println("Please try again.")
			continue
		}

		// Check if the value has changed
		if *param.Parameter.Value == string(updatedValue) {
			fmt.Println("No changes detected, SSM key not updated.")
			return nil
		}

		// Update the SSM key with the new value
		_, err = ssmSvc.PutParameter(&ssm.PutParameterInput{
			Name:      aws.String(ssmKey),
			Type:      param.Parameter.Type,
			Value:     aws.String(string(updatedValue)),
			Overwrite: aws.Bool(true),
		})
		if err != nil {
			return err
		}

		fmt.Println("SSM key updated successfully")
		break
	}
	return nil
}
