package git

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func injectHttpsCreds(rawURL string, creds *HttpsCredentials) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid repo URL %q: %w", rawURL, err)
	}
	u.User = url.UserPassword(creds.Auth.Username, creds.Auth.Password)
	return u.String(), nil
}

func sanitizeCredURL(output string, password string) string {
	if password == "" {
		return output
	}
	return strings.ReplaceAll(output, password, "***")
}

func cloneRepoExec(dir string, rawURL string, ref string, creds *HttpsCredentials) (*GitRepository, error) {
	authURL, err := injectHttpsCreds(rawURL, creds)
	if err != nil {
		return nil, err
	}
	out, execErr := exec.Command("git", "clone", "--single-branch", "--branch", ref, authURL, dir).CombinedOutput()
	if execErr != nil {
		return nil, errors.New(fmt.Sprintf("Error cloning in directory \"%s\": %s", dir, sanitizeCredURL(string(out), creds.Auth.Password)))
	}
	fmt.Println(fmt.Sprintf("Cloned branch \"%s\" of repo \"%s\"", ref, rawURL))
	return GetGitRepo(dir)
}

func pullRepoExec(dir string, rawURL string, ref string, creds *HttpsCredentials) (*GitRepository, bool, error) {
	authURL, err := injectHttpsCreds(rawURL, creds)
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
			dir, sanitizeCredURL(string(fetchOut), creds.Auth.Password),
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

func cloneRepo(dir string, url string, ref string, gitCred *GitCredentials) (*GitRepository, error) {
	if gitCred != nil && gitCred.Https != nil {
		return cloneRepoExec(dir, url, ref, gitCred.Https)
	}

	opts := gogit.CloneOptions{
		RemoteName:        "origin",
		URL:               url,
		ReferenceName:     plumbing.NewBranchReferenceName(ref),
		SingleBranch:      true,
		NoCheckout:        false,
		RecurseSubmodules: gogit.NoRecurseSubmodules,
		Progress:          nil,
		Tags:              gogit.NoTags,
	}

	if gitCred != nil && gitCred.HasAuthMethod() {
		authMethod, authMethodErr := gitCred.GetAuthMethod(url)
		if authMethodErr != nil {
			return nil, authMethodErr
		}

		opts.Auth = authMethod
	}

	repo, cloneErr := gogit.PlainClone(dir, false, &opts)
	if cloneErr != nil {
		return &GitRepository{repo}, errors.New(fmt.Sprintf("Error cloning in directory \"%s\": %s", dir, cloneErr.Error()))
	}

	fmt.Println(fmt.Sprintf("Cloned branch \"%s\" of repo \"%s\"", ref, url))
	return &GitRepository{repo}, nil
}

func pullRepo(dir string, url string, ref string, gitCred *GitCredentials) (*GitRepository, bool, error) {
	if gitCred != nil && gitCred.Https != nil {
		return pullRepoExec(dir, url, ref, gitCred.Https)
	}

	repo, gitErr := gogit.PlainOpen(dir)
	if gitErr != nil {
		return &GitRepository{repo}, true, errors.New(fmt.Sprintf("Error accessing repo in directory \"%s\": %s", dir, gitErr.Error()))
	}

	worktree, worktreeErr := repo.Worktree()
	if worktreeErr != nil {
		return &GitRepository{repo}, true, errors.New(fmt.Sprintf("Error accessing worktree in directory \"%s\": %s", dir, worktreeErr.Error()))
	}

	opts := gogit.PullOptions{
		RemoteName:        "origin",
		ReferenceName:     plumbing.NewBranchReferenceName(ref),
		SingleBranch:      true,
		RecurseSubmodules: gogit.NoRecurseSubmodules,
		Progress:          nil,
		Force:             true,
	}

	if gitCred != nil && gitCred.HasAuthMethod() {
		authMethod, authMethodErr := gitCred.GetAuthMethod(url)
		if authMethodErr != nil {
			return &GitRepository{repo}, true, authMethodErr
		}

		opts.Auth = authMethod
	}

	pullErr := worktree.Pull(&opts)
	if pullErr != nil && pullErr.Error() != gogit.NoErrAlreadyUpToDate.Error() {
		fastForwardProblems := pullErr.Error() == gogit.ErrNonFastForwardUpdate.Error()
		return &GitRepository{repo}, fastForwardProblems, errors.New(fmt.Sprintf("Error pulling latest changes in directory \"%s\": %s", dir, pullErr.Error()))
	}

	if pullErr != nil && pullErr.Error() == gogit.NoErrAlreadyUpToDate.Error() {
		fmt.Println(fmt.Sprintf("Branch \"%s\" of repo \"%s\" is up-to-date", ref, url))
	} else {
		head, headErr := repo.Head()
		if headErr != nil {
			return &GitRepository{repo}, true, errors.New(fmt.Sprintf("Error accessing top commit in directory \"%s\": %s", dir, headErr.Error()))
		}
		fmt.Println(fmt.Sprintf("Branch \"%s\" of repo \"%s\" was updated to commit %s", ref, url, head.Hash()))
	}

	return &GitRepository{repo}, false, nil
}

/*
Clone or pull the given reference of a given repo at a given path on the filesystem.
If the repo was previously cloned at the path, a pull will be done, else a clone.
The sshCred argument can be nil for an unauthenticated clone on https
*/
func SyncGitRepo(dir string, url string, ref string, gitCred *GitCredentials) (*GitRepository, bool, error) {
	_, err := os.Stat(path.Join(dir, ".git"))
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, false, errors.New(fmt.Sprintf("Error accessing repo directory's .git sub-directory: %s", err.Error()))
		}

		repo, cloneErr := cloneRepo(dir, url, ref, gitCred)
		return repo, false, cloneErr
	}

	return pullRepo(dir, url, ref, gitCred)
}

/*
Get a *GitRepository resource from a repository that has already been cloned on the filesystem
*/
func GetGitRepo(dir string) (*GitRepository, error) {
	repo, err := gogit.PlainOpen(dir)
	if err != nil {
		return nil, err
	}

	return &GitRepository{repo}, nil
}
