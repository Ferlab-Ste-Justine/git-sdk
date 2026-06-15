package git

import (
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"os"
	"path"
	"strings"
)

func injectHttpsCredsInUrl(rawURL string, creds *HttpsCredentials) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid repo URL %q: %w", rawURL, err)
	}

	u.User = url.UserPassword(creds.Auth.Username, creds.Auth.Password)
	
	return u.String(), nil
}

func sanitizePasswordFromOutput(output string, password string) string {
	return strings.ReplaceAll(output, password, "***")
}

func cloneRepoExec(dir string, rawURL string, ref string, creds *HttpsCredentials) (*GitRepository, error) {
	authURL, err := injectHttpsCredsInUrl(rawURL, creds)
	if err != nil {
		return nil, err
	}

	out, execErr := exec.Command("git", "clone", "--single-branch", "--branch", ref, authURL, dir).CombinedOutput()
	if execErr != nil {
		return nil, errors.New(fmt.Sprintf("Error cloning in directory \"%s\": %s", dir, sanitizePasswordFromOutput(string(out), creds.Auth.Password)))
	}

	fmt.Println(fmt.Sprintf("Cloned branch \"%s\" of repo \"%s\"", ref, rawURL))
	return GetGitRepo(dir)
}

func pullRepoExec(dir string, rawURL string, ref string, creds *HttpsCredentials) (*GitRepository, bool, error) {
	authURL, err := injectHttpsCredsInUrl(rawURL, creds)
	if err != nil {
		return nil, false, err
	}

	fetchOut, fetchErr := exec.Command(
		"git", "-C", dir, "fetch", authURL,
		fmt.Sprintf("refs/heads/%s", ref),
	).CombinedOutput()
	if fetchErr != nil {
		return nil, false, errors.New(fmt.Sprintf(
			"Error pulling latest changes in directory \"%s\": %s",
			dir, sanitizePasswordFromOutput(string(fetchOut), creds.Auth.Password),
		))
	}

	resetOut, resetErr := exec.Command("git", "-C", dir, "reset", "--hard", "FETCH_HEAD").CombinedOutput()
	if resetErr != nil {
		return nil, true, errors.New(fmt.Sprintf(
			"Error resetting in directory \"%s\": %s", dir, string(resetOut),
		))
	}

	repo, getErr := GetGitRepo(dir)
	if getErr != nil {
		return nil, true, getErr
	}

	head, headErr := repo.Repo.Head()
	if headErr != nil {
		return repo, true, errors.New(fmt.Sprintf("Error accessing top commit in directory \"%s\": %s", dir, headErr.Error()))
	}

	fmt.Println(fmt.Sprintf("Branch \"%s\" of repo \"%s\" is at commit %s", ref, rawURL, head.Hash()))
	return repo, false, nil
}

/*
Clone or pull the given reference of a given repo at a given path on the filesystem.
If the repo was previously cloned at the path, a pull will be done, else a clone.
This call runs an exec on a pre-existing git binary on the system. Useful for Azure Devops with https which is not compatible with go-git.
The gitCred argument can be nil for an unauthenticated clone on https or else https. Ssh is currently not supported for the exec version.
*/
func SyncGitRepoExec(dir string, url string, ref string, gitCred *GitCredentials) (*GitRepository, bool, error) {
	if !strings.HasPrefix(url, "http") {
		return nil, false, errors.New("The git exec version of the git-sdk library currently only supports git over http(s)")
	}

	_, err := os.Stat(path.Join(dir, ".git"))
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, false, errors.New(fmt.Sprintf("Error accessing repo directory's .git sub-directory: %s", err.Error()))
		}

		repo, cloneErr := cloneRepoExec(dir, url, ref, gitCred.Https)
		return repo, false, cloneErr
	}

	return pullRepoExec(dir, url, ref, gitCred.Https)
}