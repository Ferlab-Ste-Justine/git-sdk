package git

import (
	//"fmt"
	//"os"
	"path"
	"testing"
	//"time"

	"github.com/Ferlab-Ste-Justine/git-sdk/testutils"
)

func TestMemCloneGitRepo(t *testing.T) {
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

	_, store, cloneErr := MemCloneGitRepo(giteaInfo.RepoUrls[0], "main", 1, sshCreds)
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