package main

import (
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/letsencrypt/boulder/test"
)

func TestLoadPubKey(t *testing.T) {
	_, _, err := loadPubKey("../../test/test-root.pubkey.pem")
	test.AssertNotError(t, err, "should not have errored")

	_, _, err = loadPubKey("../../test/hierarchy/int-e1.key.pem")
	test.AssertError(t, err, "should have failed trying to parse a private key")

	_, _, err = loadPubKey("/path/that/will/not/ever/exist/ever")
	test.AssertError(t, err, "should have failed opening public key at non-existent path")
	test.AssertErrorIs(t, err, fs.ErrNotExist)

	_, _, err = loadPubKey("../../test/hierarchy/int-e1.cert.pem")
	test.AssertError(t, err, "should have failed when trying to parse a certificate")
}

func TestCheckOutputFileSucceeds(t *testing.T) {
	dir := t.TempDir()
	err := checkOutputFile(dir+"/example", "foo")
	if err != nil {
		t.Fatal(err)
	}
}

func TestCheckOutputFileEmpty(t *testing.T) {
	err := checkOutputFile("", "foo")
	if err == nil {
		t.Fatal("expected error, got none")
	}
	if err.Error() != "outputs.foo is required" {
		t.Fatalf("wrong error: %s", err)
	}
}

func TestCheckOutputFileExists(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/example"
	err := writeFile(filename, []byte("hi"))
	if err != nil {
		t.Fatal(err)
	}
	err = checkOutputFile(filename, "foo")
	if err == nil {
		t.Fatal("expected error, got none")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("wrong error: %s", err)
	}
}

func TestKeyGenConfigValidate(t *testing.T) {
	cases := []struct {
		name          string
		config        keyGenConfig
		expectedError string
	}{
		{
			name:          "no key.type",
			config:        keyGenConfig{},
			expectedError: "key.type is required",
		},
		{
			name: "bad key.type",
			config: keyGenConfig{
				Type: "doop",
			},
			expectedError: "key.type can only be 'rsa' or 'ecdsa'",
		},
		{
			name: "bad key.rsa-mod-length",
			config: keyGenConfig{
				Type:         "rsa",
				RSAModLength: 1337,
			},
			expectedError: "key.rsa-mod-length can only be 2048 or 4096",
		},
		{
			name: "key.type is rsa but key.ecdsa-curve is present",
			config: keyGenConfig{
				Type:         "rsa",
				RSAModLength: 2048,
				ECDSACurve:   "bad",
			},
			expectedError: "if key.type = 'rsa' then key.ecdsa-curve is not used",
		},
		{
			name: "bad key.ecdsa-curve",
			config: keyGenConfig{
				Type:       "ecdsa",
				ECDSACurve: "bad",
			},
			expectedError: "key.ecdsa-curve can only be 'P-224', 'P-256', 'P-384', or 'P-521'",
		},
		{
			name: "key.type is ecdsa but key.rsa-mod-length is present",
			config: keyGenConfig{
				Type:         "ecdsa",
				RSAModLength: 2048,
				ECDSACurve:   "P-256",
			},
			expectedError: "if key.type = 'ecdsa' then key.rsa-mod-length is not used",
		},
		{
			name: "good rsa config",
			config: keyGenConfig{
				Type:         "rsa",
				RSAModLength: 2048,
			},
		},
		{
			name: "good ecdsa config",
			config: keyGenConfig{
				Type:       "ecdsa",
				ECDSACurve: "P-256",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.validate()
			if err != nil && err.Error() != tc.expectedError {
				t.Fatalf("Unexpected error, wanted: %q, got: %q", tc.expectedError, err)
			} else if err == nil && tc.expectedError != "" {
				t.Fatalf("validate didn't fail, wanted: %q", err)
			}
		})
	}
}

