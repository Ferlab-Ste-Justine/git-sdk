package git

import (
	"path"
	"testing"

	"github.com/Ferlab-Ste-Justine/git-sdk/testutils"
)

func TestGetSshCredentials(t *testing.T) {
	teardown, giteaInfo, _, setupErr := testutils.SetupDefaultTestEnvironment()
	if setupErr != nil {
		t.Errorf("Error setting default test environment: %s", setupErr.Error())
		return
	}
	defer teardown()

	gitCreds, gitCredsErr := GetSshCredentials(path.Join("test", "keys", "ssh", "id_rsa"), giteaInfo.KnownHostsFile, "someUser")
	if gitCredsErr != nil {
		t.Errorf("Error retrieving ssh credentials: %s", gitCredsErr.Error())
		return
	}

	if !gitCreds.HasAuthMethod() {
		t.Errorf("Expected credentials to flag that auth method is defined")
		return
	}

	if gitCreds.Ssh.Keys.User != "someUser" {
		t.Errorf("Expected ssh credentials to have user 'someUser' and it had user '%s' instead", gitCreds.Ssh.Keys.User)
		return
	}
}

func TestGetHttpsCredentials(t *testing.T) {
	gitCreds := GetHttpsCredentials("someUser", "somePassword")

	if !gitCreds.HasAuthMethod() {
		t.Errorf("Expected credentials to flag that auth method is defined")
		return
	}

	if gitCreds.Https.Auth.Username != "someUser" || gitCreds.Https.Auth.Password != "somePassword" {
		t.Errorf("Expected https credentials to have username of 'someUser' and password of 'somePassword' it had username of '%s' and password of '%s' instead", gitCreds.Https.Auth.Username, gitCreds.Https.Auth.Password)
		return
	}
}