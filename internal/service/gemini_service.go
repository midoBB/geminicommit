package service

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/viper"
	"google.golang.org/api/option"
)

type GeminiService struct{}

func NewGeminiService() *GeminiService {
	return &GeminiService{}
}

func (g *GeminiService) AnalyzeChanges(
	ctx context.Context,
	diff string,
) (string, error) {
	client, err := genai.NewClient(
		ctx,
		option.WithAPIKey(viper.GetString("api.key")),
	)
	if err != nil {
		fmt.Println("Error:", err)
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-pro")
	resp, err := model.GenerateContent(
		ctx,
		genai.Text(
			fmt.Sprintf(
				`let's assume you're an automated AI that will generate a conventional git commit message based on this diff changes:
%s

Create well formed git commit message based of off the currently staged file
contents. The message should convey why something was changed and not what
changed. Use the well known format that has the prefix chore, fix, etc. Never
add in some emojis just for fun.

Only include changes to source files for the programming languages, shell configurations files, documentation such as readme and other .mds, and any changes to package management file. Exclude any lock or sum files.

Do not use markdown format for the output.

For the first line of the commit message, this must be constrained to 60 characters as a maximum and use additional lines for any further context.
Exclude anything unnecessary, because your entire response will be passed directly into git commit`,
				diff,
			),
		),
	)
	if err != nil {
		fmt.Println("Error:", err)
		return "", nil
	}

	return fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]), nil
}
