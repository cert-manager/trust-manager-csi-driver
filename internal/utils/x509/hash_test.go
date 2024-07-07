package x509_test

import (
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	x509util "github.com/cert-manager/trust-manager-csi-driver/internal/utils/x509"
)

//go:embed testdata/cacert.pem
var CABundleTestData []byte

// func TestWriteOpenSSLHashFilesToStore(t *testing.T) {
// 	// Create store
// 	store := sync.MemoryStore{}

// 	// Call WriteOpenSSLHashFilesToStore
// 	if err := x509util.WriteOpenSSLHashFilesToStore(CABundleTestData, &store); err != nil {
// 		t.Fatalf("failed to write hashes to store: %s", err)
// 	}
// }

// TestCertificateSubjectHash tests the CertificateSubjectHash against all the
// certificates in testdata.
//
// All filenames in the testdata dir were generated from the Mozilla CA bundle
// using `openssl x509 -subject_hash`. This means that the filename is the
// openssl hash of the contents, which we can compare our hash function against
// to ensure they match
func TestCertificateSubjectHash(t *testing.T) {
	const testdata = "testdata"

	entries, err := os.ReadDir(testdata)
	if err != nil {
		t.Fatalf("could not read testdata directory: %s", err)
	}

	// Hashes are always 8 hexadecimal characters with no extension
	format := regexp.MustCompile("[a-f0-9]{8}")

	for _, entry := range entries {
		name := entry.Name()

		if !format.MatchString(name) {
			continue
		}

		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(testdata, name))
			if err != nil {
				t.Fatalf("could not read %q test file: %s", name, err)
			}

			block, _ := pem.Decode(data)
			if block == nil {
				t.Fatalf("could not parse pem file %q", name)
			}

			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				t.Fatalf("could not read %q test file: %s", name, err)
			}

			hash, err := x509util.CertificateSubjectHash(cert)
			if err != nil {
				t.Fatalf("could not hash %q test file: %s", name, err)
			}

			if hash != name {
				t.Fatalf("calculated incorrect hash for file: wanted %q, got %q", name, hash)
			}
		})
	}
}
