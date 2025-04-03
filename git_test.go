package git

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp/packet"

	"github.com/Ferlab-Ste-Justine/git-sdk/testutils"
)

func TestGetSshCredentials(t *testing.T) {
	teardown, giteaInfo, _, setupErr := testutils.SetupDefaultTestEnvironment()
	if setupErr != nil {
		t.Errorf("Error setting default test environment: %s", setupErr.Error())
		return
	}
	defer teardown()

	sshCreds, sshCredsErr := GetSshCredentials(path.Join("test", "keys", "ssh", "id_rsa"), giteaInfo.KnownHostsFile, "someUser")
	if sshCredsErr != nil {
		t.Errorf("Error retrieving ssh credentials: %s", sshCredsErr.Error())
		return
	}

	if sshCreds.Keys.User != "someUser" {
		t.Errorf("Expected ssh credentials to have user 'someUser' and it had user '%s' instead", sshCreds.Keys.User)
		return
	}
}

func TestGetSignatureKey(t *testing.T) {
	sign1, err1 := GetSignatureKey(path.Join("test", "keys", "gpg_key_1"), "")
	if err1 != nil {
		t.Errorf(err1.Error())
		return
	}

	if user, ok := sign1.Entity.Identities["user1 <user1@email.com>"]; ok {
		if user.Name != "user1 <user1@email.com>" {
			t.Errorf(fmt.Sprintf("'%s' was not expected 'user1 <user1@email.com>' value for identity name", user.Name))
			return
		}
	} else {
		t.Errorf("Did not find expected identity in first gpg key")
		return
	}

	if sign1.Entity.PrimaryKey.PubKeyAlgo != packet.PubKeyAlgoRSA {
		t.Errorf("Reported algorithm for parsed key doesn't match expected algorithm that was used during key generation")
		return
	}
}

func TestCommitFiles(t *testing.T) {
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

	topCommit, topCommitErr := GetTopCommit(repo)
	if topCommitErr != nil {
		t.Errorf("Error fetching top commit: %s", topCommitErr.Error())
		return
	}

	//Test no changes commit
	commitHappened, CommitErr := CommitFiles(repo, []string{"README.md"}, "Should not happen", CommitOptions{
		Name: giteaInfo.User,
		Email: "test@test.test",
	})
	if CommitErr != nil {
		t.Errorf("Error doing commit: %s", CommitErr.Error())
		return
	}


	if commitHappened {
		t.Errorf("Commit with no changes should not have gone through, yet function return indicated it did")
		return
	}

	topCommit2, topCommit2Err := GetTopCommit(repo)
	if topCommit2Err != nil {
		t.Errorf("Error fetching top commit: %s", topCommit2Err.Error())
		return
	}

	if !topCommit.IsSame(topCommit2) {
		t.Errorf("Commit with no changes should not have gone through, yet differing top commit indicated it did")
		return
	}

	//Test regular commit with file update and addition
	readmeErr := os.WriteFile(path.Join(reposDir, "test", "README.md"), []byte("# About"), 0770)
	if readmeErr != nil {
		t.Errorf("Error changing README file: %s", readmeErr.Error())
		return
	}
	
	anotherErr := os.WriteFile(path.Join(reposDir, "test", "Another.txt"), []byte("Just some text"), 0770)
	if anotherErr != nil {
		t.Errorf("Error creating another file: %s", anotherErr.Error())
		return
	}

	commitHappened, CommitErr = CommitFiles(repo, []string{"README.md", "Another.txt"}, "Some changes", CommitOptions{
		Name: giteaInfo.User,
		Email: "test@test.test",
	})
	if CommitErr != nil {
		t.Errorf("Error doing commit: %s", CommitErr.Error())
		return
	}

	if !commitHappened {
		t.Errorf("Commit with changes should have gone through, yet function return indicated it did not")
		return
	}

	topCommit, topCommitErr = GetTopCommit(repo)
	if topCommitErr != nil {
		t.Errorf("Error fetching top commit: %s", topCommitErr.Error())
		return
	}

	if topCommit.Commit.Message != "Some changes" {
		t.Errorf("Expected top commit message to be 'Some changes', but it was '%s'", topCommit.Commit.Message)
		return
	}

	//Test signed commit with file update and deletion
	signatureKey, signatureKeyErr := GetSignatureKey(path.Join("test", "keys", "gpg_key_1"), "")
	if signatureKeyErr != nil {
		t.Errorf("Error retrieving signature key: %s", signatureKeyErr.Error())
		return
	}

	readmeErr = os.WriteFile(path.Join(reposDir, "test", "README.md"), []byte("# About\n\nWIP"), 0770)
	if readmeErr != nil {
		t.Errorf("Error changing README file: %s", readmeErr.Error())
		return
	}

	anotherErr = os.RemoveAll(path.Join(reposDir, "test", "Another.txt"))
	if anotherErr != nil {
		t.Errorf("Error deleting another file: %s", anotherErr.Error())
		return
	}

	commitHappened, CommitErr = CommitFiles(repo, []string{"README.md", "Another.txt"}, "More changes", CommitOptions{
		Name: giteaInfo.User,
		Email: "test@test.test",
		SignatureKey: signatureKey,
	})
	if CommitErr != nil {
		t.Errorf("Error doing commit: %s", CommitErr.Error())
		return
	}

	if !commitHappened {
		t.Errorf("Commit with changes should have gone through, yet function return indicated it did not")
		return
	}

	topCommit, topCommitErr = GetTopCommit(repo)
	if topCommitErr != nil {
		t.Errorf("Error fetching top commit: %s", topCommitErr.Error())
		return
	}

	if topCommit.Commit.Message != "More changes" {
		t.Errorf("Expected top commit message to be 'Some changes', but it was '%s'", topCommit.Commit.Message)
		return
	}

	if topCommit.Commit.PGPSignature == "" {
		t.Errorf("Expected a signature on the top commit, but there was nont")
		return
	}
}

