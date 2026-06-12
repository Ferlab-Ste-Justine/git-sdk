package git

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/plumbing/transport"
)

/*
Structure abstracting away http.BasicAuth structure needed by go-git to authenticate with git server in https
*/
type HttpsCredentials struct {
	Auth *http.BasicAuth
}

/*
Structure abstracting away ssh.PublicKeys structure needed by go-git to authenticate with git server in ssh
*/
type SshCredentials struct {
	Keys *ssh.PublicKeys
}

/*
Structure abstracting away git credentials, be they ssh or https (basic auth)
*/
type GitCredentials struct {
	Https *HttpsCredentials
	Ssh   *SshCredentials
}

func (creds *GitCredentials) HasAuthMethod() bool {
	return creds.Https != nil || creds.Ssh != nil
}

func (creds *GitCredentials) GetAuthMethod(url string) (transport.AuthMethod, error) {
	startsWithHttp := false
	if strings.HasPrefix(url, "http") {
		startsWithHttp = true
	}

	if creds.Https != nil {
		if !startsWithHttp && url != "" {
			return nil, errors.New("Cannot use https auth method when the protocol is not http")
		}

		return creds.Https.Auth, nil
	} else if creds.Ssh != nil {
		if startsWithHttp && url != "" {
			return nil, errors.New("Cannot use ssh auth method when the protocol is http")
		}

		return creds.Ssh.Keys, nil
	}

	return nil, errors.New("Error getting authentication method. Nothing was defined")

}

/*
Produces https credentials needed by go-git to clone/pull a remote repository and push to it via https.
*/
func GetHttpsCredentials(username string, password string) *GitCredentials {
	return &GitCredentials{
		Https: &HttpsCredentials{Auth: &http.BasicAuth{Username: username, Password: password}},
	}
}

/*
Produces ssh credentials needed by go-git to clone/pull a remote repository and push to it via ssh.
Arguments are file paths to the private ssh key of the user, ssh host key fingerprint of the git server and user to authentify as (will be 'git' if empty string is passed)
*/
func GetSshCredentials(sshKeyPath string, knownHostsPath string, user string) (*GitCredentials, error) {
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

	return &GitCredentials{
		Ssh: &SshCredentials{publicKeys},
	}, nil
}