package git

import (
	"fmt"
	"path"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp/packet"

	"github.com/Ferlab-Ste-Justine/git-sdk/testutils"
)

func TestGetSshCredentials(t *testing.T) {
	teardown, giteaInfo, _, setupErr := testutils.SetupDefaultTestEnvironment()
	if setupErr != nil {
		t.Errorf("Error setting default test environment: %s", setupErr.Error())
	}
	defer teardown()

	sshCreds, sshCredsErr := GetSshCredentials(path.Join("test", "keys", "ssh", "id_rsa"), giteaInfo.KnownHostsFile, "someUser")
	if sshCredsErr != nil {
		t.Errorf("Error retrieving ssh credentials: %s", sshCredsErr.Error())
	}

	if sshCreds.Keys.User != "someUser" {
		t.Errorf("Expected ssh credentials to have user 'someUser' and it had user '%s' instead", sshCreds.Keys.User)
	}
}

func TestGetSignatureKey(t *testing.T) {
	sign1, err1 := GetSignatureKey(path.Join("test", "keys", "gpg_key_1"), "")
	if err1 != nil {
		t.Errorf(err1.Error())
	}

	if user, ok := sign1.Entity.Identities["user1 <user1@email.com>"]; ok {
		if user.Name != "user1 <user1@email.com>" {
			t.Errorf(fmt.Sprintf("'%s' was not expected 'user1 <user1@email.com>' value for identity name", user.Name))
		}
	} else {
		t.Errorf("Did not find expected identity in first gpg key")
	}

	if sign1.Entity.PrimaryKey.PubKeyAlgo != packet.PubKeyAlgoRSA {
		t.Errorf("Reported algorithm for parsed key doesn't match expected algorithm that was used during key generation")
	}
}