package usecase

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"

	"github.com/tfkhdyt/geminicommit/internal/service"
)

type action string

const (
	confirm    action = "CONFIRM"
	regenerate action = "REGENERATE"
	clue       action = "CLUE"
	edit       action = "EDIT"
	cancel     action = "CANCEL"
)

type RootUsecase struct {
	gitService    *service.GitService
	geminiService *service.GeminiService
}

func NewRootUsecase(
	gitService *service.GitService,
	geminiService *service.GeminiService,
) *RootUsecase {
	return &RootUsecase{gitService, geminiService}
}

func (r *RootUsecase) RootCommand(stageAll *bool, promptAddition *string) error {
	if err := r.gitService.VerifyGitInstallation(); err != nil {
		return err
	}

	if err := r.gitService.VerifyGitRepository(); err != nil {
		return err
	}

	hasHook, _ := r.gitService.HasPreCommitHook()
	if hasHook {
		hookPath, _ := r.gitService.PreCommitHookPath()
		if r.gitService.IsExecutable(hookPath) {
			color.New(color.FgGreen).Println("✔ Running pre-commit hook...")
			err := r.gitService.RunPreCommitHook(hookPath)
			if err != nil {
				color.New(color.FgRed).Printf("Pre-commit hook failed: %v\n", err)
				return err
			}
			color.New(color.FgGreen).Println("✔ Pre-commit hook ran successfully.")
		}
	}
	if *stageAll {
		if err := r.gitService.StageAll(); err != nil {
			return err
		}
	}

	filesChan := make(chan []string, 1)
	diffChan := make(chan string, 1)

	if err := spinner.New().
		Title("Detecting staged files").
		Action(func() {
			files, diff, err := r.gitService.DetectDiffChanges()
			if err != nil {
				filesChan <- []string{}
				diffChan <- ""
				return
			}

			filesChan <- files
			diffChan <- diff
		}).
		Run(); err != nil {
		return err
	}

	underline := color.New(color.Underline)
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F780E2"))
	files, diff := <-filesChan, <-diffChan

	if len(files) == 0 {
		return fmt.Errorf(
			"no staged changes found. stage your changes manually, or automatically stage all changes with the `--all` flag",
		)
	} else if len(files) == 1 {
		underline.Printf("Detected %d staged file:\n", len(files))
	} else {
		underline.Printf("Detected %d staged files:\n", len(files))
	}

	for idx, file := range files {
		color.New(color.Bold).Printf("     %d. %s\n", idx+1, file)
	}

generate:
	for {
		messageChan := make(chan string, 1)
		if err := spinner.New().
			TitleStyle(titleStyle).
			Title("The AI is analyzing your changes").
			Action(func() {
				message, err := r.geminiService.AnalyzeChanges(context.Background(), diff, promptAddition)
				if err != nil {
					messageChan <- ""
					return
				}

				messageChan <- message
			}).
			Run(); err != nil {
			return err
		}

		message := <-messageChan
		fmt.Print("\n")
		underline.Println("Changes analyzed!")

		if strings.TrimSpace(message) == "" {
			return fmt.Errorf("no commit messages were generated. try again")
		}

		color.New(color.Bold).Printf("%s", message)
		fmt.Print("\n\n")

		var selectedAction action
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[action]().
					Title("Use this commit?").
					Options(
						huh.NewOption("Yes", confirm),
						huh.NewOption("Regenerate", regenerate),
						huh.NewOption("Add Clue", clue),
						huh.NewOption("Edit", edit),
						huh.NewOption("Cancel", cancel),
					).
					Value(&selectedAction),
			),
		).Run(); err != nil {
			return err
		}

		switch selectedAction {
		case confirm:
			if err := r.gitService.CommitChanges(message); err != nil {
				return err
			}
			color.New(color.FgGreen).Println("✔ Successfully committed!")
			break generate
		case regenerate:
			continue
		case clue:
			var userClue string
			if err := huh.NewInput().
				Title("Enter your clue for the AI:").
				Value(&userClue).
				Run(); err != nil {
				return err
			}
			if strings.TrimSpace(userClue) != "" {
				promptAddition = &userClue
				fmt.Print("\n")
				color.New(color.Italic).Println("Regenerating with provided clue...")
				fmt.Print("\n")
			} else {
				promptAddition = nil
			}
			continue generate
		case edit:
			for {
				tmpDir := os.TempDir()
				tmpFile, _ := os.CreateTemp(tmpDir, "COMMIT_EDITMSG")
				_ = os.WriteFile(tmpFile.Name(), []byte(message), 0o644)

				editor := os.Getenv("EDITOR")
				cmd := exec.Command(editor, tmpFile.Name())
				cmd.Dir = tmpDir
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr

				if err := cmd.Run(); err != nil {
					return err
				}

				msg, _ := os.ReadFile(tmpFile.Name())
				message = string(msg)
				_ = os.Remove(tmpFile.Name())

				underline.Print("Commit message edited!")
				fmt.Print("\n\n")
				color.New(color.Bold).Printf("%s", message)
				fmt.Print("\n\n")
				var selectedAction action
				if err := huh.NewForm(
					huh.NewGroup(
						huh.NewSelect[action]().
							Title("Use this commit message?").
							Options(
								huh.NewOption("Yes", confirm),
								huh.NewOption("Edit Again", edit),
								huh.NewOption("Regenerate", regenerate),
								huh.NewOption("Cancel", cancel),
							).
							Value(&selectedAction),
					),
				).Run(); err != nil {
					return err
				}

				switch selectedAction {
				case confirm:
					if err := r.gitService.CommitChanges(message); err != nil {
						return err
					}
					color.New(color.FgGreen).Println("✔ Successfully committed!")
					return nil
				case edit:
					continue
				case regenerate:
					continue generate
				case cancel:
					color.New(color.FgRed).Println("Commit cancelled")
					return nil
				}
			}
		case cancel:
			color.New(color.FgRed).Println("Commit cancelled")
			break generate
		}
	}

	return nil
}
