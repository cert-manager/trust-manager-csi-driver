package x509

import (
	"crypto/sha1"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"strings"
)

// CertificateSubjectHash implements `openssl x509 -subject_hash`, outputting
// an 8 character hexadecimal hash of the given Certificate subject.
func CertificateSubjectHash(cert *x509.Certificate) (string, error) {
	var value asn1.RawValue
	if _, err := asn1.Unmarshal(cert.RawSubject, &value); err != nil {
		return "", err
	}

	value, err := normalizeASN1ValueForHash(value)
	if err != nil {
		return "", err
	}

	hashed := sha1.Sum(value.Bytes)
	encoded := hex.EncodeToString(hashed[:4])
	return encoded[6:8] + encoded[4:6] + encoded[2:4] + encoded[0:2], nil
}

// normalizeASN1ValueForHash "normalizes" an AS1 value with the following rules:
// - Strings are converted to UTF8
// - Strings are trimmed, lowercased and double spaces are removed
//
// This mirrors the behavior of openssl, which does this normalization when
// hashing a certificate subject.
func normalizeASN1ValueForHash(value asn1.RawValue) (_ asn1.RawValue, err error) {
	// Depending on the "Tag" we may need to:
	// - Recurse into it (for example for each sequence value)
	// - Normalize it, for example string types
	switch value.Tag {
	case asn1.TagBoolean:
	case asn1.TagInteger:
	case asn1.TagBitString:
	case asn1.TagOctetString:
	case asn1.TagNull:
	case asn1.TagOID:
	case asn1.TagEnum:
	case asn1.TagUTF8String:
		value, err = normalizeASN1StringForHash(value)
	case asn1.TagSequence, asn1.TagSet:
		sequenceData := value.Bytes
		value.Bytes = nil
		value.FullBytes = nil
		for len(sequenceData) > 0 {
			var sequenceItem asn1.RawValue
			sequenceData, err = asn1.Unmarshal(sequenceData, &sequenceItem)
			if err != nil {
				return asn1.RawValue{}, err
			}

			normalizedSequenceItem, err := normalizeASN1ValueForHash(sequenceItem)
			if err != nil {
				return asn1.RawValue{}, err
			}

			normalizedSequenceData, err := asn1.Marshal(normalizedSequenceItem)
			if err != nil {
				return asn1.RawValue{}, err
			}

			value.Bytes = append(value.Bytes, normalizedSequenceData...)
		}
	case asn1.TagNumericString:
	case asn1.TagPrintableString:
		value, err = normalizeASN1StringForHash(value)
	case asn1.TagT61String:
		value, err = normalizeASN1StringForHash(value)
	case asn1.TagIA5String:
		value, err = normalizeASN1StringForHash(value)
	case asn1.TagUTCTime:
	case asn1.TagGeneralizedTime:
	case asn1.TagGeneralString:
		value, err = normalizeASN1StringForHash(value)
	case asn1.TagBMPString:
		value, err = normalizeASN1StringForHash(value)
	}

	return value, err
}

func normalizeASN1StringForHash(value asn1.RawValue) (asn1.RawValue, error) {
	var str string
	if _, err := asn1.Unmarshal(value.FullBytes, &str); err != nil {
		return asn1.RawValue{}, err
	}

	// Trim whitespace, remove double spaces, translate to lowercase
	str = strings.ToLower(strings.Join(strings.Fields(str), " "))

	value.Tag = asn1.TagUTF8String
	value.Bytes = []byte(str)
	value.FullBytes = nil
	return value, nil
}

// ForEachCertInBundle calls the provided function for every certificate in a
// bundle file.
func ForEachCertInBundle(bundle []byte, fn func(cert *x509.Certificate, pem []byte) error) error {
	for block, rest := pem.Decode(bundle); block != nil; block, rest = pem.Decode(rest) {
		if block.Type != "CERTIFICATE" {
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return err
		}

		if err := fn(cert, pem.EncodeToMemory(block)); err != nil {
			return err
		}
	}

	return nil
}
