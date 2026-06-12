package git

import (
	"path"
	"testing"

	"github.com/Ferlab-Ste-Justine/git-sdk/testutils"
)

func testMemCloneGitRepo(giteaInfo testutils.TestGiteaInfo, gitCreds *GitCredentials, repoUrl string, t *testing.T) {
	_, store, cloneErr := MemCloneGitRepo(repoUrl, "main", 1, gitCreds)
	if cloneErr != nil {
		t.Errorf("Error cloning repo in memory: %s", cloneErr.Error())
		return
	}

	vals, valsErr := store.GetKeyVals("")
	if cloneErr != nil {
		t.Errorf("Error reading memory repo clone: %s", valsErr.Error())
		return
	}

	if !testutils.DirectoryContent(vals).Equals(testutils.DirectoryContent(map[string]string{"README.md": "# test\n\ntest"})) {
		t.Errorf("Cloned directory content did not match expectations")
		return
	}
}

func TestMemCloneGitRepoSsh(t *testing.T) {
	teardown, giteaInfo, _, setupErr := testutils.SetupDefaultTestEnvironment()
	if setupErr != nil {
		t.Errorf("Error setting default test environment: %s", setupErr.Error())
		return
	}
	defer teardown()

	sshCreds, sshCredsErr := GetSshCredentials(path.Join("test", "keys", "ssh", "id_rsa"), giteaInfo.KnownHostsFile, giteaInfo.User)
	if sshCredsErr != nil {
		t.Errorf("Error retrieving ssh credentials: %s", sshCredsErr.Error())
		return
	}

	testMemCloneGitRepo(giteaInfo, sshCreds, giteaInfo.RepoUrls[0], t)
}

func TestMemCloneGitRepoHttp(t *testing.T) {
	teardown, giteaInfo, _, setupErr := testutils.SetupDefaultTestEnvironment()
	if setupErr != nil {
		t.Errorf("Error setting default test environment: %s", setupErr.Error())
		return
	}
	defer teardown()

	httpsCreds := GetHttpsCredentials(giteaInfo.User, "test")

	testMemCloneGitRepo(giteaInfo, httpsCreds, giteaInfo.RepoHttpUrls[0], t)
}