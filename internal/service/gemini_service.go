package service

import (
	"context"
	"fmt"
	"strings"

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
	deletedFiles []string,
	promptAddition *string,
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
	defaultModel := viper.GetString("model.default")
	if defaultModel == "" {
		defaultModel = "gemini-2.0-flash-exp"
	}
	model := client.GenerativeModel(defaultModel)
	safetySettings := []*genai.SafetySetting{
		{
			Category:  genai.HarmCategoryHarassment,
			Threshold: genai.HarmBlockNone,
		},
		{
			Category:  genai.HarmCategoryHateSpeech,
			Threshold: genai.HarmBlockNone,
		},
		{
			Category:  genai.HarmCategoryDangerousContent,
			Threshold: genai.HarmBlockNone,
		},
		{
			Category:  genai.HarmCategorySexuallyExplicit,
			Threshold: genai.HarmBlockNone,
		},
	}
	model.SafetySettings = safetySettings
	var injection string
	if promptAddition == nil {
		injection = ""
	} else {
		injection = fmt.Sprintf("with additional focus on %s", *promptAddition)
	}

	var deletedFilesInfo string
	if len(deletedFiles) > 0 {
		deletedFilesInfo = fmt.Sprintf("\n\nDeleted files:\n%s", strings.Join(deletedFiles, "\n"))
	} else {
		deletedFilesInfo = ""
	}

	resp, err := model.GenerateContent(
		ctx,
		genai.Text(
			fmt.Sprintf(
				`
You are an AI assistant specialized in generating conventional git commit messages based on provided diff changes. Follow these guidelines:

1. Analyze the following diff changes %s
%s%s

2. Generate a well-formed git commit message based on all the staged file contents (except the package configuration files (go.mod/package.json/cargo.toml/etc...)).
3. Be concise and direct
4. Focus on why the changes were made, providing context and reasoning.
5. Use conventional commit prefixes (feat, fix, docs, style, refactor, perf, test, chore).
6. Define the scope of the changes:
   - If changes are related, use a common scope (e.g., component name, feature area)
   - If changes affect multiple unrelated areas, use "misc" as the scope
7. Do not include emojis or any decorative elements.
8. Consider all changes to:
   - Source files for programming languages
   - Shell configuration files
   - Documentation (README, .md files)
   - Package management files
   - Deleted files (if any are listed separately)
9. Exclude changes to lock files, sum files, or any generated artifacts.
10. Format:
   - First line: Commit type(scope): Subject summarizing all changes (max 60 characters)
   - Blank line
   - Body: Provide an exhaustive explanation of all changes (wrap at 72 characters)
11. In the body:
    - List each change separately
    - Explain the purpose and impact of each change in detail
    - Include specific file names and paths when relevant
    - Describe any new functionality or behavior changes
    - Mention any potential side effects or areas that might be affected
    - If using "misc" scope, clearly delineate and explain each unrelated change
12. Exclude any unnecessary information or formatting.
13. Do not include any introductory text before the commit message.
14. Do not include any notes, explanations, or comments after the commit message.
15. Provide only the commit message itself, exactly as it should appear in the git commit.
16. Ensure all changes from the diff are represented in the commit message, with detailed explanations for each.

Your entire response will be used directly in a git commit command, so include only the commit message text. NEVER USE markdown formatting. Be thorough and detailed in the body of the commit message.
				`,
				injection,
				diff,
				deletedFilesInfo,
			),
		),
	)
	if err != nil {
		fmt.Println("Error:", err)
		return "", nil
	}

	return fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]), nil
}
