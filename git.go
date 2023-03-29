package git

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	gogitconf "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

func getPublicKeys(sshKeyPath string, knownHostsPath string) (*ssh.PublicKeys, error) {
	_, statErr := os.Stat(sshKeyPath)
	if statErr != nil {
		return nil, errors.New(fmt.Sprintf("Failed to access ssh key file %s: %s", sshKeyPath, statErr.Error()))
	}

	publicKeys, pkGenErr := ssh.NewPublicKeysFromFile("git", sshKeyPath, "")
	if pkGenErr != nil {
		return nil, errors.New(fmt.Sprintf("Failed to generate public key: %s", pkGenErr.Error()))
	}

	_, statErr = os.Stat(knownHostsPath)
	if statErr != nil {
		return nil, errors.New(fmt.Sprintf("Failed to access known hosts file %s: %s", knownHostsPath, statErr.Error()))
	}
	
	callback, knowHostsErr := ssh.NewKnownHostsCallback(knownHostsPath)
	if knowHostsErr != nil {
		return nil, errors.New(fmt.Sprintf("Failed to parse known hosts file %s: %s", knownHostsPath, knowHostsErr.Error()))
	}

	cliConf, cliConfErr := publicKeys.ClientConfig()
	if cliConfErr != nil {
		return nil, errors.New(fmt.Sprintf("Failed to access ssh key config %s: %s", sshKeyPath, cliConfErr.Error()))
	}
	cliConf.HostKeyCallback = callback

	return publicKeys, nil
}

func cloneRepo(dir string, url string, ref string, pk *ssh.PublicKeys) (*gogit.Repository, error) {
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
		return repo, errors.New(fmt.Sprintf("Error cloning in directory \"%s\": %s", dir, cloneErr.Error()))
	}

	fmt.Println(fmt.Sprintf("Cloned branch \"%s\" of repo \"%s\"", ref, url))
	return repo, nil
}

func pullRepo(dir string, url string, ref string, pk *ssh.PublicKeys) (*gogit.Repository, bool, error) {
	repo, gitErr := gogit.PlainOpen(dir)
	if gitErr != nil {
		return repo, true, errors.New(fmt.Sprintf("Error accessing repo in directory \"%s\": %s", dir, gitErr.Error()))
	}

	worktree, worktreeErr := repo.Worktree()
	if worktreeErr != nil {
		return repo, true, errors.New(fmt.Sprintf("Error accessing worktree in directory \"%s\": %s", dir, worktreeErr.Error()))
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
		return repo, fastForwardProblems, errors.New(fmt.Sprintf("Error pulling latest changes in directory \"%s\": %s", dir, pullErr.Error()))
	}
	
	if pullErr != nil && pullErr.Error() == gogit.NoErrAlreadyUpToDate.Error() {
		fmt.Println(fmt.Sprintf("Branch \"%s\" of repo \"%s\" is up-to-date", ref, url))
	} else {
		head, headErr := repo.Head()
		if headErr != nil {
			return repo, true, errors.New(fmt.Sprintf("Error accessing top commit in directory \"%s\": %s", dir, headErr.Error()))
		}
		fmt.Println(fmt.Sprintf("Branch \"%s\" of repo \"%s\" was updated to commit %s", ref, url, head.Hash()))
	}

	return repo, false, nil
}

/*
Clone or pull the given reference of a given repo at a given path.
If the repo was previously cloned at the path, a pull will be done, else a clone.
*/
func SyncGitRepo(dir string, url string, ref string, sshKeyPath string, knownHostsPath string) (*gogit.Repository, bool, error) {
	pk, pkErr := getPublicKeys(sshKeyPath, knownHostsPath)
	if pkErr != nil {
		return nil, false, pkErr
	}

	_, err := os.Stat(path.Join(dir, ".git"))
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, false, errors.New(fmt.Sprintf("Error accessing repo directory's .git sub-directory: %s", err.Error()))
		}

		repo, cloneErr := cloneRepo(dir, url, ref, pk)
		return repo, false, cloneErr
	}

	return pullRepo(dir, url, ref, pk)
}

/*
Verifies that the top commit of a given git repository was signed by one of the keys that are passed in the argument. 
Returns an error if it isn't.
*/
func VerifyTopCommit(repo *gogit.Repository, armoredKeyrings []string) error {
	head, headErr := repo.Head()
	if headErr != nil {
		return errors.New(fmt.Sprintf("Error accessing repo head: %s", headErr.Error()))
	}

	commit, commitErr := repo.CommitObject(head.Hash())
	if commitErr != nil {
		return errors.New(fmt.Sprintf("Error accessing repo top commit: %s", commitErr.Error()))
	}

	for _, armoredKeyring := range armoredKeyrings {
		entity, err := commit.Verify(armoredKeyring)
		if err == nil {
			for _, identity := range entity.Identities {
				fmt.Println(fmt.Sprintf("Validated top commit \"%s\" is signed by user \"%s\"", head.Hash(), (*identity).Name))
			}
			return nil
		}
	}

	return errors.New(fmt.Sprintf("Top commit \"%s\" isn't signed with any of the trusted keys", head.Hash()))
}

