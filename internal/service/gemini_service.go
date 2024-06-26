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
				`
You are an AI assistant specialized in generating conventional git commit messages based on provided diff changes. Follow these guidelines:

1. Analyze the following diff changes:
%s

2. Generate a well-formed git commit message based on all the staged file contents.
3. Focus on why the changes were made, providing context and reasoning.
4. Use conventional commit prefixes (feat, fix, docs, style, refactor, perf, test, chore).
5. Define the scope of the changes:
   - If changes are related, use a common scope (e.g., component name, feature area)
   - If changes affect multiple unrelated areas, use "misc" as the scope
6. Do not include emojis or any decorative elements.
7. Consider all changes to:
   - Source files for programming languages
   - Shell configuration files
   - Documentation (README, .md files)
   - Package management files
8. Exclude changes to lock files, sum files, or any generated artifacts.
9. Format:
   - First line: Commit type(scope): Subject summarizing all changes (max 60 characters)
   - Blank line
   - Body: Provide an exhaustive explanation of all changes (wrap at 72 characters)
10. In the body:
    - List each change separately
    - Explain the purpose and impact of each change in detail
    - Include specific file names and paths when relevant
    - Describe any new functionality or behavior changes
    - Mention any potential side effects or areas that might be affected
    - If using "misc" scope, clearly delineate and explain each unrelated change
11. Exclude any unnecessary information or formatting.
12. Do not include any introductory text before the commit message.
13. Do not include any notes, explanations, or comments after the commit message.
14. Provide only the commit message itself, exactly as it should appear in the git commit.
15. Ensure all changes from the diff are represented in the commit message, with detailed explanations for each.

Your entire response will be used directly in a git commit command, so include only the commit message text. Be thorough and detailed in the body of the commit message.
				`,
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
