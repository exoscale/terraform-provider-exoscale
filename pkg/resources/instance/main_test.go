package instance_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func TestInstance(t *testing.T) {
	t.Parallel()

	t.Run("DataSource", testDataSource)
	t.Run("DataSourceList", testListDataSource)
	t.Run("Resource", testResource)
	t.Run("Resource/ManagedNetworkInterface", testResourceManagedNetworkInterface)
	t.Run("Resource/SSHKeys", testResourceSSHKeys)
	t.Run("Resource/VPC", testResourceVPC)
	t.Run("DestroyProtection/ExplicitValue", testExplicitDestroyProtection)
	t.Run("DestroyProtection/DefaultValue", testDefaultDestroyProtection)
}

// sshTestdataSpec adds SSH public keys unique to the test to the testdata
// templates. The API reports the SSH key of an instance by fingerprint, so
// tests sharing a hardcoded public key can read back another key's name.
type sshTestdataSpec struct {
	testutils.TestdataSpec
	SSHPublicKey  string
	SSHPublicKey2 string
}

// newSSHPublicKey returns a newly generated public key in the authorized_keys
// format.
func newSSHPublicKey() string {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	key, err := ssh.NewPublicKey(pub)
	if err != nil {
		panic(err)
	}

	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
}