/*
Optional parameters to pass to the CommitFiles command
*/
type CommitOptions struct {
	//Name of the commiter
	Name           string
	//Email of the commiter
	Email          string
	//Path of gpg armored private key to sign commit with
	SignKeyPath    string
	//Path of optional passphrase if the gpg armored private key is encrypted
	PassphrasePath string
}

func getSignKey(signKeyPath string, passphrasePath string) (*openpgp.Entity, error) {
	signKey, readSignKeyErr := os.ReadFile(signKeyPath)
	if readSignKeyErr != nil {
		return nil, errors.New(fmt.Sprintf("Error reading signing key: %s", readSignKeyErr.Error()))
	}
	
	signBlock, decErr := armor.Decode(strings.NewReader(string(signKey)))
	if decErr != nil {
		return nil, errors.New(fmt.Sprintf("Error decoding signing key: %s", decErr.Error()))
	}

	if signBlock.Type != openpgp.PrivateKeyType {
		return nil, errors.New("Signing key is not a gpg private key.")
	}

	signReader := packet.NewReader(signBlock.Body)
	signEntity, readErr := openpgp.ReadEntity(signReader)
	if readErr != nil {
		return nil, errors.New(fmt.Sprintf("Error parsing signing key: %s", readErr.Error()))
	}

	if signEntity.PrivateKey.Encrypted {
		if passphrasePath == "" {
			return nil, errors.New("Signing key is encrypted and no passphrase was passed to decrypt it.")
		}

		passphrase, readPassphraseErr := os.ReadFile(passphrasePath)
		if readPassphraseErr != nil {
			return nil, errors.New(fmt.Sprintf("Error reading passphrase: %s", readPassphraseErr.Error()))
		}

		decrErr := signEntity.PrivateKey.Decrypt(passphrase)
		if decrErr != nil {
			return nil, errors.New(fmt.Sprintf("Error decrypting signing key with passphrase: %s", decrErr.Error()))
		}
	}

	return signEntity, nil
} 

/*
Commits the given list of files in the git repository.
If not changes are detected in the files provided, a commit will not be attempted.
*/
func CommitFiles(repo *gogit.Repository, files []string, msg string, opts CommitOptions) (bool, error) {
	w, wErr := repo.Worktree()
	if wErr != nil {
		return false, errors.New(fmt.Sprintf("Error accessing repo worktree: %s", wErr.Error()))
	}

	for _, file := range files {
		_, addErr := w.Add(file)
		if addErr != nil {
			return false, errors.New(fmt.Sprintf("Error staging file %s for commit: %s", file, addErr.Error()))
		}
	}

	stat, statErr := w.Status()
	if statErr != nil {
		return false, errors.New(fmt.Sprintf("Error getting repo status after staging files: %s", statErr.Error()))
	}

	if len(stat) == 0 {
		fmt.Println("Will not commit as there are no changes to commit.")
		return false, nil
	}

	comOpts := gogit.CommitOptions{}
	if opts.Name != "" || opts.Email != "" {
		comOpts.Author = &object.Signature{
			Name: opts.Name,
			Email: opts.Email,
			When: time.Now(),
		}
	}

	if opts.SignKeyPath != "" {
		signEntity, signEntErr := getSignKey(opts.SignKeyPath, opts.PassphrasePath)
		if signEntErr != nil {
			return false, signEntErr
		}
		
		comOpts.SignKey = signEntity
	}

	_, commErr := w.Commit(msg, &comOpts)
	if commErr != nil {
		return false, errors.New(fmt.Sprintf("Error commiting file changes: %s", commErr.Error()))
	}

	fmt.Printf("Committed following changes with message \"%s\": \n%s\n", msg, stat.String())

	return true, nil
}

/*
Function signature meant to be passed as an argument to the PushChanges function.
It should return a git repository with changes to push if there are changes to push otherwise it should return nil.
The function should be idempotent as it might be called repeatedly if there is a conflict during the push
*/
type PushPreHook func() (*gogit.Repository, error)

/*
Takes a function argument that should return a git repository with changes to push if there are (and nil otherwise).
From there, it will try to push the new commits in the repository to the given reference on origin.
If there are conflicts during the push, it will keep retrying by re-invoking its function argument and push on the returned repository.
*/
func PushChanges(hook PushPreHook, ref string, retries int64, retryInterval time.Duration) error {	
	repo, hookErr := hook()
	if hookErr != nil {
		return hookErr
	}

	//Repo object is nil, indicating there is nothing to push
	if repo == nil {
		return nil
	}

	refMap := gogitconf.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", ref, ref))
	pushErr := repo.Push(&gogit.PushOptions{
		Force: false,
		Prune: false,
		RemoteName: "origin",
		RefSpecs: []gogitconf.RefSpec{refMap},
	})

	if pushErr != nil {
		if pushErr.Error() == gogit.NoErrAlreadyUpToDate.Error() {
			fmt.Println("Push operation was no-op as remote was already up to date.")
			return nil
		}

		if strings.HasPrefix(pushErr.Error(), "non-fast-forward update:") {
			if retries == 0 {
				return errors.New(fmt.Sprintf("Push operation continuously failed due to remote updates. Giving up."))
			}
			
			fmt.Println("Push operation failed as remote was updated with non-local commits. Will retry.")
			time.Sleep(retryInterval)

			return PushChanges(hook, ref, retries - 1, retryInterval)
		}

		return pushErr
	}

	return nil
}