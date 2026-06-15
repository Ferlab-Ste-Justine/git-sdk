package git

import (
	"path"
	"testing"

	"github.com/Ferlab-Ste-Justine/git-sdk/testutils"
)

func testSyncGitRepoExec(giteaInfo testutils.TestGiteaInfo, reposDir string, gitCreds *GitCredentials, repoUrl string, t *testing.T) {
	_, _, syncErr := SyncGitRepoExec(path.Join(reposDir, "test"), repoUrl, "main", gitCreds)
	if syncErr != nil {
		t.Errorf("Error cloning repo test: %s", syncErr.Error())
		return
	}

	dirContent, dirContentErr := testutils.GetDirectoryContent(path.Join(reposDir, "test"), ".git")
	if dirContentErr != nil {
		t.Errorf("Error getting directory content of test: %s", dirContentErr.Error())
		return
	}

	if !dirContent.Equals(testutils.DirectoryContent(map[string]string{"README.md": "# test\n\ntest"})) {
		t.Errorf("Cloned directory content did not match expectations")
		return
	}
}

func TestSyncGitRepoExecHttp(t *testing.T) {
	teardown, giteaInfo, reposDir, setupErr := testutils.SetupDefaultTestEnvironment()
	if setupErr != nil {
		t.Errorf("Error setting default test environment: %s", setupErr.Error())
		return
	}
	defer teardown()

	httpsCreds := GetHttpsCredentials(giteaInfo.User, "test")

	testSyncGitRepoExec(giteaInfo, reposDir, httpsCreds, giteaInfo.RepoHttpUrls[0], t)

	// Validate with already cloned repo
	testSyncGitRepoExec(giteaInfo, reposDir, httpsCreds, giteaInfo.RepoHttpUrls[0], t)
}