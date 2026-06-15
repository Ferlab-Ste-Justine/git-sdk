package git

import (
	"path"
	"testing"

	"github.com/Ferlab-Ste-Justine/git-sdk/testutils"
)

func testSyncGitRepo(giteaInfo testutils.TestGiteaInfo, reposDir string, gitCreds *GitCredentials, repoUrl string, t *testing.T) {
	_, _, syncErr := SyncGitRepo(path.Join(reposDir, "test"), repoUrl, "main", gitCreds)
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

func TestSyncGitRepoSsh(t *testing.T) {
	teardown, giteaInfo, reposDir, setupErr := testutils.SetupDefaultTestEnvironment()
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

	testSyncGitRepo(giteaInfo, reposDir, sshCreds, giteaInfo.RepoUrls[0], t)
}

func TestSyncGitRepoHttp(t *testing.T) {
	teardown, giteaInfo, reposDir, setupErr := testutils.SetupDefaultTestEnvironment()
	if setupErr != nil {
		t.Errorf("Error setting default test environment: %s", setupErr.Error())
		return
	}
	defer teardown()

	httpsCreds := GetHttpsCredentials(giteaInfo.User, "test")

	testSyncGitRepo(giteaInfo, reposDir, httpsCreds, giteaInfo.RepoHttpUrls[0], t)

	// Second call exercises the pull path (exec-based fetch+reset for HTTPS)
	testSyncGitRepo(giteaInfo, reposDir, httpsCreds, giteaInfo.RepoHttpUrls[0], t)
}

func TestGetGitRepo(t *testing.T) {
	teardown, giteaInfo, reposDir, setupErr := testutils.SetupDefaultTestEnvironment()
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

	repo, _, syncErr := SyncGitRepo(path.Join(reposDir, "test"), giteaInfo.RepoUrls[0], "main", sshCreds)
	if syncErr != nil {
		t.Errorf("Error cloning repo test: %s", syncErr.Error())
		return
	}

	repoSame, repoSameErr := GetGitRepo(path.Join(reposDir, "test"))
	if repoSameErr != nil {
		t.Errorf("Error retrieving pre-existing cloned repo in a directory: %s", repoSameErr.Error())
		return
	}

	head, headErr := repo.Repo.Head()
	if headErr != nil {
		t.Errorf("Error accessing repo head: %s", headErr.Error())
		return
	}

	headSame, headSameErr := repoSame.Repo.Head()
	if headSameErr != nil {
		t.Errorf("Error accessing repo head for pre-existing cloned repo in a directory: %s", headSameErr.Error())
		return
	}

	if head.Hash().String() != headSame.Hash().String() {
		t.Errorf("Expected top commit hash from cloned repository and repository retrieved from the same location on filesystem to be the same and they weren't")
		return
	}
}