func TestRootConfigValidate(t *testing.T) {
	cases := []struct {
		name          string
		config        rootConfig
		expectedError string
	}{
		{
			name:          "no pkcs11.module",
			config:        rootConfig{},
			expectedError: "pkcs11.module is required",
		},
		{
			name: "no pkcs11.store-key-with-label",
			config: rootConfig{
				PKCS11: PKCS11KeyGenConfig{
					Module: "module",
				},
			},
			expectedError: "pkcs11.store-key-with-label is required",
		},
		{
			name: "bad key fields",
			config: rootConfig{
				PKCS11: PKCS11KeyGenConfig{
					Module:     "module",
					StoreLabel: "label",
				},
			},
			expectedError: "key.type is required",
		},
		{
			name: "no outputs.public-key-path",
			config: rootConfig{
				PKCS11: PKCS11KeyGenConfig{
					Module:     "module",
					StoreLabel: "label",
				},
				Key: keyGenConfig{
					Type:         "rsa",
					RSAModLength: 2048,
				},
			},
			expectedError: "outputs.public-key-path is required",
		},
		{
			name: "no outputs.certificate-path",
			config: rootConfig{
				PKCS11: PKCS11KeyGenConfig{
					Module:     "module",
					StoreLabel: "label",
				},
				Key: keyGenConfig{
					Type:         "rsa",
					RSAModLength: 2048,
				},
				Outputs: struct {
					PublicKeyPath   string `yaml:"public-key-path"`
					CertificatePath string `yaml:"certificate-path"`
				}{
					PublicKeyPath: "path",
				},
			},
			expectedError: "outputs.certificate-path is required",
		},
		{
			name: "bad certificate-profile",
			config: rootConfig{
				PKCS11: PKCS11KeyGenConfig{
					Module:     "module",
					StoreLabel: "label",
				},
				Key: keyGenConfig{
					Type:         "rsa",
					RSAModLength: 2048,
				},
				Outputs: struct {
					PublicKeyPath   string `yaml:"public-key-path"`
					CertificatePath string `yaml:"certificate-path"`
				}{
					PublicKeyPath:   "path",
					CertificatePath: "path",
				},
			},
			expectedError: "not-before is required",
		},
		{
			name: "good config",
			config: rootConfig{
				PKCS11: PKCS11KeyGenConfig{
					Module:     "module",
					StoreLabel: "label",
				},
				Key: keyGenConfig{
					Type:         "rsa",
					RSAModLength: 2048,
				},
				Outputs: struct {
					PublicKeyPath   string `yaml:"public-key-path"`
					CertificatePath string `yaml:"certificate-path"`
				}{
					PublicKeyPath:   "path",
					CertificatePath: "path",
				},
				CertProfile: certProfile{
					NotBefore:          "a",
					NotAfter:           "b",
					SignatureAlgorithm: "c",
					CommonName:         "d",
					Organization:       "e",
					Country:            "f",
				},
				SkipLints: []string{
					"e_ext_authority_key_identifier_missing",
					"e_ext_authority_key_identifier_no_key_identifier",
					"e_sub_ca_aia_missing",
					"e_sub_ca_certificate_policies_missing",
					"e_sub_ca_crl_distribution_points_missing",
					"n_ca_digital_signature_not_set",
					"n_mp_allowed_eku",
					"n_sub_ca_eku_missing",
					"w_sub_ca_aia_does_not_contain_issuing_ca_url",
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.validate()
			if err != nil && err.Error() != tc.expectedError {
				t.Fatalf("Unexpected error, wanted: %q, got: %q", tc.expectedError, err)
			} else if err == nil && tc.expectedError != "" {
				t.Fatalf("validate didn't fail, wanted: %q", err)
			}
		})
	}
}

func TestIntermediateConfigValidate(t *testing.T) {
	cases := []struct {
		name          string
		config        intermediateConfig
		expectedError string
	}{
		{
			name:          "no pkcs11.module",
			config:        intermediateConfig{},
			expectedError: "pkcs11.module is required",
		},
		{
			name: "no pkcs11.signing-key-label",
			config: intermediateConfig{
				PKCS11: PKCS11SigningConfig{
					Module: "module",
				},
			},
			expectedError: "pkcs11.signing-key-label is required",
		},
		{
			name: "no inputs.public-key-path",
			config: intermediateConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
			},
			expectedError: "inputs.public-key-path is required",
		},
		{
			name: "no inputs.issuer-certificate-path",
			config: intermediateConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					PublicKeyPath         string `yaml:"public-key-path"`
					IssuerCertificatePath string `yaml:"issuer-certificate-path"`
				}{
					PublicKeyPath: "path",
				},
			},
			expectedError: "inputs.issuer-certificate is required",
		},
		{
			name: "no outputs.certificate-path",
			config: intermediateConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					PublicKeyPath         string `yaml:"public-key-path"`
					IssuerCertificatePath string `yaml:"issuer-certificate-path"`
				}{
					PublicKeyPath:         "path",
					IssuerCertificatePath: "path",
				},
			},
			expectedError: "outputs.certificate-path is required",
		},
		{
			name: "bad certificate-profile",
			config: intermediateConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					PublicKeyPath         string `yaml:"public-key-path"`
					IssuerCertificatePath string `yaml:"issuer-certificate-path"`
				}{
					PublicKeyPath:         "path",
					IssuerCertificatePath: "path",
				},
				Outputs: struct {
					CertificatePath string `yaml:"certificate-path"`
				}{
					CertificatePath: "path",
				},
			},
			expectedError: "not-before is required",
		},
		{
			name: "too many policy OIDs",
			config: intermediateConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					PublicKeyPath         string `yaml:"public-key-path"`
					IssuerCertificatePath string `yaml:"issuer-certificate-path"`
				}{
					PublicKeyPath:         "path",
					IssuerCertificatePath: "path",
				},
				Outputs: struct {
					CertificatePath string `yaml:"certificate-path"`
				}{
					CertificatePath: "path",
				},
				CertProfile: certProfile{
					NotBefore:          "a",
					NotAfter:           "b",
					SignatureAlgorithm: "c",
					CommonName:         "d",
					Organization:       "e",
					Country:            "f",
					OCSPURL:            "g",
					CRLURL:             "h",
					IssuerURL:          "i",
					Policies:           []policyInfoConfig{{OID: "2.23.140.1.2.1"}, {OID: "6.6.6"}},
				},
				SkipLints: []string{},
			},
			expectedError: "policy should be exactly BRs domain-validated for subordinate CAs",
		},
		{
			name: "too few policy OIDs",
			config: intermediateConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					PublicKeyPath         string `yaml:"public-key-path"`
					IssuerCertificatePath string `yaml:"issuer-certificate-path"`
				}{
					PublicKeyPath:         "path",
					IssuerCertificatePath: "path",
				},
				Outputs: struct {
					CertificatePath string `yaml:"certificate-path"`
				}{
					CertificatePath: "path",
				},
				CertProfile: certProfile{
					NotBefore:          "a",
					NotAfter:           "b",
					SignatureAlgorithm: "c",
					CommonName:         "d",
					Organization:       "e",
					Country:            "f",
					OCSPURL:            "g",
					CRLURL:             "h",
					IssuerURL:          "i",
					Policies:           []policyInfoConfig{},
				},
				SkipLints: []string{},
			},
			expectedError: "policy should be exactly BRs domain-validated for subordinate CAs",
		},
		{
			name: "good config",
			config: intermediateConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					PublicKeyPath         string `yaml:"public-key-path"`
					IssuerCertificatePath string `yaml:"issuer-certificate-path"`
				}{
					PublicKeyPath:         "path",
					IssuerCertificatePath: "path",
				},
				Outputs: struct {
					CertificatePath string `yaml:"certificate-path"`
				}{
					CertificatePath: "path",
				},
				CertProfile: certProfile{
					NotBefore:          "a",
					NotAfter:           "b",
					SignatureAlgorithm: "c",
					CommonName:         "d",
					Organization:       "e",
					Country:            "f",
					OCSPURL:            "g",
					CRLURL:             "h",
					IssuerURL:          "i",
					Policies:           []policyInfoConfig{{OID: "2.23.140.1.2.1"}},
				},
				SkipLints: []string{},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.validate(intermediateCert)
			if err != nil && err.Error() != tc.expectedError {
				t.Fatalf("Unexpected error, wanted: %q, got: %q", tc.expectedError, err)
			} else if err == nil && tc.expectedError != "" {
				t.Fatalf("validate didn't fail, wanted: %q", err)
			}
		})
	}
}

