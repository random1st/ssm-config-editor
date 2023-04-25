package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/spf13/cobra"

	"github.com/random1st/ssm-config-editor/internal/commands"
)

var (
	version   string
	buildDate string
)

func getSSMSvc(region string) (*ssm.SSM, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)

	if err != nil {
		return nil, err
	}
	ssmSvc := ssm.New(sess)

	return ssmSvc, nil
}

func main() {
	var region, format, prefix, from string

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number and build date",
		Long:  "Print the version number and build date",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("ssm-config-editor %s, build date: %s\n", version, buildDate)
		},
	}

	editCmd := &cobra.Command{
		Use:   "edit <SSM_KEY>",
		Short: "Edit an SSM key",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ssmSvc, _ := getSSMSvc(region)
			ssmKey := args[0]

			err := commands.EditParameter(ssmSvc, ssmKey, format)
			if err != nil {
				fmt.Println("Error editing SSM key:", err)
				return
			}

			fmt.Println("SSM key updated successfully")
		},
	}

	editCmd.Flags().StringVar(&region, "region", "us-east-1", "AWS region")
	editCmd.Flags().StringVar(&format, "format", "", "Optional format (json, yaml, env) for validation")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List SSM parameters",
		Run: func(cmd *cobra.Command, args []string) {
			ssmSvc, _ := getSSMSvc(region)
			parameters, err := commands.ListParameters(ssmSvc, prefix)
			if err != nil {
				fmt.Println("Error fetching parameters:", err)
				return
			}

			// Initialize a tab writer for formatted output
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "Name\tVersion\tLast Modified")

			for _, param := range parameters {
				fmt.Fprintf(w, "%s\t%d\t%s\n", param.Name, param.Version, param.LastModifiedDate)
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
			ssmSvc, _ := getSSMSvc(region)
			ssmKey := args[0]

			// Call the refactored GetParameterValue function
			value, err := commands.GetParameterValue(ssmSvc, ssmKey)
			if err != nil {
				fmt.Println("Error fetching parameter:", err)
				return
			}

			fmt.Println(value)
		},
	}

	getCmd.Flags().StringVar(&region, "region", "us-east-1", "AWS region")

	deleteCmd := &cobra.Command{
		Use:   "delete <SSM_KEY>",
		Short: "Delete an SSM parameter",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ssmSvc, _ := getSSMSvc(region)
			ssmKey := args[0]

			err := commands.DeleteParameter(ssmSvc, ssmKey)
			if err != nil {
				fmt.Println("Error deleting parameter:", err)
				return
			}
			fmt.Println("Parameter deleted successfully")
		},
	}
	deleteCmd.Flags().StringVar(&region, "region", "us-east-1", "AWS region")

	createCmd := &cobra.Command{
		Use:   "create <SSM_KEY>",
		Short: "Create a new SSM parameter",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ssmSvc, _ := getSSMSvc(region)

			ssmKey := args[0]
			from, _ := cmd.Flags().GetString("from")

			if err := commands.CreateParameter(ssmSvc, ssmKey, from, format); err != nil {
				fmt.Println("Error creating parameter:", err)
				return
			}
		},
	}

	createCmd.Flags().StringVar(&format, "format", "", "Optional format (json, yaml, env) for validation")
	createCmd.Flags().StringVar(&from, "from", "", "Optional source key for the parameter value")
	createCmd.Flags().StringVar(&region, "region", "us-east-1", "AWS region")

	uploadCmd := &cobra.Command{
		Use:   "upload <SSM_KEY> <FILE_PATH>",
		Short: "Upload an SSM parameter value from a file",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			ssmSvc, _ := getSSMSvc(region)

			ssmKey := args[0]
			filePath := args[1]

			err := commands.UploadParameterValue(ssmSvc, ssmKey, filePath)
			if err != nil {
				fmt.Println("Error uploading SSM parameter value:", err)
				return
			}

			fmt.Println("SSM parameter value uploaded successfully")
		},
	}

	uploadCmd.Flags().StringVar(&region, "region", "us-east-1", "AWS region")

	rootCmd := &cobra.Command{Use: "ssm"}
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(uploadCmd)

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error executing root command:", err)
		os.Exit(1)
	}
}
