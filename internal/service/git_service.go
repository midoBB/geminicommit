package service

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type GitService struct{}

// HasPreCommitHook checks if .git/hooks/pre-commit exists in the current repository.
func (g *GitService) HasPreCommitHook() (bool, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("not a git repository: %v", err)
	}
	repoRoot := strings.TrimSpace(string(output))
	hookPath := repoRoot + "/.git/hooks/pre-commit"
	if _, err := exec.Command("test", "-f", hookPath).Output(); err != nil {
		return false, nil
	}
	return true, nil
}

// PreCommitHookPath returns the path to the pre-commit hook if it exists, or an empty string otherwise.
func (g *GitService) PreCommitHookPath() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %v", err)
	}
	repoRoot := strings.TrimSpace(string(output))
	hookPath := repoRoot + "/.git/hooks/pre-commit"
	if _, err := exec.Command("test", "-f", hookPath).Output(); err != nil {
		return "", nil
	}
	return hookPath, nil
}

// IsExecutable checks if the given file is executable
func (g *GitService) IsExecutable(path string) bool {
	return exec.Command("test", "-x", path).Run() == nil
}

// RunPreCommitHook executes the pre-commit hook and streams all output to the terminal
func (g *GitService) RunPreCommitHook(path string) error {
	cmd := exec.Command(path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func NewGitService() *GitService {
	return &GitService{}
}

func (g *GitService) VerifyGitInstallation() error {
	if err := exec.Command("git", "--version").Run(); err != nil {
		return fmt.Errorf("git is not installed. %v", err)
	}

	return nil
}

func (g *GitService) VerifyGitRepository() error {
	if err := exec.Command("git", "rev-parse", "--show-toplevel").Run(); err != nil {
		return fmt.Errorf(
			"the current directory must be a git repository. %v",
			err,
		)
	}

	return nil
}

func (g *GitService) StageAll() error {
	if err := exec.Command("git", "add", "-u").Run(); err != nil {
		return fmt.Errorf("failed to update tracked files. %v", err)
	}

	return nil
}

func (g *GitService) DetectDiffChanges() ([]string, string, error) {
	// Default lock files to exclude if none provided
	excludePatterns := DefaultLockFilePatterns()

	// Build git command with exclusion patterns
	fileCmd := []string{"git", "diff", "--cached", "--diff-algorithm=minimal", "--name-only", "--", "."}
	diffCmd := []string{"git", "diff", "--cached", "--diff-algorithm=minimal", "--", "."}

	// Add exclusion patterns to commands
	for _, pattern := range excludePatterns {
		fileCmd = append(fileCmd, fmt.Sprintf(":(exclude)%s", pattern))
		diffCmd = append(diffCmd, fmt.Sprintf(":(exclude)%s", pattern))
	}

	// Execute file list command
	files, err := exec.Command(fileCmd[0], fileCmd[1:]...).Output()
	if err != nil {
		fmt.Println("Error:", err)
		return nil, "", err
	}

	filesStr := strings.TrimSpace(string(files))
	if filesStr == "" {
		return nil, "", fmt.Errorf("nothing to be analyzed")
	}

	// Execute diff content command
	diff, err := exec.Command(diffCmd[0], diffCmd[1:]...).Output()
	if err != nil {
		fmt.Println("Error:", err)
		return nil, "", err
	}

	return strings.Split(filesStr, "\n"), string(diff), nil
}

// DefaultLockFilePatterns returns common lock file patterns to exclude
func DefaultLockFilePatterns() []string {
	return []string{
		"**/package-lock.json",
		"**/yarn.lock",
		"**/Gemfile.lock",
		"**/Cargo.lock",
		"**/go.sum",
		"**/composer.lock",
		"**/poetry.lock",
		"**/Pipfile.lock",
		"**/pnpm-lock.yaml",
	}
}

func (g *GitService) CommitChanges(message string) error {
	output, err := exec.Command("git", "commit", "-m", message).Output()
	if err != nil {
		return fmt.Errorf("failed to commit changes. %v", err)
	}

	fmt.Println(string(output))

	return nil
}