func TestCrossCertConfigValidate(t *testing.T) {
	cases := []struct {
		name          string
		config        crossCertConfig
		expectedError string
	}{
		{
			name:          "no pkcs11.module",
			config:        crossCertConfig{},
			expectedError: "pkcs11.module is required",
		},
		{
			name: "no pkcs11.signing-key-label",
			config: crossCertConfig{
				PKCS11: PKCS11SigningConfig{
					Module: "module",
				},
			},
			expectedError: "pkcs11.signing-key-label is required",
		},
		{
			name: "no inputs.public-key-path",
			config: crossCertConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
			},
			expectedError: "inputs.public-key-path is required",
		},
		{
			name: "no inputs.issuer-certificate-path",
			config: crossCertConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					PublicKeyPath              string `yaml:"public-key-path"`
					IssuerCertificatePath      string `yaml:"issuer-certificate-path"`
					CertificateToCrossSignPath string `yaml:"certificate-to-cross-sign-path"`
				}{
					PublicKeyPath:              "path",
					CertificateToCrossSignPath: "path",
				},
			},
			expectedError: "inputs.issuer-certificate is required",
		},
		{
			name: "no inputs.certificate-to-cross-sign-path",
			config: crossCertConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					PublicKeyPath              string `yaml:"public-key-path"`
					IssuerCertificatePath      string `yaml:"issuer-certificate-path"`
					CertificateToCrossSignPath string `yaml:"certificate-to-cross-sign-path"`
				}{
					PublicKeyPath:         "path",
					IssuerCertificatePath: "path",
				},
			},
			expectedError: "inputs.certificate-to-cross-sign-path is required",
		},
		{
			name: "no outputs.certificate-path",
			config: crossCertConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					PublicKeyPath              string `yaml:"public-key-path"`
					IssuerCertificatePath      string `yaml:"issuer-certificate-path"`
					CertificateToCrossSignPath string `yaml:"certificate-to-cross-sign-path"`
				}{
					PublicKeyPath:              "path",
					IssuerCertificatePath:      "path",
					CertificateToCrossSignPath: "path",
				},
			},
			expectedError: "outputs.certificate-path is required",
		},
		{
			name: "bad certificate-profile",
			config: crossCertConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					PublicKeyPath              string `yaml:"public-key-path"`
					IssuerCertificatePath      string `yaml:"issuer-certificate-path"`
					CertificateToCrossSignPath string `yaml:"certificate-to-cross-sign-path"`
				}{
					PublicKeyPath:              "path",
					IssuerCertificatePath:      "path",
					CertificateToCrossSignPath: "path",
				},
				Outputs: struct {
					CertificatePath string `yaml:"certificate-path"`
				}{
					CertificatePath: "path",
				},
			},
			expectedError: "not-before is required",
		},
		{
			name: "too many policy OIDs",
			config: crossCertConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					PublicKeyPath              string `yaml:"public-key-path"`
					IssuerCertificatePath      string `yaml:"issuer-certificate-path"`
					CertificateToCrossSignPath string `yaml:"certificate-to-cross-sign-path"`
				}{
					PublicKeyPath:              "path",
					IssuerCertificatePath:      "path",
					CertificateToCrossSignPath: "path",
				},
				Outputs: struct {
					CertificatePath string `yaml:"certificate-path"`
				}{
					CertificatePath: "path",
				},
				CertProfile: certProfile{
					NotBefore:          "a",
					NotAfter:           "b",
					SignatureAlgorithm: "c",
					CommonName:         "d",
					Organization:       "e",
					Country:            "f",
					OCSPURL:            "g",
					CRLURL:             "h",
					IssuerURL:          "i",
					Policies:           []policyInfoConfig{{OID: "2.23.140.1.2.1"}, {OID: "6.6.6"}},
				},
				SkipLints: []string{},
			},
			expectedError: "policy should be exactly BRs domain-validated for subordinate CAs",
		},
		{
			name: "too few policy OIDs",
			config: crossCertConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					PublicKeyPath              string `yaml:"public-key-path"`
					IssuerCertificatePath      string `yaml:"issuer-certificate-path"`
					CertificateToCrossSignPath string `yaml:"certificate-to-cross-sign-path"`
				}{
					PublicKeyPath:              "path",
					IssuerCertificatePath:      "path",
					CertificateToCrossSignPath: "path",
				},
				Outputs: struct {
					CertificatePath string `yaml:"certificate-path"`
				}{
					CertificatePath: "path",
				},
				CertProfile: certProfile{
					NotBefore:          "a",
					NotAfter:           "b",
					SignatureAlgorithm: "c",
					CommonName:         "d",
					Organization:       "e",
					Country:            "f",
					OCSPURL:            "g",
					CRLURL:             "h",
					IssuerURL:          "i",
					Policies:           []policyInfoConfig{},
				},
				SkipLints: []string{},
			},
			expectedError: "policy should be exactly BRs domain-validated for subordinate CAs",
		},
		{
			name: "good config",
			config: crossCertConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					PublicKeyPath              string `yaml:"public-key-path"`
					IssuerCertificatePath      string `yaml:"issuer-certificate-path"`
					CertificateToCrossSignPath string `yaml:"certificate-to-cross-sign-path"`
				}{
					PublicKeyPath:              "path",
					IssuerCertificatePath:      "path",
					CertificateToCrossSignPath: "path",
				},
				Outputs: struct {
					CertificatePath string `yaml:"certificate-path"`
				}{
					CertificatePath: "path",
				},
				CertProfile: certProfile{
					NotBefore:          "a",
					NotAfter:           "b",
					SignatureAlgorithm: "c",
					CommonName:         "d",
					Organization:       "e",
					Country:            "f",
					OCSPURL:            "g",
					CRLURL:             "h",
					IssuerURL:          "i",
					Policies:           []policyInfoConfig{{OID: "2.23.140.1.2.1"}},
				},
				SkipLints: []string{},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.validate()
			if err != nil && err.Error() != tc.expectedError {
				t.Fatalf("Unexpected error, wanted: %q, got: %q", tc.expectedError, err)
			} else if err == nil && tc.expectedError != "" {
				t.Fatalf("validate didn't fail, wanted: %q", err)
			}
		})
	}
}

