package notation

import (
	"context"
	"encoding/json"
	"os"

	"github.com/hashicorp/go-hclog"
	_ "github.com/notaryproject/notation-core-go/signature/cose"
	_ "github.com/notaryproject/notation-core-go/signature/jws"
	"github.com/notaryproject/notation-go"
	"github.com/notaryproject/notation-go/registry"
	"github.com/notaryproject/notation-go/verifier"
	"github.com/notaryproject/notation-go/verifier/trustpolicy"
	"github.com/notaryproject/notation-go/verifier/truststore"
	"oras.land/oras-go/v2/registry/remote"
)

type ImageVerifier interface {
	VerifyImage(ctx context.Context, imageReference string) error
}

// notationImageVerifier is a struct that represents an image verifier.
type notationImageVerifier struct {
	verifier             notation.Verifier
	repoPlainHTTP        bool
	maxSignatureAttempts int
	logger               hclog.Logger
}

// LoadTrustPolicyDocument loads a trust policy document from the given path.
// It returns the trust policy document or an error if the document cannot be loaded.
func LoadTrustPolicyDocument(path string) (*trustpolicy.Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	doc := &trustpolicy.Document{}
	err = json.Unmarshal(data, doc)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

// NewImageVerifier creates a new ImageVerifier instance with the given trust policy, trust store, and repoPlainHTTP flag.
// It returns the ImageVerifier instance or an error if the verifier cannot be created.
func NewImageVerifier(policy *trustpolicy.Document, truststore truststore.X509TrustStore, repoPlainHTTP bool, maxSignatureAttempts int, logger hclog.Logger) (ImageVerifier, error) {

	verifier, err := verifier.New(policy, truststore, nil)
	if err != nil {
		return nil, err
	}
	return &notationImageVerifier{
		verifier:             verifier,
		repoPlainHTTP:        repoPlainHTTP,
		logger:               logger,
		maxSignatureAttempts: maxSignatureAttempts,
	}, nil

}

// VerifyImage verifies the image with the given image reference.
// It returns an error if the verification fails.
func (iv *notationImageVerifier) VerifyImage(ctx context.Context, imageReference string) error {

	// derived from https://pkg.go.dev/github.com/notaryproject/notation-go@v1.0.1#example-package-RemoteVerify

	remoteRepo, err := remote.NewRepository(imageReference)
	remoteRepo.PlainHTTP = iv.repoPlainHTTP
	if err != nil {
		iv.logger.Debug("Remote repository creation failed", "err", err, "reference", imageReference)
		return err
	}
	repo := registry.NewRepository(remoteRepo)

	// verifyOptions is an example of notation.VerifyOptions.
	verifyOptions := notation.VerifyOptions{
		ArtifactReference:    imageReference,
		MaxSignatureAttempts: iv.maxSignatureAttempts,
	}

	// remote verify core process
	// upon successful verification, the target manifest descriptor
	// and signature verification outcome are returned.
	targetDesc, _, err := notation.Verify(ctx, iv.verifier, repo, verifyOptions)
	if err != nil {
		iv.logger.Debug("Notation verify failed", "err", err, "reference", imageReference)
		return err
	}

	iv.logger.Debug("Notation verify succeeded", "reference", imageReference, "digest", targetDesc.Digest, "size", targetDesc.Size, "mediaType", targetDesc.MediaType)
	return nil
}
