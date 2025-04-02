package git

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	gogitconf "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

/*
Structure abstracting away ssh.PublicKeys structure needed by go-git to authenticate with git server
*/
type SshCredentials struct {
	Keys *ssh.PublicKeys
}

/*
Structure abstracting away openpgp.Entity structure needed by go-git to sign keys
*/
type CommitSignatureKey struct {
	Entity *openpgp.Entity
}

/*
Structure abstracting away gogit.Repository structure needed by go-git to manipulate a git repository
*/
type GitRepository struct {
	Repo *gogit.Repository
}

/*
Produces ssh credentials needed by go-git to clone/pull a remote repository and push to it.
Arguments are file paths to the private ssh key of the user, ssh host key fingerprint of the git server and user to authentify as (will be 'git' if empty string is passed)
*/
func GetSshCredentials(sshKeyPath string, knownHostsPath string, user string) (*SshCredentials, error) {
	_, statErr := os.Stat(sshKeyPath)
	if statErr != nil {
		return nil, errors.New(fmt.Sprintf("Failed to access ssh key file %s: %s", sshKeyPath, statErr.Error()))
	}

	if user == "" {
		user = "git"
	}

	publicKeys, pkGenErr := ssh.NewPublicKeysFromFile(user, sshKeyPath, "")
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

	(*publicKeys).HostKeyCallbackHelper.HostKeyCallback = callback

	return &SshCredentials{publicKeys}, nil
}

/*
Produces a commit signature needed to sign a commit.
Arguments are file paths to an armored private pgp key and optionally a passphrase to decrypt it if it is encrypted
*/
func GetSignatureKey(signKeyPath string, passphrasePath string) (*CommitSignatureKey, error) {
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

	return &CommitSignatureKey{signEntity}, nil
} 

/*
Verifies that the top commit of a given git repository was signed by one of the keys that are passed in the argument. 
Returns an error if it isn't.
*/
func VerifyTopCommit(repo *GitRepository, armoredKeyrings []string) error {
	head, headErr := repo.Repo.Head()
	if headErr != nil {
		return errors.New(fmt.Sprintf("Error accessing repo head: %s", headErr.Error()))
	}

	commit, commitErr := repo.Repo.CommitObject(head.Hash())
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
	//Optional key used to signed the git commit
	SignatureKey   *CommitSignatureKey
}

/*
Commits the given list of files in the git repository.
If not changes are detected in the files provided, a commit will not be attempted.
*/
func CommitFiles(repo *GitRepository, files []string, msg string, opts CommitOptions) (bool, error) {
	w, wErr := repo.Repo.Worktree()
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

	if opts.SignatureKey != nil {
		comOpts.SignKey = opts.SignatureKey.Entity
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
type PushPreHook func() (*GitRepository, error)

/*
Takes a function argument that should return a git repository with changes to push if there are (and nil otherwise).
From there, it will try to push the new commits in the repository to the given reference on origin.
If there are conflicts during the push, it will keep retrying by re-invoking its function argument and push on the returned repository.
*/
func PushChanges(hook PushPreHook, ref string, sshCred *SshCredentials, retries int64, retryInterval time.Duration) error {
	repo, hookErr := hook()
	if hookErr != nil {
		return hookErr
	}

	//Repo object is nil, indicating there is nothing to push
	if repo == nil {
		return nil
	}

	refMap := gogitconf.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", ref, ref))
	pushErr := repo.Repo.Push(&gogit.PushOptions{
		Auth: sshCred.Keys,
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

			return PushChanges(hook, ref, sshCred, retries - 1, retryInterval)
		}

		return errors.New(fmt.Sprintf("Error pushing file changes: %s", pushErr.Error()))
	}

	return nil
}