func TestCSRConfigValidate(t *testing.T) {
	cases := []struct {
		name          string
		config        csrConfig
		expectedError string
	}{
		{
			name:          "no pkcs11.module",
			config:        csrConfig{},
			expectedError: "pkcs11.module is required",
		},
		{
			name: "no pkcs11.signing-key-label",
			config: csrConfig{
				PKCS11: PKCS11SigningConfig{
					Module: "module",
				},
			},
			expectedError: "pkcs11.signing-key-label is required",
		},
		{
			name: "no inputs.public-key-path",
			config: csrConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
			},
			expectedError: "inputs.public-key-path is required",
		},
		{
			name: "no outputs.csr-path",
			config: csrConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					PublicKeyPath string `yaml:"public-key-path"`
				}{
					PublicKeyPath: "path",
				},
			},
			expectedError: "outputs.csr-path is required",
		},
		{
			name: "bad certificate-profile",
			config: csrConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					PublicKeyPath string `yaml:"public-key-path"`
				}{
					PublicKeyPath: "path",
				},
				Outputs: struct {
					CSRPath string `yaml:"csr-path"`
				}{
					CSRPath: "path",
				},
			},
			expectedError: "common-name is required",
		},
		{
			name: "good config",
			config: csrConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					PublicKeyPath string `yaml:"public-key-path"`
				}{
					PublicKeyPath: "path",
				},
				Outputs: struct {
					CSRPath string `yaml:"csr-path"`
				}{
					CSRPath: "path",
				},
				CertProfile: certProfile{
					CommonName:   "d",
					Organization: "e",
					Country:      "f",
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.validate()
			if err != nil && err.Error() != tc.expectedError {
				t.Fatalf("Unexpected error, wanted: %q, got: %q", tc.expectedError, err)
			} else if err == nil && tc.expectedError != "" {
				t.Fatalf("validate didn't fail, wanted: %q", err)
			}
		})
	}
}