func TestVerifyTopCommit(t *testing.T) {
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

	signatureKey, signatureKeyErr := GetSignatureKey(path.Join("test", "keys", "gpg_key_1"), "")
	if signatureKeyErr != nil {
		t.Errorf("Error retrieving signature key: %s", signatureKeyErr.Error())
		return
	}

	key1Pub, key1PubErr := os.ReadFile(path.Join("test", "keys", "gpg_key_1.pub"))
	if key1PubErr != nil {
		t.Errorf("Error retrieving first public key: %s", key1PubErr.Error())
		return
	}

	key2Pub, key2PubErr := os.ReadFile(path.Join("test", "keys", "gpg_key_2.pub"))
	if key2PubErr != nil {
		t.Errorf("Error retrieving second public key: %s", key2PubErr.Error())
		return
	}

	key3Pub, key3PubErr := os.ReadFile(path.Join("test", "keys", "gpg_key_3.pub"))
	if key3PubErr != nil {
		t.Errorf("Error retrieving third public key: %s", key3PubErr.Error())
		return
	}

	//Test that unsigned commit doesn't pass a signature verification
	readmeErr := os.WriteFile(path.Join(reposDir, "test", "README.md"), []byte("# About"), 0770)
	if readmeErr != nil {
		t.Errorf("Error changing README file: %s", readmeErr.Error())
		return
	}

	_, CommitErr := CommitFiles(repo, []string{"README.md"}, "Some changes", CommitOptions{
		Name: giteaInfo.User,
		Email: "test@test.test",
	})
	if CommitErr != nil {
		t.Errorf("Error doing commit: %s", CommitErr.Error())
		return
	}

	if VerifyTopCommit(repo, []string{string(key1Pub), string(key2Pub), string(key3Pub)}) == nil {
		t.Errorf("Expected unsigned commit not to pass verification, but it did")
		return
	}

	//Test that signed commit with wrong key doesn't pass a signature verification
	readmeErr = os.WriteFile(path.Join(reposDir, "test", "README.md"), []byte("# About\n\nWIP"), 0770)
	if readmeErr != nil {
		t.Errorf("Error changing README file: %s", readmeErr.Error())
		return
	}

	_, CommitErr = CommitFiles(repo, []string{"README.md"}, "More changes", CommitOptions{
		Name: giteaInfo.User,
		Email: "test@test.test",
		SignatureKey: signatureKey,
	})
	if CommitErr != nil {
		t.Errorf("Error doing commit: %s", CommitErr.Error())
		return
	}

	if VerifyTopCommit(repo, []string{string(key2Pub), string(key3Pub)}) == nil {
		t.Errorf("Expected commit signed with wrong key not to pass verification, but it did")
		return
	}
	
	//Test that a signed commit with right key passes a signature verification
	if VerifyTopCommit(repo, []string{string(key2Pub), string(key3Pub), string(key1Pub)}) != nil {
		t.Errorf("Expected commit signed with right key to pass verification, but it did not")
		return
	}
}

