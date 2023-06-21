package commands

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"

	"github.com/random1st/ssm-config-editor/internal/ssmutil"
)

func CreateParameter(ssmSvc *ssm.SSM, ssmKey, from, format string) error {
	var fromValue string
	if from != "" {
		// Fetch the value of the SSM key specified in '--from'
		fromParam, err := ssmSvc.GetParameter(&ssm.GetParameterInput{
			Name:           aws.String(from),
			WithDecryption: aws.Bool(true),
		})
		if err != nil {
			return fmt.Errorf("Error fetching SSM key value from '--from': %v", err)
		}
		fromValue = *fromParam.Parameter.Value
		if format == "" {
			fmt.Println("Format not specified, detecting format from '--from' value")
			format = ssmutil.DetectFormat([]byte(*fromParam.Parameter.Value))
		}
	}

	for {
		// Create a temporary file for editing
		tempFile, err := os.CreateTemp("", "ssm-create-*")
		if err != nil {
			return fmt.Errorf("Error creating temporary file: %v", err)
		}
		defer os.Remove(tempFile.Name())

		// Write the value from '--from' flag if provided
		if fromValue != "" {
			_, err = tempFile.WriteString(fromValue)
			if err != nil {
				return fmt.Errorf("Error writing to temporary file: %v", err)
			}
		}
		tempFile.Close()

		// Open the temporary file using the system editor
		editor := getEditor()
		err = executeEditorCommand(editor, tempFile.Name())
		if err != nil {
			return fmt.Errorf("Error running editor: %v", err)
		}

		// Read the content of the temporary file after the editor is closed
		newValue, err := os.ReadFile(tempFile.Name())
		if err != nil {
			return fmt.Errorf("Error reading the temporary file: %v", err)
		}

		// Validate the updated content based on the specified format
		if format != "" {
			err = ssmutil.ValidateFormat(newValue, format)
		}
		if err != nil {
			fmt.Println("Error validating created file:", err)
			fmt.Println("Please try again.")
			continue
		}
		// check parameter size
		tier := ssm.ParameterTierStandard
		if len(newValue) > 4096 {
			tier = ssm.ParameterTierAdvanced
		}

		_, err = ssmSvc.PutParameter(&ssm.PutParameterInput{
			Name:      aws.String(ssmKey),
			Value:     aws.String(string(newValue)),
			Type:      aws.String("String"),
			Overwrite: aws.Bool(false),
			Tier:      aws.String(tier),
		})
		if err != nil {
			return fmt.Errorf("Error creating SSM parameter: %v", err)
		}

		fmt.Printf("SSM parameter '%s' created successfully\n", ssmKey)
		break
	}
	return nil
}