func TestKeyConfigValidate(t *testing.T) {
	cases := []struct {
		name          string
		config        keyConfig
		expectedError string
	}{
		{
			name:          "no pkcs11.module",
			config:        keyConfig{},
			expectedError: "pkcs11.module is required",
		},
		{
			name: "no pkcs11.store-key-with-label",
			config: keyConfig{
				PKCS11: PKCS11KeyGenConfig{
					Module: "module",
				},
			},
			expectedError: "pkcs11.store-key-with-label is required",
		},
		{
			name: "bad key fields",
			config: keyConfig{
				PKCS11: PKCS11KeyGenConfig{
					Module:     "module",
					StoreLabel: "label",
				},
			},
			expectedError: "key.type is required",
		},
		{
			name: "no outputs.public-key-path",
			config: keyConfig{
				PKCS11: PKCS11KeyGenConfig{
					Module:     "module",
					StoreLabel: "label",
				},
				Key: keyGenConfig{
					Type:         "rsa",
					RSAModLength: 2048,
				},
			},
			expectedError: "outputs.public-key-path is required",
		},
		{
			name: "good config",
			config: keyConfig{
				PKCS11: PKCS11KeyGenConfig{
					Module:     "module",
					StoreLabel: "label",
				},
				Key: keyGenConfig{
					Type:         "rsa",
					RSAModLength: 2048,
				},
				Outputs: struct {
					PublicKeyPath    string `yaml:"public-key-path"`
					PKCS11ConfigPath string `yaml:"pkcs11-config-path"`
				}{
					PublicKeyPath:    "path",
					PKCS11ConfigPath: "path.json",
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.validate()
			if err != nil && err.Error() != tc.expectedError {
				t.Fatalf("Unexpected error, wanted: %q, got: %q", tc.expectedError, err)
			} else if err == nil && tc.expectedError != "" {
				t.Fatalf("validate didn't fail, wanted: %q", err)
			}
		})
	}
}

func TestOCSPRespConfig(t *testing.T) {
	cases := []struct {
		name          string
		config        ocspRespConfig
		expectedError string
	}{
		{
			name:          "no pkcs11.module",
			config:        ocspRespConfig{},
			expectedError: "pkcs11.module is required",
		},
		{
			name: "no pkcs11.signing-key-label",
			config: ocspRespConfig{
				PKCS11: PKCS11SigningConfig{
					Module: "module",
				},
			},
			expectedError: "pkcs11.signing-key-label is required",
		},
		{
			name: "no inputs.certificate-path",
			config: ocspRespConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
			},
			expectedError: "inputs.certificate-path is required",
		},
		{
			name: "no inputs.issuer-certificate-path",
			config: ocspRespConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					CertificatePath                string `yaml:"certificate-path"`
					IssuerCertificatePath          string `yaml:"issuer-certificate-path"`
					DelegatedIssuerCertificatePath string `yaml:"delegated-issuer-certificate-path"`
				}{
					CertificatePath: "path",
				},
			},
			expectedError: "inputs.issuer-certificate-path is required",
		},
		{
			name: "no outputs.response-path",
			config: ocspRespConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					CertificatePath                string `yaml:"certificate-path"`
					IssuerCertificatePath          string `yaml:"issuer-certificate-path"`
					DelegatedIssuerCertificatePath string `yaml:"delegated-issuer-certificate-path"`
				}{
					CertificatePath:       "path",
					IssuerCertificatePath: "path",
				},
			},
			expectedError: "outputs.response-path is required",
		},
		{
			name: "no ocsp-profile.this-update",
			config: ocspRespConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					CertificatePath                string `yaml:"certificate-path"`
					IssuerCertificatePath          string `yaml:"issuer-certificate-path"`
					DelegatedIssuerCertificatePath string `yaml:"delegated-issuer-certificate-path"`
				}{
					CertificatePath:       "path",
					IssuerCertificatePath: "path",
				},
				Outputs: struct {
					ResponsePath string `yaml:"response-path"`
				}{
					ResponsePath: "path",
				},
			},
			expectedError: "ocsp-profile.this-update is required",
		},
		{
			name: "no ocsp-profile.next-update",
			config: ocspRespConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					CertificatePath                string `yaml:"certificate-path"`
					IssuerCertificatePath          string `yaml:"issuer-certificate-path"`
					DelegatedIssuerCertificatePath string `yaml:"delegated-issuer-certificate-path"`
				}{
					CertificatePath:       "path",
					IssuerCertificatePath: "path",
				},
				Outputs: struct {
					ResponsePath string `yaml:"response-path"`
				}{
					ResponsePath: "path",
				},
				OCSPProfile: struct {
					ThisUpdate string `yaml:"this-update"`
					NextUpdate string `yaml:"next-update"`
					Status     string `yaml:"status"`
				}{
					ThisUpdate: "this-update",
				},
			},
			expectedError: "ocsp-profile.next-update is required",
		},
		{
			name: "no ocsp-profile.status",
			config: ocspRespConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					CertificatePath                string `yaml:"certificate-path"`
					IssuerCertificatePath          string `yaml:"issuer-certificate-path"`
					DelegatedIssuerCertificatePath string `yaml:"delegated-issuer-certificate-path"`
				}{
					CertificatePath:       "path",
					IssuerCertificatePath: "path",
				},
				Outputs: struct {
					ResponsePath string `yaml:"response-path"`
				}{
					ResponsePath: "path",
				},
				OCSPProfile: struct {
					ThisUpdate string `yaml:"this-update"`
					NextUpdate string `yaml:"next-update"`
					Status     string `yaml:"status"`
				}{
					ThisUpdate: "this-update",
					NextUpdate: "next-update",
				},
			},
			expectedError: "ocsp-profile.status must be either \"good\" or \"revoked\"",
		},
		{
			name: "good config",
			config: ocspRespConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					CertificatePath                string `yaml:"certificate-path"`
					IssuerCertificatePath          string `yaml:"issuer-certificate-path"`
					DelegatedIssuerCertificatePath string `yaml:"delegated-issuer-certificate-path"`
				}{
					CertificatePath:       "path",
					IssuerCertificatePath: "path",
				},
				Outputs: struct {
					ResponsePath string `yaml:"response-path"`
				}{
					ResponsePath: "path",
				},
				OCSPProfile: struct {
					ThisUpdate string `yaml:"this-update"`
					NextUpdate string `yaml:"next-update"`
					Status     string `yaml:"status"`
				}{
					ThisUpdate: "this-update",
					NextUpdate: "next-update",
					Status:     "good",
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.validate()
			if err != nil && err.Error() != tc.expectedError {
				t.Fatalf("Unexpected error, wanted: %q, got: %q", tc.expectedError, err)
			} else if err == nil && tc.expectedError != "" {
				t.Fatalf("validate didn't fail, wanted: %q", err)
			}
		})
	}
}

