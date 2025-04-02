package git

import (
	"path"
	"testing"

	"github.com/Ferlab-Ste-Justine/git-sdk/testutils"
)

func TestSyncGitRepo(t *testing.T) {
	teardown, giteaInfo, reposDir, setupErr := testutils.SetupDefaultTestEnvironment()
	if setupErr != nil {
		t.Errorf("Error setting default test environment: %s", setupErr.Error())
	}
	defer teardown()

	sshCreds, sshCredsErr := GetSshCredentials(path.Join("test", "keys", "ssh", "id_rsa"), giteaInfo.KnownHostsFile, giteaInfo.User)
	if sshCredsErr != nil {
		t.Errorf("Error retrieving ssh credentials: %s", sshCredsErr.Error())
	}

	_, _, syncErr := SyncGitRepo(path.Join(reposDir, "test"), giteaInfo.RepoUrls[0], "main", sshCreds)
	if syncErr != nil {
		t.Errorf("Error cloning repo test: %s", syncErr.Error())
	}

	dirContent, dirContentErr := testutils.GetDirectoryContent(path.Join(reposDir, "test"), ".git")
	if dirContentErr != nil {
		t.Errorf("Error getting directory content of test: %s", dirContentErr.Error())
	}

	if !dirContent.Equals(testutils.DirectoryContent(map[string]string{"README.md": "# test\n\ntest"})) {
		t.Errorf("Cloned directory content did not match expectations")
	}
}
