package git

import (
	"fmt"
	//"os"
	"path"
	"testing"
	//"time"

	"github.com/Ferlab-Ste-Justine/git-sdk/testutils"
)

func TestSyncGitRepo(t *testing.T) {
	teardown, giteaInfo, reposDir, setupErr := testutils.SetupDefaultTestEnvironment()
	if setupErr != nil {
		t.Errorf("Error setting default test environment: %s", setupErr.Error())
	}
	defer teardown()

	sshCreds, sshCredsErr := GetSshCredentials(path.Join("test", "keys", "id_rsa"), giteaInfo.KnownHostsFile)
	if sshCredsErr != nil {
		t.Errorf("Error retrieving ssh credentials: %s", sshCredsErr.Error())
	}

	fmt.Println(giteaInfo)
	//time.Sleep(60 * time.Second)

	_, _, syncErr := SyncGitRepo(path.Join(reposDir, "test"), giteaInfo.RepoUrls[0], "main", sshCreds)
	if syncErr != nil {
		t.Errorf("Error cloning repo test: %s", syncErr.Error())
	}
}
