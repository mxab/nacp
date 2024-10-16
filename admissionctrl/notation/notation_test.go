package notation

import (
	"bufio"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/go-connections/nat"
	"github.com/hashicorp/go-hclog"
	"github.com/notaryproject/notation-core-go/signature/cose"
	"github.com/notaryproject/notation-core-go/testhelper"
	"github.com/notaryproject/notation-go"
	"github.com/notaryproject/notation-go/dir"
	"github.com/notaryproject/notation-go/registry"

	dockerRegistry "github.com/docker/docker/api/types/registry"
	"github.com/notaryproject/notation-go/signer"
	"github.com/notaryproject/notation-go/verifier/trustpolicy"
	"github.com/notaryproject/notation-go/verifier/truststore"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/bcrypt"
	"oras.land/oras-go/v2/registry/remote"
)

type creds struct {
	username string
	password string
}

func (c creds) credsJsonBase64() string {

	configJson, err := json.Marshal(dockerRegistry.AuthConfig{
		Username: c.username,
		Password: c.password,
	})
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString([]byte(configJson))
}
func (c creds) credsBase64() string {

	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.username, c.password)))
}
func (c creds) htpasswd() string {
	bcryptPassword, err := bcrypt.GenerateFromPassword([]byte(c.password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s:%s", c.username, string(bcryptPassword))
}

var registryTestCreds = creds{
	username: "testuser",
	password: "testpassword",
}

func TestLoadTrustPolicyDocument(t *testing.T) {
	pathDir := t.TempDir()

	content := `{
		"version": "1.0",
		"trustPolicies": [
			{
				"name": "wabbit-networks-images",
				"registryScopes": [ "*" ],
				"signatureVerification": {
					"level" : "strict"
				},
				"trustStores": [ "ca:wabbit-networks.io" ],
				"trustedIdentities": [
					"*"
				]
			}
		]
	}
	`
	path := filepath.Join(pathDir, "trust-policy.json")
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := LoadTrustPolicyDocument(path)
	require.NoError(t, err)
	require.Equal(t, &trustpolicy.Document{
		Version: "1.0",
		TrustPolicies: []trustpolicy.TrustPolicy{
			{
				Name: "wabbit-networks-images",

				RegistryScopes:        []string{"*"},
				SignatureVerification: trustpolicy.SignatureVerification{VerificationLevel: trustpolicy.LevelStrict.Name},
				TrustStores:           []string{"ca:wabbit-networks.io"},
				TrustedIdentities:     []string{"*"},
			},
		},
	}, doc)

}

func TestVerifyImage(t *testing.T) {

	tt := []struct {
		name  string
		creds *creds
	}{
		{
			name: "no password",
		},
		{
			name: "with password",

			creds: &registryTestCreds,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			configDir := t.TempDir()
			cleanupRegistry, _, address := launchRegistry(t, tc.creds)

			defer cleanupRegistry()

			digest := buildImage(t, address, tc.creds)

			testCertTuple := testhelper.GetRSASelfSignedSigningCertTuple("NACP Notation Testing")

			configFile := fmt.Sprintf("%s/config.json", configDir)

			if tc.creds != nil {

				configJson := []byte(fmt.Sprintf(`{
					"auths": {
						"%s": {
							"auth": "%s",
							"email": ""
						}
					}
				}`, address, tc.creds.credsBase64()))

				err := os.WriteFile(configFile, configJson, 0644)
				require.NoError(t, err)
			}
			client, err := NewClientWithFileCredStore(configFile)
			require.NoError(t, err)
			signImage(t, testCertTuple, digest, client)

			truststoreDir := t.TempDir()
			truststore := truststore.NewX509TrustStore(dir.NewSysFS(truststoreDir))
			writeTruststore(t, truststoreDir, "valid-trust-store", testCertTuple.Cert)

			imageVerifer, err := NewImageVerifier(policy(), truststore, true, 50, configFile, hclog.NewNullLogger())
			require.NoError(t, err)

			err = imageVerifer.VerifyImage(context.Background(), digest)
			require.NoError(t, err)
		})
	}
}

func policy() *trustpolicy.Document {
	policy := trustpolicy.Document{
		Version: "1.0",
		TrustPolicies: []trustpolicy.TrustPolicy{
			{
				Name:                  "test-statement-name",
				RegistryScopes:        []string{"*"},
				SignatureVerification: trustpolicy.SignatureVerification{VerificationLevel: trustpolicy.LevelStrict.Name},
				TrustStores:           []string{"ca:valid-trust-store"},
				TrustedIdentities:     []string{"*"},
			},
		},
	}
	return &policy
}

func launchRegistry(t *testing.T, creds *creds) (func(), nat.Port, string) {
	t.Helper()
	ctx := context.Background()

	env := map[string]string{
		"REGISTRY_STORAGE_DELETE_ENABLED": "true",
	}
	if creds != nil {
		env["REGISTRY_AUTH"] = "htpasswd"
		env["REGISTRY_AUTH_HTPASSWD_REALM"] = "Registry Realm"
		env["REGISTRY_AUTH_HTPASSWD_PATH"] = "/etc/registry.htpasswd"

	}

	req := testcontainers.ContainerRequest{
		Image:      "registry",
		WaitingFor: wait.ForListeningPort("5000"),
		Env:        env,
		// I have no clue why, but otherwise docker is not able to connect and push to the registry
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.PortBindings = map[nat.Port][]nat.PortBinding{
				"5000/tcp": {
					{
						HostIP:   "",
						HostPort: "0",
					},
				},
			}
		},
		ExposedPorts: []string{"5000"},
	}
	registryC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if creds != nil {
		entry := creds.htpasswd()
		err := registryC.CopyToContainer(ctx, []byte(entry), "/etc/registry.htpasswd", 777)
		if err != nil {
			t.Fatal(err)
		}
	}
	err = registryC.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}

	host, err := registryC.Host(ctx)
	//host = "host.docker.internal"

	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(host)

	port, err := registryC.MappedPort(ctx, "5000")
	require.NoError(t, err)
	if err != nil {
		t.Fatal(err)
	}

	return func() {
		if err := registryC.Terminate(ctx); err != nil {
			panic(err)
		}
	}, port, fmt.Sprintf("%s:%s", host, port.Port())

}

