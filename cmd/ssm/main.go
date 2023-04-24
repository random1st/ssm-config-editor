package main

import (
	"fmt"
	"os"
	"os/exec"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/spf13/cobra"

	"github.com/random1st/ssm-config-editor/internal/ssmutil"
)

// createSession creates a new AWS session with the specified region.
func createSession(region string) (*session.Session, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	return sess, err
}

// getSSMService creates a new SSM service using the provided session.
func getSSMService(sess *session.Session) *ssm.SSM {
	return ssm.New(sess)
}

// getEditor returns the appropriate system editor.
func getEditor() string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Default to 'vi' if $EDITOR is not set
	}
	return editor
}

// executeEditorCommand opens the specified file using the system editor and returns an error, if any.
func executeEditorCommand(editor, fileName string) error {
	cmd := exec.Command(editor, fileName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	// Define and parse command-line flags
	var region, format, prefix string

	// Create a new 'edit' command
	editCmd := &cobra.Command{
		Use:   "edit <SSM_KEY>",
		Short: "Edit an SSM key",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ssmKey := args[0]

			sess, err := createSession(region)
			if err != nil {
				fmt.Println("Error creating session:", err)
				return
			}
			ssmSvc := getSSMService(sess)

			// Fetch the value of the SSM key
			param, err := ssmSvc.GetParameter(&ssm.GetParameterInput{
				Name:           aws.String(ssmKey),
				WithDecryption: aws.Bool(true),
			})
			if err != nil {
				fmt.Println("Error fetching SSM key value:", err)
				return
			}
			if format == "" {
				format = ssmutil.DetectFormat([]byte(*param.Parameter.Value))
			}

			for {
				// Create a temporary file and write the fetched value to it
				tempFile, err := os.CreateTemp("", "ssm-edit-*")
				if err != nil {
					fmt.Println("Error creating temporary file:", err)
					return
				}
				defer os.Remove(tempFile.Name())
				_, err = tempFile.WriteString(*param.Parameter.Value)
				if err != nil {
					fmt.Println("Error writing to temporary file:", err)
					return
				}
				tempFile.Close()

				// Open the temporary file using the system editor
				editor := getEditor()
				err = executeEditorCommand(editor, tempFile.Name())
				if err != nil {
					fmt.Println("Error running editor:", err)
					return
				}

				// Read the updated content of the temporary file after the editor is closed
				updatedValue, err := os.ReadFile(tempFile.Name())
				if err != nil {
					fmt.Println("Error reading updated file:", err)
					return
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
					return
				}

				// Update the SSM key with the new value
				_, err = ssmSvc.PutParameter(&ssm.PutParameterInput{
					Name:      aws.String(ssmKey),
					Type:      param.Parameter.Type,
					Value:     aws.String(string(updatedValue)),
					Overwrite: aws.Bool(true),
				})
				if err != nil {
					fmt.Println("Error updating SSM key value:", err)
					return
				}

				fmt.Println("SSM key updated successfully")
				break
			}
		},
	}
	// Add flags to the 'edit' command
	editCmd.Flags().StringVar(&region, "region", "us-east-1", "AWS region")
	editCmd.Flags().StringVar(&format, "format", "", "Optional format (json, yaml, env) for validation")

	// Create a new 'list' command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List SSM parameters",
		Run: func(cmd *cobra.Command, args []string) {
			sess, err := createSession(region)
			if err != nil {
				fmt.Println("Error creating session:", err)
				return
			}
			ssmSvc := getSSMService(sess)

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
			err = ssmSvc.DescribeParametersPages(input,
				func(page *ssm.DescribeParametersOutput, lastPage bool) bool {
					parameters = append(parameters, page.Parameters...)
					return !lastPage
				})
			if err != nil {
				fmt.Println("Error fetching parameters:", err)
				return
			}
			// Initialize a tab writer for formatted output
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "Name\tVersion\tLast Modified")

			for _, param := range parameters {
				// Convert the last modified timestamp to a readable date format
				lastModified := param.LastModifiedDate.Format("2006-01-02 15:04:05")

				fmt.Fprintf(w, "%s\t%d\t%s\n", *param.Name, *param.Version, lastModified)
			}

			// Flush the tab writer to print the formatted output
			w.Flush()
		},
	}

	// Add the --prefix flag to the 'list' command
	listCmd.Flags().StringVar(&prefix, "prefix", "", "Optional prefix to filter parameters")
	listCmd.Flags().StringVar(&region, "region", "us-east-1", "AWS region")

	getCmd := &cobra.Command{
		Use:   "get <SSM_KEY>",
		Short: "Get an SSM parameter value",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ssmKey := args[0]

			sess, err := createSession(region)
			if err != nil {
				fmt.Println("Error creating session:", err)
				return
			}
			ssmSvc := getSSMService(sess)

			param, err := ssmSvc.GetParameter(&ssm.GetParameterInput{
				Name:           aws.String(ssmKey),
				WithDecryption: aws.Bool(true),
			})
			if err != nil {
				fmt.Println("Error fetching parameter:", err)
				return
			}

			fmt.Println(*param.Parameter.Value)
		},
	}
	getCmd.Flags().StringVar(&region, "region", "us-east-1", "AWS region")

	// Create a new 'delete' command
	deleteCmd := &cobra.Command{
		Use:   "delete <SSM_KEY>",
		Short: "Delete an SSM parameter",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ssmKey := args[0]

			sess, err := createSession(region)
			if err != nil {
				fmt.Println("Error creating session:", err)
				return
			}
			ssmSvc := getSSMService(sess)

			_, err = ssmSvc.DeleteParameter(&ssm.DeleteParameterInput{
				Name: aws.String(ssmKey),
			})
			if err != nil {
				fmt.Println("Error deleting parameter:", err)
				return
			}

			fmt.Println("Parameter deleted successfully")
		},
	}
	deleteCmd.Flags().StringVar(&region, "region", "us-east-1", "AWS region")

	// Create a new 'create' command
	createCmd := &cobra.Command{
		Use:   "create <SSM_KEY>",
		Short: "Create a new SSM parameter",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ssmKey := args[0]
			from, _ := cmd.Flags().GetString("from")

			sess, err := createSession(region)
			if err != nil {
				fmt.Println("Error creating session:", err)
				return
			}
			ssmSvc := getSSMService(sess)

			var fromValue string
			if from != "" {
				// Fetch the value of the SSM key specified in '--from'
				fromParam, err := ssmSvc.GetParameter(&ssm.GetParameterInput{
					Name:           aws.String(from),
					WithDecryption: aws.Bool(true),
				})
				if err != nil {
					fmt.Println("Error fetching SSM key value from '--from':", err)
					return
				}
				fromValue = *fromParam.Parameter.Value
				if format == "" {
					format = ssmutil.DetectFormat([]byte(*fromParam.Parameter.Value))
				}
			}

			for {
				// Create a temporary file for editing
				tempFile, err := os.CreateTemp("", "ssm-create-*")
				if err != nil {
					fmt.Println("Error creating temporary file:", err)
					return
				}
				defer os.Remove(tempFile.Name())

				// Write the value from '--from' flag if provided
				if fromValue != "" {
					_, err = tempFile.WriteString(fromValue)
					if err != nil {
						fmt.Println("Error writing to temporary file:", err)
						return
					}
				}
				tempFile.Close()

				// Open the temporary file using the system editor
				editor := getEditor()
				err = executeEditorCommand(editor, tempFile.Name())
				if err != nil {
					fmt.Println("Error running editor:", err)
					return
				}

				// Read the content of the temporary file after the editor is closed
				newValue, err := os.ReadFile(tempFile.Name())
				if err != nil {
					fmt.Println("Error reading the temporary file:", err)
					return
				}

				// Validate the updated content based on the specified format
				err = ssmutil.ValidateFormat(newValue, format)
				if err != nil {
					fmt.Println("Error validating created file:", err)
					fmt.Println("Please try again.")
					continue
				}

				_, err = ssmSvc.PutParameter(&ssm.PutParameterInput{
					Name:      aws.String(ssmKey),
					Value:     aws.String(string(newValue)),
					Type:      aws.String("String"),
					Overwrite: aws.Bool(false),
				})
				if err != nil {
					fmt.Println("Error creating SSM parameter:", err)
					return
				}

				fmt.Printf("SSM parameter '%s' created successfully\n", ssmKey)
				break
			}
		},
	}

	// Add the 'format' flag to the 'create' command
	createCmd.Flags().StringVar(&format, "format", "", "Optional format (json, yaml, env) for validation")
	createCmd.Flags().StringVar(&region, "region", "us-east-1", "AWS region")

	// Create a new 'upload' command
	uploadCmd := &cobra.Command{
		Use:   "upload <SSM_KEY> <FILE_PATH>",
		Short: "Upload an SSM parameter value from a file",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			ssmKey := args[0]
			filePath := args[1]

			// Read the content of the file
			fileContent, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Println("Error reading the file:", err)
				return
			}

			sess, err := createSession(region)
			if err != nil {
				fmt.Println("Error creating session:", err)
				return
			}
			ssmSvc := getSSMService(sess)

			// Create or update the SSM parameter with the new value
			_, err = ssmSvc.PutParameter(&ssm.PutParameterInput{
				Name:      aws.String(ssmKey),
				Value:     aws.String(string(fileContent)),
				Type:      aws.String("String"),
				Overwrite: aws.Bool(true),
			})
			if err != nil {
				fmt.Println("Error uploading SSM parameter value:", err)
				return
			}

			fmt.Println("SSM parameter value uploaded successfully")
		},
	}

	// Add the 'region' flag to the 'upload' command
	uploadCmd.Flags().StringVar(&region, "region", "us-east-1", "AWS region")

	// Create a root command and add the 'edit' command as a subcommand
	rootCmd := &cobra.Command{Use: "ssm"}
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(uploadCmd)

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
