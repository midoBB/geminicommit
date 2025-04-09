/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package model

import (
	"context"
	"fmt"
	"log"
	"os" // Import os package
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func getModelName(oldName string) string {
	lastSlashIndex := strings.LastIndex(oldName, "/")
	// Extract the substring after the last '/'
	if lastSlashIndex != -1 {
		modelName := oldName[lastSlashIndex+1:] // +1 to skip the '/'
		return modelName
	} else {
		return oldName
	}
}

// setCmd represents the set command for the model
var setCmd = &cobra.Command{
	Use:   "set",
	Short: "Set the default Google Gemini model",
	Long:  `Lists available Google Gemini models and allows you to select one as the default.`,
	Run: func(cmd *cobra.Command, args []string) {
		apiKey := viper.GetString("api.key")
		if apiKey == "" {
			// Attempt to get from environment variable if not in config
			apiKey = os.Getenv("GEMINI_API_KEY")
			if apiKey == "" {
				log.Fatal("Google Gemini API key not found. Please set it using 'geminicommit config key set <your-api-key>' or the GEMINI_API_KEY environment variable.")
				return
			}
		}

		ctx := context.Background()
		client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
		if err != nil {
			log.Fatalf("Failed to create genai client: %v", err)
		}
		defer client.Close()

		var modelOptions []huh.Option[string]
		action := func() {
			iter := client.ListModels(ctx)
			for {
				m, iterErr := iter.Next()
				if iterErr == iterator.Done {
					break
				}
				if iterErr != nil {
					log.Fatalf("Failed to list models: %v", iterErr)
				}
				// We only want models that support generateContent
				supported := false
				for _, method := range m.SupportedGenerationMethods {
					if method == "generateContent" {
						supported = true
						break
					}
				}
				if supported {
					// Use model name as both the display value and the stored key
					modelOptions = append(modelOptions, huh.NewOption(fmt.Sprintf("%s (%s)", m.DisplayName, m.Name), m.Name))
				}
			}

			if len(modelOptions) == 0 {
				log.Fatal("No suitable models found.")
				return
			}
		}

		spinner.New().Title("Fetching available models...").Action(action).Run()

		var selectedModel string
		currentModel := viper.GetString("model.default")

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select the default Gemini model").
					Options(modelOptions...).
					Value(&selectedModel).
					Description("The selected model will be used for future operations."),
			),
		).WithTheme(huh.ThemeCatppuccin())

		// Pre-select the current model if it exists in the list
		if currentModel != "" {
			for _, opt := range modelOptions {
				if opt.Key == currentModel {
					selectedModel = currentModel
					break
				}
			}
		}

		err = form.Run()
		if err != nil {
			log.Fatalf("Model selection failed: %v", err)
		}

		if selectedModel != "" {
			viper.Set("model.default", getModelName(selectedModel))
			if err := viper.WriteConfig(); err != nil {
				// Handle case where config file doesn't exist yet
				if os.IsNotExist(err) {
					if configErr := viper.SafeWriteConfig(); configErr != nil {
						log.Fatalf("Error creating config file: %v", configErr)
					}
					fmt.Printf("Set default model to: %s (Config file created)", selectedModel)
				} else {
					log.Fatalf("Error writing config file: %v", err)
				}
			} else {
				fmt.Printf("Set default model to: %s", selectedModel)
			}
		} else {
			fmt.Println("No model selected.")
		}
	},
}

func init() {
}