func TestPushChanges(t *testing.T) {
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

	oneMinute, _ := time.ParseDuration("1m")

	pushErr := PushChanges(func() (*GitRepository, error) {
		repo, _, syncErr := SyncGitRepo(path.Join(reposDir, "test"), giteaInfo.RepoUrls[0], "main", sshCreds)
		if syncErr != nil {
			return nil, syncErr
		}

		anotherErr := os.WriteFile(path.Join(reposDir, "test", "Another.txt"), []byte("Just some text"), 0770)
		if anotherErr != nil {
			return repo, anotherErr
		}

		_, commitErr := CommitFiles(repo, []string{"Another.txt"}, "Some changes", CommitOptions{
			Name: giteaInfo.User,
			Email: "test@test.test",
		})
		if commitErr != nil {
			return repo, commitErr
		}

		yetAnotherErr := os.WriteFile(path.Join(reposDir, "test", "YetAnother.txt"), []byte("Yet more text"), 0770)
		if yetAnotherErr != nil {
			return repo, yetAnotherErr
		}

		_, commitErr = CommitFiles(repo, []string{"YetAnother.txt"}, "Yet more changes", CommitOptions{
			Name: giteaInfo.User,
			Email: "test@test.test",
		})
		if commitErr != nil {
			return repo, commitErr
		}

		return repo, nil
	}, "main", sshCreds, 3, oneMinute)
	
	if pushErr != nil {
		t.Errorf("Error pushing changes to gitea server: %s", pushErr.Error())
		return
	}

	repo, _, syncErr := SyncGitRepo(path.Join(reposDir, "test2"), giteaInfo.RepoUrls[0], "main", sshCreds)
	if syncErr != nil {
		t.Errorf("Error cloning repo test: %s", syncErr.Error())
		return
	}

	topCommit, topCommitErr := GetTopCommit(repo)
	if topCommitErr != nil {
		t.Errorf("Error fetching top commit: %s", topCommitErr.Error())
		return
	}

	if topCommit.Commit.Message != "Yet more changes" {
		t.Errorf("Expected top commit message to be 'Yet more changes', but it was '%s'", topCommit.Commit.Message)
		return
	}

	dirContent, dirContentErr := testutils.GetDirectoryContent(path.Join(reposDir, "test2"), ".git")
	if dirContentErr != nil {
		t.Errorf("Error getting directory content of test2: %s", dirContentErr.Error())
		return
	}

	expectedDirContent := map[string]string{
		"README.md": "# test\n\ntest",
		"Another.txt": "Just some text",
		"YetAnother.txt": "Yet more text",
	}
	if !dirContent.Equals(testutils.DirectoryContent(expectedDirContent)) {
		t.Errorf("Cloned directory content did not match expectations")
		return
	}
}