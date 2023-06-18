package git

import (
	"errors"
	"fmt"
	"os"
	"path"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

func cloneRepo(dir string, url string, ref string, pk *ssh.PublicKeys) (*GitRepository, error) {
	repo, cloneErr := gogit.PlainClone(dir, false, &gogit.CloneOptions{
		Auth:              pk,
		RemoteName:        "origin",
		URL:               url,
		ReferenceName:     plumbing.NewBranchReferenceName(ref),
		SingleBranch:      true,
		NoCheckout:        false,
		RecurseSubmodules: gogit.NoRecurseSubmodules,
		Progress:          nil,
		Tags:              gogit.NoTags,
	})
	if cloneErr != nil {
		return &GitRepository{repo}, errors.New(fmt.Sprintf("Error cloning in directory \"%s\": %s", dir, cloneErr.Error()))
	}

	fmt.Println(fmt.Sprintf("Cloned branch \"%s\" of repo \"%s\"", ref, url))
	return &GitRepository{repo}, nil
}

func pullRepo(dir string, url string, ref string, pk *ssh.PublicKeys) (*GitRepository, bool, error) {
	repo, gitErr := gogit.PlainOpen(dir)
	if gitErr != nil {
		return &GitRepository{repo}, true, errors.New(fmt.Sprintf("Error accessing repo in directory \"%s\": %s", dir, gitErr.Error()))
	}

	worktree, worktreeErr := repo.Worktree()
	if worktreeErr != nil {
		return &GitRepository{repo}, true, errors.New(fmt.Sprintf("Error accessing worktree in directory \"%s\": %s", dir, worktreeErr.Error()))
	}

	pullErr := worktree.Pull(&gogit.PullOptions{
		Auth:              pk,
		RemoteName:        "origin",
		ReferenceName:     plumbing.NewBranchReferenceName(ref),
		SingleBranch:      true,
		RecurseSubmodules: gogit.NoRecurseSubmodules,
		Progress:          nil,
		Force:             true,
	})
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
*/
func SyncGitRepo(dir string, url string, ref string, sshCred *SshCredentials) (*GitRepository, bool, error) {
	_, err := os.Stat(path.Join(dir, ".git"))
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, false, errors.New(fmt.Sprintf("Error accessing repo directory's .git sub-directory: %s", err.Error()))
		}

		repo, cloneErr := cloneRepo(dir, url, ref, sshCred.Keys)
		return repo, false, cloneErr
	}

	return pullRepo(dir, url, ref, sshCred.Keys)
}