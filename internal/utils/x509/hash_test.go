/*
Copyright 2024 The cert-manager Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package x509_test

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	x509util "github.com/cert-manager/trust-manager-csi-driver/internal/utils/x509"

	_ "embed"
)

//go:embed testdata/cacert.pem
var CABundleTestData []byte

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

func TestForEachCertInBundle(t *testing.T) {
	tests := []struct {
		Name      string
		Bundle    []byte
		Test      func(t *testing.T, i int, cert *x509.Certificate, data []byte) error
		ExpectErr bool
	}{
		{
			Name: "single_cert_bundle",
			Bundle: []byte(`-----BEGIN CERTIFICATE-----
MIICCTCCAY6gAwIBAgINAgPluILrIPglJ209ZjAKBggqhkjOPQQDAzBHMQswCQYD
VQQGEwJVUzEiMCAGA1UEChMZR29vZ2xlIFRydXN0IFNlcnZpY2VzIExMQzEUMBIG
A1UEAxMLR1RTIFJvb3QgUjMwHhcNMTYwNjIyMDAwMDAwWhcNMzYwNjIyMDAwMDAw
WjBHMQswCQYDVQQGEwJVUzEiMCAGA1UEChMZR29vZ2xlIFRydXN0IFNlcnZpY2Vz
IExMQzEUMBIGA1UEAxMLR1RTIFJvb3QgUjMwdjAQBgcqhkjOPQIBBgUrgQQAIgNi
AAQfTzOHMymKoYTey8chWEGJ6ladK0uFxh1MJ7x/JlFyb+Kf1qPKzEUURout736G
jOyxfi//qXGdGIRFBEFVbivqJn+7kAHjSxm65FSWRQmx1WyRRK2EE46ajA2ADDL2
4CejQjBAMA4GA1UdDwEB/wQEAwIBhjAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQW
BBTB8Sa6oC2uhYHP0/EqEr24Cmf9vDAKBggqhkjOPQQDAwNpADBmAjEA9uEglRR7
VKOQFhG/hMjqb2sXnh5GmCCbn9MN2azTL818+FsuVbu/3ZL3pAzcMeGiAjEA/Jdm
ZuVDFhOD3cffL74UOO0BzrEXGhF16b0DjyZ+hOXJYKaV11RZt+cRLInUue4X
-----END CERTIFICATE-----`),
			Test: func(t *testing.T, i int, cert *x509.Certificate, data []byte) error {
				switch i {
				case 0:
					if cert.Subject.CommonName != "GTS Root R3" {
						t.Fatalf("expected common name to be %q, got %q", "GTS Root R3", cert.Subject.CommonName)
					}
				default:
					t.Fatalf("unexpected call to function")
				}

				return nil
			},
			ExpectErr: false,
		},
		{
			Name: "multiple_single_cert_bundle",
			Bundle: []byte(`-----BEGIN CERTIFICATE-----
MIICCTCCAY6gAwIBAgINAgPluILrIPglJ209ZjAKBggqhkjOPQQDAzBHMQswCQYD
VQQGEwJVUzEiMCAGA1UEChMZR29vZ2xlIFRydXN0IFNlcnZpY2VzIExMQzEUMBIG
A1UEAxMLR1RTIFJvb3QgUjMwHhcNMTYwNjIyMDAwMDAwWhcNMzYwNjIyMDAwMDAw
WjBHMQswCQYDVQQGEwJVUzEiMCAGA1UEChMZR29vZ2xlIFRydXN0IFNlcnZpY2Vz
IExMQzEUMBIGA1UEAxMLR1RTIFJvb3QgUjMwdjAQBgcqhkjOPQIBBgUrgQQAIgNi
AAQfTzOHMymKoYTey8chWEGJ6ladK0uFxh1MJ7x/JlFyb+Kf1qPKzEUURout736G
jOyxfi//qXGdGIRFBEFVbivqJn+7kAHjSxm65FSWRQmx1WyRRK2EE46ajA2ADDL2
4CejQjBAMA4GA1UdDwEB/wQEAwIBhjAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQW
BBTB8Sa6oC2uhYHP0/EqEr24Cmf9vDAKBggqhkjOPQQDAwNpADBmAjEA9uEglRR7
VKOQFhG/hMjqb2sXnh5GmCCbn9MN2azTL818+FsuVbu/3ZL3pAzcMeGiAjEA/Jdm
ZuVDFhOD3cffL74UOO0BzrEXGhF16b0DjyZ+hOXJYKaV11RZt+cRLInUue4X
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIICQjCCAcmgAwIBAgIQNjqWjMlcsljN0AFdxeVXADAKBggqhkjOPQQDAzBjMQswCQYDVQQGEwJE
RTEnMCUGA1UECgweRGV1dHNjaGUgVGVsZWtvbSBTZWN1cml0eSBHbWJIMSswKQYDVQQDDCJUZWxl
a29tIFNlY3VyaXR5IFRMUyBFQ0MgUm9vdCAyMDIwMB4XDTIwMDgyNTA3NDgyMFoXDTQ1MDgyNTIz
NTk1OVowYzELMAkGA1UEBhMCREUxJzAlBgNVBAoMHkRldXRzY2hlIFRlbGVrb20gU2VjdXJpdHkg
R21iSDErMCkGA1UEAwwiVGVsZWtvbSBTZWN1cml0eSBUTFMgRUNDIFJvb3QgMjAyMDB2MBAGByqG
SM49AgEGBSuBBAAiA2IABM6//leov9Wq9xCazbzREaK9Z0LMkOsVGJDZos0MKiXrPk/OtdKPD/M1
2kOLAoC+b1EkHQ9rK8qfwm9QMuU3ILYg/4gND21Ju9sGpIeQkpT0CdDPf8iAC8GXs7s1J8nCG6NC
MEAwHQYDVR0OBBYEFONyzG6VmUex5rNhTNHLq+O6zd6fMA8GA1UdEwEB/wQFMAMBAf8wDgYDVR0P
AQH/BAQDAgEGMAoGCCqGSM49BAMDA2cAMGQCMHVSi7ekEE+uShCLsoRbQuHmKjYC2qBuGT8lv9pZ
Mo7k+5Dck2TOrbRBR2Diz6fLHgIwN0GMZt9Ba9aDAEH9L1r3ULRn0SyocddDypwnJJGDSA3PzfdU
ga/sf+Rn27iQ7t0l
-----END CERTIFICATE-----`),
			Test: func(t *testing.T, i int, cert *x509.Certificate, data []byte) error {
				switch i {
				case 0:
					if cert.Subject.CommonName != "GTS Root R3" {
						t.Fatalf("expected common name to be %q, got %q", "GTS Root R3", cert.Subject.CommonName)
					}
				case 1:
					if cert.Subject.CommonName != "Telekom Security TLS ECC Root 2020" {
						t.Fatalf("expected common name to be %q, got %q", "Telekom Security TLS ECC Root 2020", cert.Subject.CommonName)
					}
				default:
					t.Fatalf("unexpected call to function")
				}

				return nil
			},
			ExpectErr: false,
		},
		{
			Name: "returning_error",
			Bundle: []byte(`-----BEGIN CERTIFICATE-----
MIICCTCCAY6gAwIBAgINAgPluILrIPglJ209ZjAKBggqhkjOPQQDAzBHMQswCQYD
VQQGEwJVUzEiMCAGA1UEChMZR29vZ2xlIFRydXN0IFNlcnZpY2VzIExMQzEUMBIG
A1UEAxMLR1RTIFJvb3QgUjMwHhcNMTYwNjIyMDAwMDAwWhcNMzYwNjIyMDAwMDAw
WjBHMQswCQYDVQQGEwJVUzEiMCAGA1UEChMZR29vZ2xlIFRydXN0IFNlcnZpY2Vz
IExMQzEUMBIGA1UEAxMLR1RTIFJvb3QgUjMwdjAQBgcqhkjOPQIBBgUrgQQAIgNi
AAQfTzOHMymKoYTey8chWEGJ6ladK0uFxh1MJ7x/JlFyb+Kf1qPKzEUURout736G
jOyxfi//qXGdGIRFBEFVbivqJn+7kAHjSxm65FSWRQmx1WyRRK2EE46ajA2ADDL2
4CejQjBAMA4GA1UdDwEB/wQEAwIBhjAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQW
BBTB8Sa6oC2uhYHP0/EqEr24Cmf9vDAKBggqhkjOPQQDAwNpADBmAjEA9uEglRR7
VKOQFhG/hMjqb2sXnh5GmCCbn9MN2azTL818+FsuVbu/3ZL3pAzcMeGiAjEA/Jdm
ZuVDFhOD3cffL74UOO0BzrEXGhF16b0DjyZ+hOXJYKaV11RZt+cRLInUue4X
-----END CERTIFICATE-----`),
			Test: func(t *testing.T, i int, cert *x509.Certificate, data []byte) error {
				return fmt.Errorf("testing")
			},
			ExpectErr: true,
		},
		{
			Name:   "mozilla_bundle",
			Bundle: CABundleTestData,
			Test: func(t *testing.T, i int, cert *x509.Certificate, data []byte) error {
				return nil
			},
			ExpectErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// Count calls to the func
			i := 0

			// Call func for every cert in bundle
			err := x509util.ForEachCertInBundle(test.Bundle, func(cert *x509.Certificate, data []byte) error {
				err := test.Test(t, i, cert, data)
				i++
				return err
			})

			// Test error result
			if (err != nil) != test.ExpectErr {
				t.Fatalf("unexpected error value: %v", err)
			}
		})
	}
}
