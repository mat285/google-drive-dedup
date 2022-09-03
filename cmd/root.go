/*

Copyright (c) 2021 - Present. Blend Labs, Inc. All rights reserved
Blend Confidential - Restricted

*/

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// *** root -- all commands derive from this ***

func newRootCmd() *cobra.Command {
	var directory *string
	var delete *bool
	var rootCmd = &cobra.Command{
		Use:     "dedup-drive",
		Example: "dedup-drive",
		Short:   "",
		Long:    "",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return dedupDrive(directory, delete)
		},
	}
	directory = rootCmd.PersistentFlags().StringP("directory", "d", "", "The root of the google drive directory")
	delete = rootCmd.PersistentFlags().Bool("delete", false, "Deletes any duplicate files found")
	return rootCmd
}

// *** functions ***

// Execute executes the root command
func Execute() {
	rootCmd := newRootCmd()
	err := rootCmd.Execute()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