func TestCRLConfig(t *testing.T) {
	cases := []struct {
		name          string
		config        crlConfig
		expectedError string
	}{
		{
			name:          "no pkcs11.module",
			config:        crlConfig{},
			expectedError: "pkcs11.module is required",
		},
		{
			name: "no pkcs11.signing-key-label",
			config: crlConfig{
				PKCS11: PKCS11SigningConfig{
					Module: "module",
				},
			},
			expectedError: "pkcs11.signing-key-label is required",
		},
		{
			name: "no inputs.issuer-certificate-path",
			config: crlConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
			},
			expectedError: "inputs.issuer-certificate-path is required",
		},
		{
			name: "no outputs.crl-path",
			config: crlConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					IssuerCertificatePath string `yaml:"issuer-certificate-path"`
				}{
					IssuerCertificatePath: "path",
				},
			},
			expectedError: "outputs.crl-path is required",
		},
		{
			name: "no crl-profile.this-update",
			config: crlConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					IssuerCertificatePath string `yaml:"issuer-certificate-path"`
				}{
					IssuerCertificatePath: "path",
				},
				Outputs: struct {
					CRLPath string `yaml:"crl-path"`
				}{
					CRLPath: "path",
				},
			},
			expectedError: "crl-profile.this-update is required",
		},
		{
			name: "no crl-profile.next-update",
			config: crlConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					IssuerCertificatePath string `yaml:"issuer-certificate-path"`
				}{
					IssuerCertificatePath: "path",
				},
				Outputs: struct {
					CRLPath string `yaml:"crl-path"`
				}{
					CRLPath: "path",
				},
				CRLProfile: struct {
					ThisUpdate          string `yaml:"this-update"`
					NextUpdate          string `yaml:"next-update"`
					Number              int64  `yaml:"number"`
					RevokedCertificates []struct {
						CertificatePath  string `yaml:"certificate-path"`
						RevocationDate   string `yaml:"revocation-date"`
						RevocationReason int    `yaml:"revocation-reason"`
					} `yaml:"revoked-certificates"`
					IssuingDistributionPoint string `yaml:"issuing-distribution-point"`
				}{
					ThisUpdate:               "this-update",
					IssuingDistributionPoint: "http://example.com",
				},
			},
			expectedError: "crl-profile.next-update is required",
		},
		{
			name: "no crl-profile.number",
			config: crlConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					IssuerCertificatePath string `yaml:"issuer-certificate-path"`
				}{
					IssuerCertificatePath: "path",
				},
				Outputs: struct {
					CRLPath string `yaml:"crl-path"`
				}{
					CRLPath: "path",
				},
				CRLProfile: struct {
					ThisUpdate          string `yaml:"this-update"`
					NextUpdate          string `yaml:"next-update"`
					Number              int64  `yaml:"number"`
					RevokedCertificates []struct {
						CertificatePath  string `yaml:"certificate-path"`
						RevocationDate   string `yaml:"revocation-date"`
						RevocationReason int    `yaml:"revocation-reason"`
					} `yaml:"revoked-certificates"`
					IssuingDistributionPoint string `yaml:"issuing-distribution-point"`
				}{
					ThisUpdate:               "this-update",
					NextUpdate:               "next-update",
					IssuingDistributionPoint: "http://example.com",
				},
			},
			expectedError: "crl-profile.number must be non-zero",
		},
		{
			name: "no crl-profile.revoked-certificates.certificate-path",
			config: crlConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					IssuerCertificatePath string `yaml:"issuer-certificate-path"`
				}{
					IssuerCertificatePath: "path",
				},
				Outputs: struct {
					CRLPath string `yaml:"crl-path"`
				}{
					CRLPath: "path",
				},
				CRLProfile: struct {
					ThisUpdate          string `yaml:"this-update"`
					NextUpdate          string `yaml:"next-update"`
					Number              int64  `yaml:"number"`
					RevokedCertificates []struct {
						CertificatePath  string `yaml:"certificate-path"`
						RevocationDate   string `yaml:"revocation-date"`
						RevocationReason int    `yaml:"revocation-reason"`
					} `yaml:"revoked-certificates"`
					IssuingDistributionPoint string `yaml:"issuing-distribution-point"`
				}{
					ThisUpdate: "this-update",
					NextUpdate: "next-update",
					Number:     1,
					RevokedCertificates: []struct {
						CertificatePath  string `yaml:"certificate-path"`
						RevocationDate   string `yaml:"revocation-date"`
						RevocationReason int    `yaml:"revocation-reason"`
					}{{}},
					IssuingDistributionPoint: "http://example.com",
				},
			},
			expectedError: "crl-profile.revoked-certificates.certificate-path is required",
		},
		{
			name: "no crl-profile.revoked-certificates.revocation-date",
			config: crlConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					IssuerCertificatePath string `yaml:"issuer-certificate-path"`
				}{
					IssuerCertificatePath: "path",
				},
				Outputs: struct {
					CRLPath string `yaml:"crl-path"`
				}{
					CRLPath: "path",
				},
				CRLProfile: struct {
					ThisUpdate          string `yaml:"this-update"`
					NextUpdate          string `yaml:"next-update"`
					Number              int64  `yaml:"number"`
					RevokedCertificates []struct {
						CertificatePath  string `yaml:"certificate-path"`
						RevocationDate   string `yaml:"revocation-date"`
						RevocationReason int    `yaml:"revocation-reason"`
					} `yaml:"revoked-certificates"`
					IssuingDistributionPoint string `yaml:"issuing-distribution-point"`
				}{
					ThisUpdate: "this-update",
					NextUpdate: "next-update",
					Number:     1,
					RevokedCertificates: []struct {
						CertificatePath  string `yaml:"certificate-path"`
						RevocationDate   string `yaml:"revocation-date"`
						RevocationReason int    `yaml:"revocation-reason"`
					}{{
						CertificatePath: "path",
					}},
					IssuingDistributionPoint: "http://example.com",
				},
			},
			expectedError: "crl-profile.revoked-certificates.revocation-date is required",
		},
		{
			name: "no revocation reason",
			config: crlConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					IssuerCertificatePath string `yaml:"issuer-certificate-path"`
				}{
					IssuerCertificatePath: "path",
				},
				Outputs: struct {
					CRLPath string `yaml:"crl-path"`
				}{
					CRLPath: "path",
				},
				CRLProfile: struct {
					ThisUpdate          string `yaml:"this-update"`
					NextUpdate          string `yaml:"next-update"`
					Number              int64  `yaml:"number"`
					RevokedCertificates []struct {
						CertificatePath  string `yaml:"certificate-path"`
						RevocationDate   string `yaml:"revocation-date"`
						RevocationReason int    `yaml:"revocation-reason"`
					} `yaml:"revoked-certificates"`
					IssuingDistributionPoint string `yaml:"issuing-distribution-point"`
				}{
					ThisUpdate: "this-update",
					NextUpdate: "next-update",
					Number:     1,
					RevokedCertificates: []struct {
						CertificatePath  string `yaml:"certificate-path"`
						RevocationDate   string `yaml:"revocation-date"`
						RevocationReason int    `yaml:"revocation-reason"`
					}{{
						CertificatePath: "path",
						RevocationDate:  "date",
					}},
					IssuingDistributionPoint: "http://example.com",
				},
			},
			expectedError: "crl-profile.revoked-certificates.revocation-reason is required",
		},
		{
			name: "no issuing distribution point path provided",
			config: crlConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					IssuerCertificatePath string `yaml:"issuer-certificate-path"`
				}{
					IssuerCertificatePath: "path",
				},
				Outputs: struct {
					CRLPath string `yaml:"crl-path"`
				}{
					CRLPath: "path",
				},
				CRLProfile: struct {
					ThisUpdate          string `yaml:"this-update"`
					NextUpdate          string `yaml:"next-update"`
					Number              int64  `yaml:"number"`
					RevokedCertificates []struct {
						CertificatePath  string `yaml:"certificate-path"`
						RevocationDate   string `yaml:"revocation-date"`
						RevocationReason int    `yaml:"revocation-reason"`
					} `yaml:"revoked-certificates"`
					IssuingDistributionPoint string `yaml:"issuing-distribution-point"`
				}{
					ThisUpdate: "this-update",
					NextUpdate: "next-update",
					Number:     1,
					RevokedCertificates: []struct {
						CertificatePath  string `yaml:"certificate-path"`
						RevocationDate   string `yaml:"revocation-date"`
						RevocationReason int    `yaml:"revocation-reason"`
					}{{
						CertificatePath: "path",
						RevocationDate:  "date",
					}},
				},
			},
			expectedError: "crl-profile.issuing-distribution-point is required",
		},
		{
			name: "good",
			config: crlConfig{
				PKCS11: PKCS11SigningConfig{
					Module:       "module",
					SigningLabel: "label",
				},
				Inputs: struct {
					IssuerCertificatePath string `yaml:"issuer-certificate-path"`
				}{
					IssuerCertificatePath: "path",
				},
				Outputs: struct {
					CRLPath string `yaml:"crl-path"`
				}{
					CRLPath: "path",
				},
				CRLProfile: struct {
					ThisUpdate          string `yaml:"this-update"`
					NextUpdate          string `yaml:"next-update"`
					Number              int64  `yaml:"number"`
					RevokedCertificates []struct {
						CertificatePath  string `yaml:"certificate-path"`
						RevocationDate   string `yaml:"revocation-date"`
						RevocationReason int    `yaml:"revocation-reason"`
					} `yaml:"revoked-certificates"`
					IssuingDistributionPoint string `yaml:"issuing-distribution-point"`
				}{
					ThisUpdate: "this-update",
					NextUpdate: "next-update",
					Number:     1,
					RevokedCertificates: []struct {
						CertificatePath  string `yaml:"certificate-path"`
						RevocationDate   string `yaml:"revocation-date"`
						RevocationReason int    `yaml:"revocation-reason"`
					}{{
						CertificatePath:  "path",
						RevocationDate:   "date",
						RevocationReason: 1,
					}},
					IssuingDistributionPoint: "http://example.com",
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.validate()
			if err != nil && err.Error() != tc.expectedError {
				t.Fatalf("Unexpected error, wanted: %q, got: %q", tc.expectedError, err)
			} else if err == nil && tc.expectedError != "" {
				t.Fatalf("validate didn't fail, wanted: %q", err)
			}
		})
	}
}

func TestSignAndWriteNoLintCert(t *testing.T) {
	_, err := signAndWriteCert(nil, nil, nil, nil, nil, "")
	test.AssertError(t, err, "should have failed because no lintCert was provided")
	test.AssertDeepEquals(t, err, fmt.Errorf("linting was not performed prior to issuance"))
}
