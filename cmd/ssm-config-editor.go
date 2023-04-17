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

			sess, err := session.NewSession(&aws.Config{
				Region: aws.String(region),
			})
			if err != nil {
				fmt.Println("Error creating session:", err)
				return
			}
			ssmSvc := ssm.New(sess)

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
				editor := os.Getenv("EDITOR")
				if editor == "" {
					editor = "vi" // Default to 'vi' if $EDITOR is not set
				}
				cmd := exec.Command(editor, tempFile.Name())
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err = cmd.Run()
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
			sess, err := session.NewSession(&aws.Config{
				Region: aws.String(region),
			})
			if err != nil {
				fmt.Println("Error creating session:", err)
				return
			}
			ssmSvc := ssm.New(sess)

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

	getCmd := &cobra.Command{
		Use:   "get <SSM_KEY>",
		Short: "Get an SSM parameter value",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ssmKey := args[0]

			sess, err := session.NewSession(&aws.Config{
				Region: aws.String(region),
			})
			if err != nil {
				fmt.Println("Error creating session:", err)
				return
			}
			ssmSvc := ssm.New(sess)

			param, err := ssmSvc.GetParameter(&ssm.GetParameterInput{
				Name:           aws.String(ssmKey),
				WithDecryption: aws.Bool(true),
			})
			if err != nil {
				fmt.Println("Error fetching parameter:", err)
				return
			}

			fmt.Println("Parameter value:", *param.Parameter.Value)
		},
	}

	// Create a new 'delete' command
	deleteCmd := &cobra.Command{
		Use:   "delete <SSM_KEY>",
		Short: "Delete an SSM parameter",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ssmKey := args[0]

			sess, err := session.NewSession(&aws.Config{
				Region: aws.String(region),
			})
			if err != nil {
				fmt.Println("Error creating session:", err)
				return
			}
			ssmSvc := ssm.New(sess)

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

	// Create a new 'create' command
	createCmd := &cobra.Command{
		Use:   "create <SSM_KEY>",
		Short: "Create a new SSM parameter",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ssmKey := args[0]
			from, _ := cmd.Flags().GetString("from")

			sess, err := session.NewSession(&aws.Config{
				Region: aws.String(region),
			})
			if err != nil {
				fmt.Println("Error creating session:", err)
				return
			}
			ssmSvc := ssm.New(sess)

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
				editor := os.Getenv("EDITOR")
				if editor == "" {
					editor = "vi" // Default to 'vi' if $EDITOR is not setgo
				}
				cmd := exec.Command(editor, tempFile.Name())
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err = cmd.Run()
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

	// Create a root command and add the 'edit' command as a subcommand
	rootCmd := &cobra.Command{Use: "ssm"}
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(deleteCmd)

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