func buildImage(t *testing.T, address string, creds *creds) string {
	t.Helper()

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	if err != nil {
		t.Fatal(err)
	}
	// show connection details

	if err != nil {
		t.Fatal(err)
	}

	defer cli.Close()

	// create a temp file with Dockerfile contents

	//dfPath := "./testdata/verification.Dockerfile"
	tempDir := t.TempDir()

	//write a Dockerfile to tempDir
	dockerfile := `FROM alpine:latest
	RUN echo $(date) > /tmp/date.txt

	CMD ["cat", "/tmp/date.txt"]
	`
	// write contents to tempDir/Dockerfile
	dockerfilePath := filepath.Join(tempDir, "Dockerfile")
	err = os.WriteFile(dockerfilePath, []byte(dockerfile), 0644)
	if err != nil {
		t.Fatal(err)
	}

	tar, err := archive.TarWithOptions(tempDir, &archive.TarOptions{})
	if err != nil {
		t.Fatal(err)
	}
	baseImageTag := address + "/my-image"

	imageTag := baseImageTag + ":v1"

	opts := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{imageTag},

		NoCache: true,
	}

	buildRes, err := cli.ImageBuild(ctx, tar, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {

		items, err := cli.ImageRemove(ctx, imageTag, image.RemoveOptions{
			Force:         true,
			PruneChildren: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range items {
			fmt.Println(item)
		}
	}()
	scanner := bufio.NewScanner(buildRes.Body)
	for scanner.Scan() {
		lastLine := scanner.Text()
		// read last line as json
		info := map[string]interface{}{}
		err := json.Unmarshal([]byte(lastLine), &info)

		if err != nil {
			t.Fatal(err)
		}
		if info["error"] != nil {
			t.Fatalf("%v %v", info["error"], info["errorDetail"])
		}

		fmt.Println(lastLine)
	}

	auth := "nothing"
	if creds != nil {
		auth = creds.credsJsonBase64()
	}
	pushRes, err := cli.ImagePush(ctx, imageTag, image.PushOptions{
		All:          true,
		RegistryAuth: auth,
	})
	scanner = bufio.NewScanner(pushRes)
	for scanner.Scan() {
		lastLine := scanner.Text()

		info := map[string]interface{}{}
		err := json.Unmarshal([]byte(lastLine), &info)

		if err != nil {
			t.Fatal(err)
		}
		if info["error"] != nil {
			t.Fatalf("%v %v", info["error"], info["errorDetail"])
		}

		fmt.Println(lastLine)
	}
	if err != nil {
		t.Fatal(err)
	}

	insp, _, err := cli.ImageInspectWithRaw(ctx, imageTag)
	if err != nil {
		t.Fatal(err)
	}
	if len(insp.RepoDigests) == 0 {
		t.Fatal("no digest")
	}
	digest := insp.RepoDigests[0]

	return digest

}

// ExampleRemoteSign demonstrates how to use notation.Sign to sign an artifact
// in the remote registry and push the signature to the remote.
func signImage(t *testing.T, testCertTuple testhelper.RSACertTuple, exampleArtifactReference string, client remote.Client) {
	t.Helper()

	// testCertTuple contains a RSA privateKey and a self-signed X509
	// certificate generated for demo purpose ONLY.

	testingCerts := []*x509.Certificate{testCertTuple.Cert}

	// exampleSigner is a notation.Signer given key and X509 certificate chain.
	// Users should replace `exampleCertTuple.PrivateKey` with their own private
	// key and replace `exampleCerts` with the corresponding full certificate
	// chain, following the Notary certificate requirements:
	// https://github.com/notaryproject/notaryproject/blob/v1.0.0-rc.1/specs/signature-specification.md#certificate-requirements
	exampleSigner, err := signer.New(testCertTuple.PrivateKey, testingCerts)
	if err != nil {
		t.Fatal(err)
	}

	// exampleRepo is an example of registry.Repository.
	remoteRepo, err := remote.NewRepository(exampleArtifactReference)

	remoteRepo.Client = client
	remoteRepo.PlainHTTP = true
	if err != nil {
		t.Fatal(err)
	}
	exampleRepo := registry.NewRepository(remoteRepo)

	// exampleSignOptions is an example of notation.SignOptions.
	exampleSignOptions := notation.SignOptions{
		SignerSignOptions: notation.SignerSignOptions{
			SignatureMediaType: cose.MediaTypeEnvelope,
		},
		ArtifactReference: exampleArtifactReference,
	}

	// remote sign core process
	// upon successful signing, descriptor of the sign content is returned and
	// the generated signature is pushed into remote registry.
	targetDesc, err := notation.Sign(context.Background(), exampleSigner, exampleRepo, exampleSignOptions)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("Successfully signed")
	fmt.Println("targetDesc MediaType:", targetDesc.MediaType)
	fmt.Println("targetDesc Digest:", targetDesc.Digest)
	fmt.Println("targetDesc Size:", targetDesc.Size)
}

func writeTruststore(t *testing.T, path string, name string, cert *x509.Certificate) {
	t.Helper()
	// changing the path of the trust store for demo purpose.
	// Users could keep the default value, i.e. os.UserConfigDir.

	// Adding the certificate into the trust store.
	if err := os.MkdirAll(fmt.Sprintf("%s/truststore/x509/ca/%s", path, name), 0700); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(fmt.Sprintf("%s/truststore/x509/ca/%s/NotationExample.pem", path, name))

	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	err = pem.Encode(file, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	if err != nil {
		t.Fatal(err)
	}

}
