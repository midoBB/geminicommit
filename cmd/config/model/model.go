/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package model

import (
	"github.com/spf13/cobra"
)

// ModelCmd represents the model command
var ModelCmd = &cobra.Command{
	Use:   "model",
	Short: "Manage Google Gemini model configuration",
	Long:  `Manage the Google Gemini model used by the application.`,
}

func init() {
	ModelCmd.AddCommand(setCmd)
}
