// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package etls

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/hmac"
	"crypto/rc4"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"
	"runtime"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/sys/cpu"
)

// CipherSuite is a TLS cipher suite. Note that most functions in this package
// accept and expose cipher suite IDs instead of this type.
type CipherSuite struct {
	ID   uint16
	Name string

	// Supported versions is the list of TLS protocol versions that can
	// negotiate this cipher suite.
	SupportedVersions []uint16

	// Insecure is true if the cipher suite has known security issues
	// due to its primitives, design, or implementation.
	Insecure bool
}

var (
	supportedUpToTLS12 = []uint16{VersionTLS10, VersionTLS11, VersionTLS12}
	supportedOnlyTLS12 = []uint16{VersionTLS12}
	supportedOnlyTLS13 = []uint16{VersionTLS13}
)

// CipherSuites returns a list of cipher suites currently implemented by this
// package, excluding those with security issues, which are returned by
// InsecureCipherSuites.
//
// The list is sorted by ID. Note that the default cipher suites selected by
// this package might depend on logic that can't be captured by a static list,
// and might not match those returned by this function.
func CipherSuites() []*CipherSuite {
	return []*CipherSuite{
		{TLS_RSA_WITH_AES_128_CBC_SHA, "TLS_RSA_WITH_AES_128_CBC_SHA", supportedUpToTLS12, false},
		{TLS_RSA_WITH_AES_256_CBC_SHA, "TLS_RSA_WITH_AES_256_CBC_SHA", supportedUpToTLS12, false},
		{TLS_RSA_WITH_AES_128_GCM_SHA256, "TLS_RSA_WITH_AES_128_GCM_SHA256", supportedOnlyTLS12, false},
		{TLS_RSA_WITH_AES_256_GCM_SHA384, "TLS_RSA_WITH_AES_256_GCM_SHA384", supportedOnlyTLS12, false},

		{TLS_DHE_RSA_WITH_AES_128_GCM_SHA256, "TLS_DHE_RSA_WITH_AES_128_GCM_SHA256", supportedOnlyTLS12, false},
		{TLS_DHE_RSA_WITH_AES_256_GCM_SHA384, "TLS_DHE_RSA_WITH_AES_256_GCM_SHA384", supportedOnlyTLS12, false},
		{TLS_DHE_DSS_WITH_AES_128_GCM_SHA256, "TLS_DHE_DSS_WITH_AES_128_GCM_SHA256", supportedOnlyTLS12, false},
		{TLS_DHE_DSS_WITH_AES_256_GCM_SHA384, "TLS_DHE_DSS_WITH_AES_256_GCM_SHA384", supportedOnlyTLS12, false},

		{TLS_AES_128_GCM_SHA256, "TLS_AES_128_GCM_SHA256", supportedOnlyTLS13, false},
		{TLS_AES_256_GCM_SHA384, "TLS_AES_256_GCM_SHA384", supportedOnlyTLS13, false},
		{TLS_CHACHA20_POLY1305_SHA256, "TLS_CHACHA20_POLY1305_SHA256", supportedOnlyTLS13, false},

		{TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA, "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA", supportedUpToTLS12, false},
		{TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA, "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA", supportedUpToTLS12, false},
		{TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA, "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA", supportedUpToTLS12, false},
		{TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA, "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA", supportedUpToTLS12, false},
		{TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256", supportedOnlyTLS12, false},
		{TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384, "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384", supportedOnlyTLS12, false},
		{TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256", supportedOnlyTLS12, false},
		{TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384, "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384", supportedOnlyTLS12, false},
		{TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256, "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256", supportedOnlyTLS12, false},
		{TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256, "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256", supportedOnlyTLS12, false},
	}
}

// InsecureCipherSuites returns a list of cipher suites currently implemented by
// this package and which have security issues.
//
// Most applications should not use the cipher suites in this list, and should
// only use those returned by CipherSuites.
func InsecureCipherSuites() []*CipherSuite {
	// This list includes RC4, CBC_SHA256, and 3DES cipher suites. See
	// cipherSuitesPreferenceOrder for details.
	return []*CipherSuite{
		{TLS_RSA_WITH_NULL_MD5, "TLS_RSA_WITH_NULL_MD5", supportedUpToTLS12, true},
		{TLS_RSA_WITH_NULL_SHA, "TLS_RSA_WITH_NULL_SHA", supportedUpToTLS12, true},
		{TLS_RSA_EXPORT_WITH_RC4_40_MD5, "TLS_RSA_EXPORT_WITH_RC4_40_MD5", supportedUpToTLS12, true},
		{TLS_RSA_WITH_RC4_128_MD5, "TLS_RSA_WITH_RC4_128_MD5", supportedUpToTLS12, true},
		{TLS_RSA_WITH_RC4_128_SHA, "TLS_RSA_WITH_RC4_128_SHA", supportedUpToTLS12, true},
		{TLS_RSA_EXPORT_WITH_RC2_CBC_40_MD5, "TLS_RSA_EXPORT_WITH_RC2_CBC_40_MD5", supportedUpToTLS12, true},
		{TLS_RSA_WITH_IDEA_CBC_SHA, "TLS_RSA_WITH_IDEA_CBC_SHA", supportedUpToTLS12, true},
		{TLS_RSA_EXPORT_WITH_DES40_CBC_SHA, "TLS_RSA_EXPORT_WITH_DES40_CBC_SHA", supportedUpToTLS12, true},
		{TLS_RSA_WITH_DES_CBC_SHA, "TLS_RSA_WITH_DES_CBC_SHA", supportedUpToTLS12, true},
		{TLS_RSA_WITH_3DES_EDE_CBC_SHA, "TLS_RSA_WITH_3DES_EDE_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_DSS_EXPORT_WITH_DES40_CBC_SHA, "TLS_DH_DSS_EXPORT_WITH_DES40_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_DSS_WITH_DES_CBC_SHA, "TLS_DH_DSS_WITH_DES_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_DSS_WITH_3DES_EDE_CBC_SHA, "TLS_DH_DSS_WITH_3DES_EDE_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_RSA_EXPORT_WITH_DES40_CBC_SHA, "TLS_DH_RSA_EXPORT_WITH_DES40_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_RSA_WITH_DES_CBC_SHA, "TLS_DH_RSA_WITH_DES_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_RSA_WITH_3DES_EDE_CBC_SHA, "TLS_DH_RSA_WITH_3DES_EDE_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DHE_DSS_EXPORT_WITH_DES40_CBC_SHA, "TLS_DHE_DSS_EXPORT_WITH_DES40_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DHE_DSS_WITH_DES_CBC_SHA, "TLS_DHE_DSS_WITH_DES_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DHE_DSS_WITH_3DES_EDE_CBC_SHA, "TLS_DHE_DSS_WITH_3DES_EDE_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DHE_RSA_EXPORT_WITH_DES40_CBC_SHA, "TLS_DHE_RSA_EXPORT_WITH_DES40_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DHE_RSA_WITH_DES_CBC_SHA, "TLS_DHE_RSA_WITH_DES_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DHE_RSA_WITH_3DES_EDE_CBC_SHA, "TLS_DHE_RSA_WITH_3DES_EDE_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_anon_EXPORT_WITH_RC4_40_MD5, "TLS_DH_anon_EXPORT_WITH_RC4_40_MD5", supportedUpToTLS12, true},
		{TLS_DH_anon_WITH_RC4_128_MD5, "TLS_DH_anon_WITH_RC4_128_MD5", supportedUpToTLS12, true},
		{TLS_DH_anon_EXPORT_WITH_DES40_CBC_SHA, "TLS_DH_anon_EXPORT_WITH_DES40_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_anon_WITH_DES_CBC_SHA, "TLS_DH_anon_WITH_DES_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_anon_WITH_3DES_EDE_CBC_SHA, "TLS_DH_anon_WITH_3DES_EDE_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_DSS_WITH_AES_128_CBC_SHA, "TLS_DH_DSS_WITH_AES_128_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_RSA_WITH_AES_128_CBC_SHA, "TLS_DH_RSA_WITH_AES_128_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DHE_DSS_WITH_AES_128_CBC_SHA, "TLS_DHE_DSS_WITH_AES_128_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DHE_RSA_WITH_AES_128_CBC_SHA, "TLS_DHE_RSA_WITH_AES_128_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_anon_WITH_AES_128_CBC_SHA, "TLS_DH_anon_WITH_AES_128_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_DSS_WITH_AES_256_CBC_SHA, "TLS_DH_DSS_WITH_AES_256_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_RSA_WITH_AES_256_CBC_SHA, "TLS_DH_RSA_WITH_AES_256_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DHE_DSS_WITH_AES_256_CBC_SHA, "TLS_DHE_DSS_WITH_AES_256_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DHE_RSA_WITH_AES_256_CBC_SHA, "TLS_DHE_RSA_WITH_AES_256_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_anon_WITH_AES_256_CBC_SHA, "TLS_DH_anon_WITH_AES_256_CBC_SHA", supportedUpToTLS12, true},
		{TLS_RSA_WITH_NULL_SHA256, "TLS_RSA_WITH_NULL_SHA256", supportedOnlyTLS12, true},
		{TLS_RSA_WITH_AES_128_CBC_SHA256, "TLS_RSA_WITH_AES_128_CBC_SHA256", supportedOnlyTLS12, true},
		{TLS_RSA_WITH_AES_256_CBC_SHA256, "TLS_RSA_WITH_AES_256_CBC_SHA256", supportedOnlyTLS12, true},
		{TLS_DH_DSS_WITH_AES_128_CBC_SHA256, "TLS_DH_DSS_WITH_AES_128_CBC_SHA256", supportedOnlyTLS12, true},
		{TLS_DH_RSA_WITH_AES_128_CBC_SHA256, "TLS_DH_RSA_WITH_AES_128_CBC_SHA256", supportedOnlyTLS12, true},
		{TLS_DHE_DSS_WITH_AES_128_CBC_SHA256, "TLS_DHE_DSS_WITH_AES_128_CBC_SHA256", supportedOnlyTLS12, true},
		{TLS_RSA_WITH_CAMELLIA_128_CBC_SHA, "TLS_RSA_WITH_CAMELLIA_128_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_DSS_WITH_CAMELLIA_128_CBC_SHA, "TLS_DH_DSS_WITH_CAMELLIA_128_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_RSA_WITH_CAMELLIA_128_CBC_SHA, "TLS_DH_RSA_WITH_CAMELLIA_128_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DHE_DSS_WITH_CAMELLIA_128_CBC_SHA, "TLS_DHE_DSS_WITH_CAMELLIA_128_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DHE_RSA_WITH_CAMELLIA_128_CBC_SHA, "TLS_DHE_RSA_WITH_CAMELLIA_128_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_anon_WITH_CAMELLIA_128_CBC_SHA, "TLS_DH_anon_WITH_CAMELLIA_128_CBC_SHA", supportedUpToTLS12, true},
		{TLS_RSA_EXPORT1024_WITH_RC4_56_MD5, "TLS_RSA_EXPORT1024_WITH_RC4_56_MD5", supportedUpToTLS12, true},
		{TLS_RSA_EXPORT1024_WITH_RC2_CBC_56_MD5, "TLS_RSA_EXPORT1024_WITH_RC2_CBC_56_MD5", supportedUpToTLS12, true},
		{TLS_RSA_EXPORT1024_WITH_DES_CBC_SHA, "TLS_RSA_EXPORT1024_WITH_DES_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DHE_DSS_EXPORT1024_WITH_DES_CBC_SHA, "TLS_DHE_DSS_EXPORT1024_WITH_DES_CBC_SHA", supportedUpToTLS12, true},
		{TLS_RSA_EXPORT1024_WITH_RC4_56_SHA, "TLS_RSA_EXPORT1024_WITH_RC4_56_SHA", supportedUpToTLS12, true},
		{TLS_DHE_DSS_EXPORT1024_WITH_RC4_56_SHA, "TLS_DHE_DSS_EXPORT1024_WITH_RC4_56_SHA", supportedUpToTLS12, true},
		{TLS_DHE_DSS_WITH_RC4_128_SHA, "TLS_DHE_DSS_WITH_RC4_128_SHA", supportedUpToTLS12, true},
		{TLS_DHE_RSA_WITH_AES_128_CBC_SHA256, "TLS_DHE_RSA_WITH_AES_128_CBC_SHA256", supportedOnlyTLS12, true},
		{TLS_DH_DSS_WITH_AES_256_CBC_SHA256, "TLS_DH_DSS_WITH_AES_256_CBC_SHA256", supportedUpToTLS12, true},
		{TLS_DH_RSA_WITH_AES_256_CBC_SHA256, "TLS_DH_RSA_WITH_AES_256_CBC_SHA256", supportedOnlyTLS12, true},
		{TLS_DHE_DSS_WITH_AES_256_CBC_SHA256, "TLS_DHE_DSS_WITH_AES_256_CBC_SHA256", supportedOnlyTLS12, true},
		{TLS_DHE_RSA_WITH_AES_256_CBC_SHA256, "TLS_DHE_RSA_WITH_AES_256_CBC_SHA256", supportedOnlyTLS12, true},
		{TLS_DH_anon_WITH_AES_128_CBC_SHA256, "TLS_DH_anon_WITH_AES_128_CBC_SHA256", supportedOnlyTLS12, true},
		{TLS_DH_anon_WITH_AES_256_CBC_SHA256, "TLS_DH_anon_WITH_AES_256_CBC_SHA256", supportedOnlyTLS12, true},
		{TLS_RSA_WITH_CAMELLIA_256_CBC_SHA, "TLS_RSA_WITH_CAMELLIA_256_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_DSS_WITH_CAMELLIA_256_CBC_SHA, "TLS_DH_DSS_WITH_CAMELLIA_256_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_RSA_WITH_CAMELLIA_256_CBC_SHA, "TLS_DH_RSA_WITH_CAMELLIA_256_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DHE_DSS_WITH_CAMELLIA_256_CBC_SHA, "TLS_DHE_DSS_WITH_CAMELLIA_256_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DHE_RSA_WITH_CAMELLIA_256_CBC_SHA, "TLS_DHE_RSA_WITH_CAMELLIA_256_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_anon_WITH_CAMELLIA_256_CBC_SHA, "TLS_DH_anon_WITH_CAMELLIA_256_CBC_SHA", supportedUpToTLS12, true},
		{TLS_PSK_WITH_RC4_128_SHA, "TLS_PSK_WITH_RC4_128_SHA", supportedUpToTLS12, true},
		{TLS_PSK_WITH_3DES_EDE_CBC_SHA, "TLS_PSK_WITH_3DES_EDE_CBC_SHA", supportedUpToTLS12, true},
		{TLS_PSK_WITH_AES_128_CBC_SHA, "TLS_PSK_WITH_AES_128_CBC_SHA", supportedUpToTLS12, true},
		{TLS_PSK_WITH_AES_256_CBC_SHA, "TLS_PSK_WITH_AES_256_CBC_SHA", supportedUpToTLS12, true},
		{TLS_RSA_PSK_WITH_RC4_128_SHA, "TLS_RSA_PSK_WITH_RC4_128_SHA", supportedUpToTLS12, true},
		{TLS_RSA_PSK_WITH_3DES_EDE_CBC_SHA, "TLS_RSA_PSK_WITH_3DES_EDE_CBC_SHA", supportedUpToTLS12, true},
		{TLS_RSA_PSK_WITH_AES_128_CBC_SHA, "TLS_RSA_PSK_WITH_AES_128_CBC_SHA", supportedUpToTLS12, true},
		{TLS_RSA_PSK_WITH_AES_256_CBC_SHA, "TLS_RSA_PSK_WITH_AES_256_CBC_SHA", supportedUpToTLS12, true},
		{TLS_RSA_WITH_SEED_CBC_SHA, "TLS_RSA_WITH_SEED_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_DSS_WITH_SEED_CBC_SHA, "TLS_DH_DSS_WITH_SEED_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_RSA_WITH_SEED_CBC_SHA, "TLS_DH_RSA_WITH_SEED_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DHE_DSS_WITH_SEED_CBC_SHA, "TLS_DHE_DSS_WITH_SEED_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DHE_RSA_WITH_SEED_CBC_SHA, "TLS_DHE_RSA_WITH_SEED_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_anon_WITH_SEED_CBC_SHA, "TLS_DH_anon_WITH_SEED_CBC_SHA", supportedUpToTLS12, true},
		{TLS_DH_RSA_WITH_AES_128_GCM_SHA256, "TLS_DH_RSA_WITH_AES_128_GCM_SHA256", supportedOnlyTLS12, true},
		{TLS_DH_RSA_WITH_AES_256_GCM_SHA384, "TLS_DH_RSA_WITH_AES_256_GCM_SHA384", supportedOnlyTLS12, true},
		{TLS_DH_DSS_WITH_AES_128_GCM_SHA256, "TLS_DH_DSS_WITH_AES_128_GCM_SHA256", supportedOnlyTLS12, true},
		{TLS_DH_DSS_WITH_AES_256_GCM_SHA384, "TLS_DH_DSS_WITH_AES_256_GCM_SHA384", supportedOnlyTLS12, true},
		{TLS_DH_anon_WITH_AES_128_GCM_SHA256, "TLS_DH_anon_WITH_AES_128_GCM_SHA256", supportedOnlyTLS12, true},
		{TLS_DH_anon_WITH_AES_256_GCM_SHA384, "TLS_DH_anon_WITH_AES_256_GCM_SHA384", supportedOnlyTLS12, true},
		{TLS_RSA_WITH_CAMELLIA_128_CBC_SHA256, "TLS_RSA_WITH_CAMELLIA_128_CBC_SHA256", supportedUpToTLS12, true},
		{TLS_DH_DSS_WITH_CAMELLIA_128_CBC_SHA256, "TLS_DH_DSS_WITH_CAMELLIA_128_CBC_SHA256", supportedUpToTLS12, true},
		{TLS_DH_RSA_WITH_CAMELLIA_128_CBC_SHA256, "TLS_DH_RSA_WITH_CAMELLIA_128_CBC_SHA256", supportedUpToTLS12, true},
		{TLS_DHE_DSS_WITH_CAMELLIA_128_CBC_SHA256, "TLS_DHE_DSS_WITH_CAMELLIA_128_CBC_SHA256", supportedUpToTLS12, true},
		{TLS_DHE_RSA_WITH_CAMELLIA_128_CBC_SHA256, "TLS_DHE_RSA_WITH_CAMELLIA_128_CBC_SHA256", supportedUpToTLS12, true},
		{TLS_DH_anon_WITH_CAMELLIA_128_CBC_SHA256, "TLS_DH_anon_WITH_CAMELLIA_128_CBC_SHA256", supportedUpToTLS12, true},
		{TLS_RSA_WITH_CAMELLIA_256_CBC_SHA256, "TLS_RSA_WITH_CAMELLIA_256_CBC_SHA256", supportedUpToTLS12, true},
		{TLS_DH_DSS_WITH_CAMELLIA_256_CBC_SHA256, "TLS_DH_DSS_WITH_CAMELLIA_256_CBC_SHA256", supportedUpToTLS12, true},
		{TLS_DH_RSA_WITH_CAMELLIA_256_CBC_SHA256, "TLS_DH_RSA_WITH_CAMELLIA_256_CBC_SHA256", supportedUpToTLS12, true},
		{TLS_DHE_DSS_WITH_CAMELLIA_256_CBC_SHA256, "TLS_DHE_DSS_WITH_CAMELLIA_256_CBC_SHA256", supportedUpToTLS12, true},
		{TLS_DHE_RSA_WITH_CAMELLIA_256_CBC_SHA256, "TLS_DHE_RSA_WITH_CAMELLIA_256_CBC_SHA256", supportedUpToTLS12, true},
		{TLS_DH_anon_WITH_CAMELLIA_256_CBC_SHA256, "TLS_DH_anon_WITH_CAMELLIA_256_CBC_SHA256", supportedUpToTLS12, true},
		{TLS_ECDH_ECDSA_WITH_NULL_SHA, "TLS_ECDH_ECDSA_WITH_NULL_SHA", supportedUpToTLS12, true},
		{TLS_ECDH_ECDSA_WITH_RC4_128_SHA, "TLS_ECDH_ECDSA_WITH_RC4_128_SHA", supportedUpToTLS12, true},
		{TLS_ECDH_ECDSA_WITH_3DES_EDE_CBC_SHA, "TLS_ECDH_ECDSA_WITH_3DES_EDE_CBC_SHA", supportedUpToTLS12, true},
		{TLS_ECDH_ECDSA_WITH_AES_128_CBC_SHA, "TLS_ECDH_ECDSA_WITH_AES_128_CBC_SHA", supportedUpToTLS12, true},
		{TLS_ECDH_ECDSA_WITH_AES_256_CBC_SHA, "TLS_ECDH_ECDSA_WITH_AES_256_CBC_SHA", supportedUpToTLS12, true},
		{TLS_ECDHE_ECDSA_WITH_NULL_SHA, "TLS_ECDHE_ECDSA_WITH_NULL_SHA", supportedUpToTLS12, true},
		{TLS_ECDHE_ECDSA_WITH_RC4_128_SHA, "TLS_ECDHE_ECDSA_WITH_RC4_128_SHA", supportedUpToTLS12, true},
		{TLS_ECDHE_ECDSA_WITH_3DES_EDE_CBC_SHA, "TLS_ECDHE_ECDSA_WITH_3DES_EDE_CBC_SHA", supportedUpToTLS12, true},
		{TLS_ECDH_RSA_WITH_NULL_SHA, "TLS_ECDH_RSA_WITH_NULL_SHA", supportedUpToTLS12, true},
		{TLS_ECDH_RSA_WITH_RC4_128_SHA, "TLS_ECDH_RSA_WITH_RC4_128_SHA", supportedUpToTLS12, true},
		{TLS_ECDH_RSA_WITH_3DES_EDE_CBC_SHA, "TLS_ECDH_RSA_WITH_3DES_EDE_CBC_SHA", supportedUpToTLS12, true},
		{TLS_ECDH_RSA_WITH_AES_128_CBC_SHA, "TLS_ECDH_RSA_WITH_AES_128_CBC_SHA", supportedUpToTLS12, true},
		{TLS_ECDH_RSA_WITH_AES_256_CBC_SHA, "TLS_ECDH_RSA_WITH_AES_256_CBC_SHA", supportedUpToTLS12, true},
		{TLS_ECDHE_RSA_WITH_NULL_SHA, "TLS_ECDHE_RSA_WITH_NULL_SHA", supportedUpToTLS12, true},
		{TLS_ECDHE_RSA_WITH_RC4_128_SHA, "TLS_ECDHE_RSA_WITH_RC4_128_SHA", supportedUpToTLS12, true},
		{TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA, "TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA", supportedUpToTLS12, true},
		{TLS_ECDH_anon_WITH_NULL_SHA, "TLS_ECDH_anon_WITH_NULL_SHA", supportedUpToTLS12, true},
		{TLS_ECDH_anon_WITH_RC4_128_SHA, "TLS_ECDH_anon_WITH_RC4_128_SHA", supportedUpToTLS12, true},
		{TLS_ECDH_anon_WITH_3DES_EDE_CBC_SHA, "TLS_ECDH_anon_WITH_3DES_EDE_CBC_SHA", supportedUpToTLS12, true},
		{TLS_ECDH_anon_WITH_AES_128_CBC_SHA, "TLS_ECDH_anon_WITH_AES_128_CBC_SHA", supportedUpToTLS12, true},
		{TLS_ECDH_anon_WITH_AES_256_CBC_SHA, "TLS_ECDH_anon_WITH_AES_256_CBC_SHA", supportedUpToTLS12, true},
		{TLS_SRP_SHA_WITH_3DES_EDE_CBC_SHA, "TLS_SRP_SHA_WITH_3DES_EDE_CBC_SHA", supportedUpToTLS12, true},
		{TLS_SRP_SHA_RSA_WITH_3DES_EDE_CBC_SHA, "TLS_SRP_SHA_RSA_WITH_3DES_EDE_CBC_SHA", supportedUpToTLS12, true},
		{TLS_SRP_SHA_DSS_WITH_3DES_EDE_CBC_SHA, "TLS_SRP_SHA_DSS_WITH_3DES_EDE_CBC_SHA", supportedUpToTLS12, true},
		{TLS_SRP_SHA_WITH_AES_128_CBC_SHA, "TLS_SRP_SHA_WITH_AES_128_CBC_SHA", supportedUpToTLS12, true},
		{TLS_SRP_SHA_RSA_WITH_AES_128_CBC_SHA, "TLS_SRP_SHA_RSA_WITH_AES_128_CBC_SHA", supportedUpToTLS12, true},
		{TLS_SRP_SHA_DSS_WITH_AES_128_CBC_SHA, "TLS_SRP_SHA_DSS_WITH_AES_128_CBC_SHA", supportedUpToTLS12, true},
		{TLS_SRP_SHA_WITH_AES_256_CBC_SHA, "TLS_SRP_SHA_WITH_AES_256_CBC_SHA", supportedUpToTLS12, true},
		{TLS_SRP_SHA_RSA_WITH_AES_256_CBC_SHA, "TLS_SRP_SHA_RSA_WITH_AES_256_CBC_SHA", supportedUpToTLS12, true},
		{TLS_SRP_SHA_DSS_WITH_AES_256_CBC_SHA, "TLS_SRP_SHA_DSS_WITH_AES_256_CBC_SHA", supportedUpToTLS12, true},
		{TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256, "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256", supportedOnlyTLS12, true},
		{TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA384, "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA384", supportedOnlyTLS12, true},
		{TLS_ECDH_ECDSA_WITH_AES_128_CBC_SHA256, "TLS_ECDH_ECDSA_WITH_AES_128_CBC_SHA256", supportedOnlyTLS12, true},
		{TLS_ECDH_ECDSA_WITH_AES_256_CBC_SHA384, "TLS_ECDH_ECDSA_WITH_AES_256_CBC_SHA384", supportedOnlyTLS12, true},
		{TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256, "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256", supportedOnlyTLS12, true},
		{TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA384, "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA384", supportedOnlyTLS12, true},
		{TLS_ECDH_RSA_WITH_AES_128_CBC_SHA256, "TLS_ECDH_RSA_WITH_AES_128_CBC_SHA256", supportedOnlyTLS12, true},
		{TLS_ECDH_RSA_WITH_AES_256_CBC_SHA384, "TLS_ECDH_RSA_WITH_AES_256_CBC_SHA384", supportedOnlyTLS12, true},
		{TLS_ECDH_ECDSA_WITH_AES_128_GCM_SHA256, "TLS_ECDH_ECDSA_WITH_AES_128_GCM_SHA256", supportedOnlyTLS12, true},
		{TLS_ECDH_ECDSA_WITH_AES_256_GCM_SHA384, "TLS_ECDH_ECDSA_WITH_AES_256_GCM_SHA384", supportedOnlyTLS12, true},
		{TLS_ECDH_RSA_WITH_AES_128_GCM_SHA256, "TLS_ECDH_RSA_WITH_AES_128_GCM_SHA256", supportedOnlyTLS12, true},
		{TLS_ECDH_RSA_WITH_AES_256_GCM_SHA384, "TLS_ECDH_RSA_WITH_AES_256_GCM_SHA384", supportedOnlyTLS12, true},
		{TLS_ECDHE_ECDSA_WITH_CAMELLIA_128_CBC_SHA256, "TLS_ECDHE_ECDSA_WITH_CAMELLIA_128_CBC_SHA256", supportedOnlyTLS12, true},
		{TLS_ECDHE_ECDSA_WITH_CAMELLIA_256_CBC_SHA384, "TLS_ECDHE_ECDSA_WITH_CAMELLIA_256_CBC_SHA384", supportedOnlyTLS12, true},
		{TLS_ECDH_ECDSA_WITH_CAMELLIA_128_CBC_SHA256, "TLS_ECDH_ECDSA_WITH_CAMELLIA_128_CBC_SHA256", supportedOnlyTLS12, true},
		{TLS_ECDH_ECDSA_WITH_CAMELLIA_256_CBC_SHA384, "TLS_ECDH_ECDSA_WITH_CAMELLIA_256_CBC_SHA384", supportedOnlyTLS12, true},
		{TLS_ECDHE_RSA_WITH_CAMELLIA_128_CBC_SHA256, "TLS_ECDHE_RSA_WITH_CAMELLIA_128_CBC_SHA256", supportedOnlyTLS12, true},
		{TLS_ECDHE_RSA_WITH_CAMELLIA_256_CBC_SHA384, "TLS_ECDHE_RSA_WITH_CAMELLIA_256_CBC_SHA384", supportedOnlyTLS12, true},
		{TLS_ECDH_RSA_WITH_CAMELLIA_128_CBC_SHA256, "TLS_ECDH_RSA_WITH_CAMELLIA_128_CBC_SHA256", supportedOnlyTLS12, true},
		{TLS_ECDH_RSA_WITH_CAMELLIA_256_CBC_SHA384, "TLS_ECDH_RSA_WITH_CAMELLIA_256_CBC_SHA384", supportedOnlyTLS12, true},
		{TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256_OLD, "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256_OLD", supportedOnlyTLS12, true},
		{TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256_OLD, "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256_OLD", supportedOnlyTLS12, true},
		{TLS_DHE_RSA_WITH_CHACHA20_POLY1305_SHA256_OLD, "TLS_DHE_RSA_WITH_CHACHA20_POLY1305_SHA256_OLD", supportedOnlyTLS12, true},
	}
}

// CipherSuiteName returns the standard name for the passed cipher suite ID
// (e.g. "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"), or a fallback representation
// of the ID value if the cipher suite is not implemented by this package.
func CipherSuiteName(id uint16) string {
	for _, c := range CipherSuites() {
		if c.ID == id {
			return c.Name
		}
	}
	for _, c := range InsecureCipherSuites() {
		if c.ID == id {
			return c.Name
		}
	}
	return fmt.Sprintf("0x%04X", id)
}

const (
	// suiteECDHE indicates that the cipher suite involves elliptic curve
	// Diffie-Hellman. This means that it should only be selected when the
	// client indicates that it supports ECC with a curve and point format
	// that we're happy with.
	suiteECDHE = 1 << iota
	// suiteECSign indicates that the cipher suite involves an ECDSA or
	// EdDSA signature and therefore may only be selected when the server's
	// certificate is ECDSA or EdDSA. If this is not set then the cipher suite
	// is RSA based.
	suiteECSign
	// suiteTLS12 indicates that the cipher suite should only be advertised
	// and accepted when using TLS 1.2.
	suiteTLS12
	// suiteSHA384 indicates that the cipher suite uses SHA384 as the
	// handshake hash.
	suiteSHA384
	// suiteTestingOnly indicates that the cipher suite is only available
	// for testing server compatiblility
	suiteTestingOnly
)

// A cipherSuite is a TLS 1.0–1.2 cipher suite, and defines the key exchange
// mechanism, as well as the cipher+MAC pair or the AEAD.
type cipherSuite struct {
	id uint16
	// the lengths, in bytes, of the key material needed for each component.
	keyLen int
	macLen int
	ivLen  int
	ka     func(version uint16) keyAgreement
	// flags is a bitmask of the suite* values, above.
	flags  int
	cipher func(key, iv []byte, isRead bool) any
	mac    func(key []byte) hash.Hash
	aead   func(key, fixedNonce []byte) aead
}

var cipherSuites = []*cipherSuite{ // TODO: replace with a map, since the order doesn't matter.
	{TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305, 32, 0, 12, ecdheRSAKA, suiteECDHE | suiteTLS12, nil, nil, aeadChaCha20Poly1305},
	{TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305, 32, 0, 12, ecdheECDSAKA, suiteECDHE | suiteECSign | suiteTLS12, nil, nil, aeadChaCha20Poly1305},
	{TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, 16, 0, 4, ecdheRSAKA, suiteECDHE | suiteTLS12, nil, nil, aeadAESGCM},
	{TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, 16, 0, 4, ecdheECDSAKA, suiteECDHE | suiteECSign | suiteTLS12, nil, nil, aeadAESGCM},
	{TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384, 32, 0, 4, ecdheRSAKA, suiteECDHE | suiteTLS12 | suiteSHA384, nil, nil, aeadAESGCM},
	{TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384, 32, 0, 4, ecdheECDSAKA, suiteECDHE | suiteECSign | suiteTLS12 | suiteSHA384, nil, nil, aeadAESGCM},
	{TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256, 16, 32, 16, ecdheRSAKA, suiteECDHE | suiteTLS12, cipherAES, macSHA256, nil},
	{TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA, 16, 20, 16, ecdheRSAKA, suiteECDHE, cipherAES, macSHA1, nil},
	{TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256, 16, 32, 16, ecdheECDSAKA, suiteECDHE | suiteECSign | suiteTLS12, cipherAES, macSHA256, nil},
	{TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA, 16, 20, 16, ecdheECDSAKA, suiteECDHE | suiteECSign, cipherAES, macSHA1, nil},
	{TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA, 32, 20, 16, ecdheRSAKA, suiteECDHE, cipherAES, macSHA1, nil},
	{TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA, 32, 20, 16, ecdheECDSAKA, suiteECDHE | suiteECSign, cipherAES, macSHA1, nil},
	{TLS_RSA_WITH_AES_128_GCM_SHA256, 16, 0, 4, rsaKA, suiteTLS12, nil, nil, aeadAESGCM},
	{TLS_RSA_WITH_AES_256_GCM_SHA384, 32, 0, 4, rsaKA, suiteTLS12 | suiteSHA384, nil, nil, aeadAESGCM},
	{TLS_RSA_WITH_AES_128_CBC_SHA256, 16, 32, 16, rsaKA, suiteTLS12, cipherAES, macSHA256, nil},
	{TLS_RSA_WITH_AES_128_CBC_SHA, 16, 20, 16, rsaKA, 0, cipherAES, macSHA1, nil},
	{TLS_RSA_WITH_AES_256_CBC_SHA, 32, 20, 16, rsaKA, 0, cipherAES, macSHA1, nil},
	{TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA, 24, 20, 8, ecdheRSAKA, suiteECDHE, cipher3DES, macSHA1, nil},
	{TLS_RSA_WITH_3DES_EDE_CBC_SHA, 24, 20, 8, rsaKA, 0, cipher3DES, macSHA1, nil},
	{TLS_RSA_WITH_RC4_128_SHA, 16, 20, 0, rsaKA, 0, cipherRC4, macSHA1, nil},
	{TLS_ECDHE_RSA_WITH_RC4_128_SHA, 16, 20, 0, ecdheRSAKA, suiteECDHE, cipherRC4, macSHA1, nil},
	{TLS_ECDHE_ECDSA_WITH_RC4_128_SHA, 16, 20, 0, ecdheECDSAKA, suiteECDHE | suiteECSign, cipherRC4, macSHA1, nil},
	// start of ciphers for testing purposes
	{TLS_RSA_WITH_NULL_MD5, 0, 0, 0, rsaKA, suiteTestingOnly, nil, nil, nil},
	{TLS_RSA_WITH_NULL_SHA, 0, 0, 0, rsaKA, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_RSA_EXPORT_WITH_RC4_40_MD5, 0, 0, 0, rsaKA, suiteTestingOnly, cipherRC4, nil, nil},
	{TLS_RSA_WITH_RC4_128_MD5, 0, 0, 0, rsaKA, suiteTestingOnly, cipherRC4, nil, nil},
	{TLS_RSA_WITH_RC4_128_SHA, 0, 0, 0, rsaKA, suiteTestingOnly, cipherRC4, macSHA1, nil},
	{TLS_RSA_EXPORT_WITH_RC2_CBC_40_MD5, 0, 0, 0, rsaKA, suiteTestingOnly, nil, nil, nil},
	{TLS_RSA_WITH_IDEA_CBC_SHA, 0, 0, 0, rsaKA, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_RSA_EXPORT_WITH_DES40_CBC_SHA, 0, 0, 0, rsaKA, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_RSA_WITH_DES_CBC_SHA, 0, 0, 0, rsaKA, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_RSA_WITH_3DES_EDE_CBC_SHA, 0, 0, 0, rsaKA, suiteTestingOnly, cipher3DES, macSHA1, nil},
	{TLS_DH_DSS_EXPORT_WITH_DES40_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DH_DSS_WITH_DES_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DH_DSS_WITH_3DES_EDE_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipher3DES, macSHA1, nil},
	{TLS_DH_RSA_EXPORT_WITH_DES40_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DH_RSA_WITH_DES_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DH_RSA_WITH_3DES_EDE_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipher3DES, macSHA1, nil},
	{TLS_DHE_DSS_EXPORT_WITH_DES40_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DHE_DSS_WITH_DES_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DHE_DSS_WITH_3DES_EDE_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipher3DES, macSHA1, nil},
	{TLS_DHE_RSA_EXPORT_WITH_DES40_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DHE_RSA_WITH_DES_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DHE_RSA_WITH_3DES_EDE_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipher3DES, macSHA1, nil},
	{TLS_DH_anon_EXPORT_WITH_RC4_40_MD5, 0, 0, 0, nil, suiteTestingOnly, cipherRC4, nil, nil},
	{TLS_DH_anon_WITH_RC4_128_MD5, 0, 0, 0, nil, suiteTestingOnly, cipherRC4, nil, nil},
	{TLS_DH_anon_EXPORT_WITH_DES40_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DH_anon_WITH_DES_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DH_anon_WITH_3DES_EDE_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipher3DES, macSHA1, nil},
	{TLS_RSA_WITH_AES_128_CBC_SHA, 0, 0, 0, rsaKA, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_DH_DSS_WITH_AES_128_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_DH_RSA_WITH_AES_128_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_DHE_DSS_WITH_AES_128_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_DHE_RSA_WITH_AES_128_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_DH_anon_WITH_AES_128_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_RSA_WITH_AES_256_CBC_SHA, 0, 0, 0, rsaKA, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_DH_DSS_WITH_AES_256_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_DH_RSA_WITH_AES_256_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_DHE_DSS_WITH_AES_256_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_DHE_RSA_WITH_AES_256_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_DH_anon_WITH_AES_256_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_RSA_WITH_NULL_SHA256, 0, 0, 0, rsaKA, suiteTLS12 | suiteTestingOnly, nil, macSHA1, nil},
	{TLS_RSA_WITH_AES_256_CBC_SHA256, 0, 0, 0, rsaKA, suiteTLS12 | suiteTestingOnly, cipherAES, macSHA256, nil},
	{TLS_DH_DSS_WITH_AES_128_CBC_SHA256, 0, 0, 0, nil, suiteTLS12 | suiteTestingOnly, cipherAES, macSHA256, nil},
	{TLS_DH_RSA_WITH_AES_128_CBC_SHA256, 0, 0, 0, nil, suiteTLS12 | suiteTestingOnly, cipherAES, macSHA256, nil},
	{TLS_DHE_DSS_WITH_AES_128_CBC_SHA256, 0, 0, 0, nil, suiteTLS12 | suiteTestingOnly, cipherAES, macSHA256, nil},
	{TLS_RSA_WITH_CAMELLIA_128_CBC_SHA, 0, 0, 0, rsaKA, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DH_DSS_WITH_CAMELLIA_128_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DH_RSA_WITH_CAMELLIA_128_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DHE_DSS_WITH_CAMELLIA_128_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DHE_RSA_WITH_CAMELLIA_128_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DH_anon_WITH_CAMELLIA_128_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_RSA_EXPORT1024_WITH_RC4_56_MD5, 0, 0, 0, rsaKA, suiteTestingOnly, cipherRC4, nil, nil},
	{TLS_RSA_EXPORT1024_WITH_RC2_CBC_56_MD5, 0, 0, 0, rsaKA, suiteTestingOnly, nil, nil, nil},
	{TLS_RSA_EXPORT1024_WITH_DES_CBC_SHA, 0, 0, 0, rsaKA, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DHE_DSS_EXPORT1024_WITH_DES_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_RSA_EXPORT1024_WITH_RC4_56_SHA, 0, 0, 0, rsaKA, suiteTestingOnly, cipherRC4, macSHA1, nil},
	{TLS_DHE_DSS_EXPORT1024_WITH_RC4_56_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherRC4, macSHA1, nil},
	{TLS_DHE_DSS_WITH_RC4_128_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherRC4, macSHA1, nil},
	{TLS_DHE_RSA_WITH_AES_128_CBC_SHA256, 0, 0, 0, nil, suiteTLS12 | suiteTestingOnly, cipherAES, macSHA256, nil},
	{TLS_DH_DSS_WITH_AES_256_CBC_SHA256, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA256, nil},
	{TLS_DH_RSA_WITH_AES_256_CBC_SHA256, 0, 0, 0, nil, suiteTLS12 | suiteTestingOnly, cipherAES, macSHA256, nil},
	{TLS_DHE_DSS_WITH_AES_256_CBC_SHA256, 0, 0, 0, nil, suiteTLS12 | suiteTestingOnly, cipherAES, macSHA256, nil},
	{TLS_DHE_RSA_WITH_AES_256_CBC_SHA256, 0, 0, 0, nil, suiteTLS12 | suiteTestingOnly, cipherAES, macSHA256, nil},
	{TLS_DH_anon_WITH_AES_128_CBC_SHA256, 0, 0, 0, nil, suiteTLS12 | suiteTestingOnly, cipherAES, macSHA256, nil},
	{TLS_DH_anon_WITH_AES_256_CBC_SHA256, 0, 0, 0, nil, suiteTLS12 | suiteTestingOnly, cipherAES, macSHA256, nil},
	{TLS_RSA_WITH_CAMELLIA_256_CBC_SHA, 0, 0, 0, rsaKA, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DH_DSS_WITH_CAMELLIA_256_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DH_RSA_WITH_CAMELLIA_256_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DHE_DSS_WITH_CAMELLIA_256_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DHE_RSA_WITH_CAMELLIA_256_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DH_anon_WITH_CAMELLIA_256_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_PSK_WITH_RC4_128_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherRC4, macSHA1, nil},
	{TLS_PSK_WITH_3DES_EDE_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipher3DES, macSHA1, nil},
	{TLS_PSK_WITH_AES_128_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_PSK_WITH_AES_256_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_RSA_PSK_WITH_RC4_128_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherRC4, macSHA1, nil},
	{TLS_RSA_PSK_WITH_3DES_EDE_CBC_SHA, 0, 0, 0, rsaKA, suiteTestingOnly, cipher3DES, macSHA1, nil},
	{TLS_RSA_PSK_WITH_AES_128_CBC_SHA, 0, 0, 0, rsaKA, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_RSA_PSK_WITH_AES_256_CBC_SHA, 0, 0, 0, rsaKA, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_RSA_WITH_SEED_CBC_SHA, 0, 0, 0, rsaKA, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DH_DSS_WITH_SEED_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DH_RSA_WITH_SEED_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DHE_DSS_WITH_SEED_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DHE_RSA_WITH_SEED_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DH_anon_WITH_SEED_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_DHE_RSA_WITH_AES_128_GCM_SHA256, 0, 0, 0, nil, suiteTLS12 | suiteTestingOnly, cipherAES, nil, aeadAESGCM},
	{TLS_DHE_RSA_WITH_AES_256_GCM_SHA384, 0, 0, 0, nil, suiteTLS12 | suiteSHA384 | suiteTestingOnly, cipherAES, nil, aeadAESGCM},
	{TLS_DH_RSA_WITH_AES_128_GCM_SHA256, 0, 0, 0, nil, suiteTLS12 | suiteTestingOnly, cipherAES, nil, aeadAESGCM},
	{TLS_DH_RSA_WITH_AES_256_GCM_SHA384, 0, 0, 0, nil, suiteTLS12 | suiteSHA384 | suiteTestingOnly, cipherAES, nil, aeadAESGCM},
	{TLS_DHE_DSS_WITH_AES_128_GCM_SHA256, 0, 0, 0, nil, suiteTLS12 | suiteTestingOnly, cipherAES, nil, aeadAESGCM},
	{TLS_DHE_DSS_WITH_AES_256_GCM_SHA384, 0, 0, 0, nil, suiteTLS12 | suiteSHA384 | suiteTestingOnly, cipherAES, nil, aeadAESGCM},
	{TLS_DH_DSS_WITH_AES_128_GCM_SHA256, 0, 0, 0, nil, suiteTLS12 | suiteTestingOnly, cipherAES, nil, aeadAESGCM},
	{TLS_DH_DSS_WITH_AES_256_GCM_SHA384, 0, 0, 0, nil, suiteTLS12 | suiteSHA384 | suiteTestingOnly, cipherAES, nil, aeadAESGCM},
	{TLS_DH_anon_WITH_AES_128_GCM_SHA256, 0, 0, 0, nil, suiteTLS12 | suiteTestingOnly, cipherAES, nil, aeadAESGCM},
	{TLS_DH_anon_WITH_AES_256_GCM_SHA384, 0, 0, 0, nil, suiteTLS12 | suiteSHA384 | suiteTestingOnly, cipherAES, nil, aeadAESGCM},
	{TLS_RSA_WITH_CAMELLIA_128_CBC_SHA256, 0, 0, 0, rsaKA, suiteTestingOnly, nil, macSHA256, nil},
	{TLS_DH_DSS_WITH_CAMELLIA_128_CBC_SHA256, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA256, nil},
	{TLS_DH_RSA_WITH_CAMELLIA_128_CBC_SHA256, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA256, nil},
	{TLS_DHE_DSS_WITH_CAMELLIA_128_CBC_SHA256, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA256, nil},
	{TLS_DHE_RSA_WITH_CAMELLIA_128_CBC_SHA256, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA256, nil},
	{TLS_DH_anon_WITH_CAMELLIA_128_CBC_SHA256, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA256, nil},
	{TLS_RSA_WITH_CAMELLIA_256_CBC_SHA256, 0, 0, 0, rsaKA, suiteTestingOnly, nil, macSHA256, nil},
	{TLS_DH_DSS_WITH_CAMELLIA_256_CBC_SHA256, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA256, nil},
	{TLS_DH_RSA_WITH_CAMELLIA_256_CBC_SHA256, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA256, nil},
	{TLS_DHE_DSS_WITH_CAMELLIA_256_CBC_SHA256, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA256, nil},
	{TLS_DHE_RSA_WITH_CAMELLIA_256_CBC_SHA256, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA256, nil},
	{TLS_DH_anon_WITH_CAMELLIA_256_CBC_SHA256, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA256, nil},
	{TLS_ECDH_ECDSA_WITH_NULL_SHA, 0, 0, 0, nil, suiteECSign | suiteTestingOnly, nil, macSHA1, nil},
	{TLS_ECDH_ECDSA_WITH_RC4_128_SHA, 0, 0, 0, nil, suiteECSign | suiteTestingOnly, cipherRC4, macSHA1, nil},
	{TLS_ECDH_ECDSA_WITH_3DES_EDE_CBC_SHA, 0, 0, 0, nil, suiteECSign | suiteTestingOnly, cipher3DES, macSHA1, nil},
	{TLS_ECDH_ECDSA_WITH_AES_128_CBC_SHA, 0, 0, 0, nil, suiteECSign | suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_ECDH_ECDSA_WITH_AES_256_CBC_SHA, 0, 0, 0, nil, suiteECSign | suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_ECDHE_ECDSA_WITH_NULL_SHA, 0, 0, 0, ecdheECDSAKA, suiteECDHE | suiteECSign | suiteTestingOnly, nil, macSHA1, nil},
	{TLS_ECDHE_ECDSA_WITH_RC4_128_SHA, 0, 0, 0, ecdheECDSAKA, suiteECDHE | suiteECSign | suiteTestingOnly, cipherRC4, macSHA1, nil},
	{TLS_ECDHE_ECDSA_WITH_3DES_EDE_CBC_SHA, 0, 0, 0, ecdheECDSAKA, suiteECDHE | suiteECSign | suiteTestingOnly, cipher3DES, macSHA1, nil},
	{TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA, 0, 0, 0, ecdheECDSAKA, suiteECDHE | suiteECSign | suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA, 0, 0, 0, ecdheECDSAKA, suiteECDHE | suiteECSign | suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_ECDH_RSA_WITH_NULL_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_ECDH_RSA_WITH_RC4_128_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherRC4, macSHA1, nil},
	{TLS_ECDH_RSA_WITH_3DES_EDE_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipher3DES, macSHA1, nil},
	{TLS_ECDH_RSA_WITH_AES_128_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_ECDH_RSA_WITH_AES_256_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_ECDHE_RSA_WITH_NULL_SHA, 0, 0, 0, ecdheRSAKA, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_ECDHE_RSA_WITH_RC4_128_SHA, 0, 0, 0, ecdheRSAKA, suiteTestingOnly, cipherRC4, macSHA1, nil},
	{TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA, 0, 0, 0, ecdheRSAKA, suiteTestingOnly, cipher3DES, macSHA1, nil},
	{TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA, 0, 0, 0, ecdheRSAKA, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA, 0, 0, 0, ecdheRSAKA, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_ECDH_anon_WITH_NULL_SHA, 0, 0, 0, nil, suiteTestingOnly, nil, macSHA1, nil},
	{TLS_ECDH_anon_WITH_RC4_128_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherRC4, macSHA1, nil},
	{TLS_ECDH_anon_WITH_3DES_EDE_CBC_SHA, 0, 0, 8, nil, suiteTestingOnly, cipher3DES, macSHA1, nil},
	{TLS_ECDH_anon_WITH_AES_128_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_ECDH_anon_WITH_AES_256_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_SRP_SHA_WITH_3DES_EDE_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipher3DES, macSHA1, nil},
	{TLS_SRP_SHA_RSA_WITH_3DES_EDE_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipher3DES, macSHA1, nil},
	{TLS_SRP_SHA_DSS_WITH_3DES_EDE_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipher3DES, macSHA1, nil},
	{TLS_SRP_SHA_WITH_AES_128_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_SRP_SHA_RSA_WITH_AES_128_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_SRP_SHA_DSS_WITH_AES_128_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_SRP_SHA_WITH_AES_256_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_SRP_SHA_RSA_WITH_AES_256_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_SRP_SHA_DSS_WITH_AES_256_CBC_SHA, 0, 0, 0, nil, suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA384, 0, 0, 0, ecdheECDSAKA, suiteECDHE | suiteECSign | suiteTLS12 | suiteSHA384 | suiteTestingOnly, cipherAES, nil, nil},
	{TLS_ECDH_ECDSA_WITH_AES_128_CBC_SHA256, 0, 0, 0, nil, suiteECSign | suiteTLS12 | suiteTestingOnly, cipherAES, macSHA256, nil},
	{TLS_ECDH_ECDSA_WITH_AES_256_CBC_SHA384, 0, 0, 0, nil, suiteECSign | suiteTLS12 | suiteSHA384 | suiteTestingOnly, cipherAES, nil, nil},
	{TLS_ECDH_RSA_WITH_AES_128_CBC_SHA256, 0, 0, 0, nil, suiteTLS12 | suiteTestingOnly, cipherAES, macSHA1, nil},
	{TLS_ECDH_RSA_WITH_AES_256_CBC_SHA384, 0, 0, 0, nil, suiteTLS12 | suiteSHA384 | suiteTestingOnly, cipherAES, nil, nil},
	{TLS_ECDH_ECDSA_WITH_AES_128_GCM_SHA256, 0, 0, 0, nil, suiteECSign | suiteTLS12 | suiteTestingOnly, nil, nil, aeadAESGCM},
	{TLS_ECDH_ECDSA_WITH_AES_256_GCM_SHA384, 0, 0, 0, nil, suiteECSign | suiteTLS12 | suiteSHA384 | suiteTestingOnly, nil, nil, aeadAESGCM},
	{TLS_ECDH_RSA_WITH_AES_128_GCM_SHA256, 0, 0, 0, nil, suiteTLS12 | suiteTestingOnly, nil, nil, aeadAESGCM},
	{TLS_ECDH_RSA_WITH_AES_256_GCM_SHA384, 0, 0, 0, nil, suiteTLS12 | suiteSHA384 | suiteTestingOnly, nil, nil, aeadAESGCM},
	{TLS_ECDHE_ECDSA_WITH_CAMELLIA_128_CBC_SHA256, 0, 0, 0, ecdheECDSAKA, suiteECDHE | suiteECSign | suiteTLS12 | suiteTestingOnly, nil, macSHA256, nil},
	{TLS_ECDHE_ECDSA_WITH_CAMELLIA_256_CBC_SHA384, 0, 0, 0, ecdheECDSAKA, suiteECDHE | suiteECSign | suiteTLS12 | suiteSHA384 | suiteTestingOnly, nil, nil, nil},
	{TLS_ECDH_ECDSA_WITH_CAMELLIA_128_CBC_SHA256, 0, 0, 0, nil, suiteECSign | suiteTLS12 | suiteTestingOnly, nil, macSHA256, nil},
	{TLS_ECDH_ECDSA_WITH_CAMELLIA_256_CBC_SHA384, 0, 0, 0, nil, suiteECSign | suiteTLS12 | suiteSHA384 | suiteTestingOnly, nil, nil, nil},
	{TLS_ECDHE_RSA_WITH_CAMELLIA_128_CBC_SHA256, 0, 0, 0, ecdheRSAKA, suiteTLS12 | suiteTestingOnly, nil, macSHA256, nil},
	{TLS_ECDHE_RSA_WITH_CAMELLIA_256_CBC_SHA384, 0, 0, 0, ecdheRSAKA, suiteTLS12 | suiteSHA384 | suiteTestingOnly, nil, nil, nil},
	{TLS_ECDH_RSA_WITH_CAMELLIA_128_CBC_SHA256, 0, 0, 0, nil, suiteTLS12 | suiteTestingOnly, nil, macSHA256, nil},
	{TLS_ECDH_RSA_WITH_CAMELLIA_256_CBC_SHA384, 0, 0, 0, nil, suiteTLS12 | suiteSHA384 | suiteTestingOnly, nil, nil, nil},
	{TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256_OLD, 0, 0, 0, ecdheRSAKA, suiteECDHE | suiteTLS12 | suiteTestingOnly, nil, nil, aeadChaCha20Poly1305},
	{TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256_OLD, 0, 0, 0, ecdheECDSAKA, suiteECDHE | suiteECSign | suiteTLS12 | suiteTestingOnly, nil, nil, aeadChaCha20Poly1305},
	{TLS_DHE_RSA_WITH_CHACHA20_POLY1305_SHA256_OLD, 0, 0, 0, nil, suiteTLS12 | suiteTestingOnly, nil, nil, aeadChaCha20Poly1305},
}

// selectCipherSuite returns the first TLS 1.0–1.2 cipher suite from ids which
// is also in supportedIDs and passes the ok filter.
func selectCipherSuite(ids, supportedIDs []uint16, ok func(*cipherSuite) bool) *cipherSuite {
	for _, id := range ids {
		candidate := cipherSuiteByID(id)
		if candidate == nil || !ok(candidate) {
			continue
		}

		for _, suppID := range supportedIDs {
			if id == suppID {
				return candidate
			}
		}
	}
	return nil
}

// A cipherSuiteTLS13 defines only the pair of the AEAD algorithm and hash
// algorithm to be used with HKDF. See RFC 8446, Appendix B.4.
type cipherSuiteTLS13 struct {
	id     uint16
	keyLen int
	aead   func(key, fixedNonce []byte) aead
	hash   crypto.Hash
}

var cipherSuitesTLS13 = []*cipherSuiteTLS13{ // TODO: replace with a map.
	{TLS_AES_128_GCM_SHA256, 16, aeadAESGCMTLS13, crypto.SHA256},
	{TLS_CHACHA20_POLY1305_SHA256, 32, aeadChaCha20Poly1305, crypto.SHA256},
	{TLS_AES_256_GCM_SHA384, 32, aeadAESGCMTLS13, crypto.SHA384},
	// {TLS_AES_128_CCM_SHA256, 16, nil, crypto.SHA256},
	// {TLS_AES_128_CCM_8_SHA256, 16, nil, crypto.SHA256},
}

// cipherSuitesPreferenceOrder is the order in which we'll select (on the
// server) or advertise (on the client) TLS 1.0–1.2 cipher suites.
//
// Cipher suites are filtered but not reordered based on the application and
// peer's preferences, meaning we'll never select a suite lower in this list if
// any higher one is available. This makes it more defensible to keep weaker
// cipher suites enabled, especially on the server side where we get the last
// word, since there are no known downgrade attacks on cipher suites selection.
//
// The list is sorted by applying the following priority rules, stopping at the
// first (most important) applicable one:
//
//   - Anything else comes before RC4
//
//     RC4 has practically exploitable biases. See https://www.rc4nomore.com.
//
//   - Anything else comes before CBC_SHA256
//
//     SHA-256 variants of the CBC ciphersuites don't implement any Lucky13
//     countermeasures. See http://www.isg.rhul.ac.uk/tls/Lucky13.html and
//     https://www.imperialviolet.org/2013/02/04/luckythirteen.html.
//
//   - Anything else comes before 3DES
//
//     3DES has 64-bit blocks, which makes it fundamentally susceptible to
//     birthday attacks. See https://sweet32.info.
//
//   - ECDHE comes before anything else
//
//     Once we got the broken stuff out of the way, the most important
//     property a cipher suite can have is forward secrecy. We don't
//     implement FFDHE, so that means ECDHE.
//
//   - AEADs come before CBC ciphers
//
//     Even with Lucky13 countermeasures, MAC-then-Encrypt CBC cipher suites
//     are fundamentally fragile, and suffered from an endless sequence of
//     padding oracle attacks. See https://eprint.iacr.org/2015/1129,
//     https://www.imperialviolet.org/2014/12/08/poodleagain.html, and
//     https://blog.cloudflare.com/yet-another-padding-oracle-in-openssl-cbc-ciphersuites/.
//
//   - AES comes before ChaCha20
//
//     When AES hardware is available, AES-128-GCM and AES-256-GCM are faster
//     than ChaCha20Poly1305.
//
//     When AES hardware is not available, AES-128-GCM is one or more of: much
//     slower, way more complex, and less safe (because not constant time)
//     than ChaCha20Poly1305.
//
//     We use this list if we think both peers have AES hardware, and
//     cipherSuitesPreferenceOrderNoAES otherwise.
//
//   - AES-128 comes before AES-256
//
//     The only potential advantages of AES-256 are better multi-target
//     margins, and hypothetical post-quantum properties. Neither apply to
//     TLS, and AES-256 is slower due to its four extra rounds (which don't
//     contribute to the advantages above).
//
//   - ECDSA comes before RSA
//
//     The relative order of ECDSA and RSA cipher suites doesn't matter,
//     as they depend on the certificate. Pick one to get a stable order.
var cipherSuitesPreferenceOrder = []uint16{
	// AEADs w/ ECDHE
	TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384, TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305, TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,

	// CBC w/ ECDHE
	TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA, TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
	TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA, TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,

	// AEADs w/o ECDHE
	TLS_RSA_WITH_AES_128_GCM_SHA256,
	TLS_RSA_WITH_AES_256_GCM_SHA384,

	// CBC w/o ECDHE
	TLS_RSA_WITH_AES_128_CBC_SHA,
	TLS_RSA_WITH_AES_256_CBC_SHA,

	// 3DES
	TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
	TLS_RSA_WITH_3DES_EDE_CBC_SHA,

	// CBC_SHA256
	TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256, TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
	TLS_RSA_WITH_AES_128_CBC_SHA256,

	// RC4
	TLS_ECDHE_ECDSA_WITH_RC4_128_SHA, TLS_ECDHE_RSA_WITH_RC4_128_SHA,
	TLS_RSA_WITH_RC4_128_SHA,
}

var cipherSuitesPreferenceOrderNoAES = []uint16{
	// ChaCha20Poly1305
	TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305, TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,

	// AES-GCM w/ ECDHE
	TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384, TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,

	// The rest of cipherSuitesPreferenceOrder.
	TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA, TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
	TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA, TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
	TLS_RSA_WITH_AES_128_GCM_SHA256,
	TLS_RSA_WITH_AES_256_GCM_SHA384,
	TLS_RSA_WITH_AES_128_CBC_SHA,
	TLS_RSA_WITH_AES_256_CBC_SHA,
	TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
	TLS_RSA_WITH_3DES_EDE_CBC_SHA,
	TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256, TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
	TLS_RSA_WITH_AES_128_CBC_SHA256,
	TLS_ECDHE_ECDSA_WITH_RC4_128_SHA, TLS_ECDHE_RSA_WITH_RC4_128_SHA,
	TLS_RSA_WITH_RC4_128_SHA,
}

// disabledCipherSuites are not used unless explicitly listed in
// Config.CipherSuites. They MUST be at the end of cipherSuitesPreferenceOrder.
var disabledCipherSuites = []uint16{
	// CBC_SHA256
	TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256, TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
	TLS_RSA_WITH_AES_128_CBC_SHA256,

	// RC4
	TLS_ECDHE_ECDSA_WITH_RC4_128_SHA, TLS_ECDHE_RSA_WITH_RC4_128_SHA,
	TLS_RSA_WITH_RC4_128_SHA,
}

var (
	defaultCipherSuitesLen = len(cipherSuitesPreferenceOrder) - len(disabledCipherSuites)
	defaultCipherSuites    = cipherSuitesPreferenceOrder[:defaultCipherSuitesLen]
)

// defaultCipherSuitesTLS13 is also the preference order, since there are no
// disabled by default TLS 1.3 cipher suites. The same AES vs ChaCha20 logic as
// cipherSuitesPreferenceOrder applies.
var defaultCipherSuitesTLS13 = []uint16{
	TLS_AES_128_GCM_SHA256,
	TLS_AES_256_GCM_SHA384,
	TLS_CHACHA20_POLY1305_SHA256,
}

var defaultCipherSuitesTLS13NoAES = []uint16{
	TLS_CHACHA20_POLY1305_SHA256,
	TLS_AES_128_GCM_SHA256,
	TLS_AES_256_GCM_SHA384,
}

var (
	hasGCMAsmAMD64 = cpu.X86.HasAES && cpu.X86.HasPCLMULQDQ
	hasGCMAsmARM64 = cpu.ARM64.HasAES && cpu.ARM64.HasPMULL
	// Keep in sync with crypto/aes/cipher_s390x.go.
	hasGCMAsmS390X = cpu.S390X.HasAES && cpu.S390X.HasAESCBC && cpu.S390X.HasAESCTR &&
		(cpu.S390X.HasGHASH || cpu.S390X.HasAESGCM)

	hasAESGCMHardwareSupport = runtime.GOARCH == "amd64" && hasGCMAsmAMD64 ||
		runtime.GOARCH == "arm64" && hasGCMAsmARM64 ||
		runtime.GOARCH == "s390x" && hasGCMAsmS390X
)

var aesgcmCiphers = map[uint16]bool{
	// TLS 1.2
	TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256:   true,
	TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384:   true,
	TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256: true,
	TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384: true,
	// TLS 1.3
	TLS_AES_128_GCM_SHA256: true,
	TLS_AES_256_GCM_SHA384: true,
}

// var nonAESGCMAEADCiphers = map[uint16]bool{
// 	// TLS 1.2
// 	TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305:   true,
// 	TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305: true,
// 	// TLS 1.3
// 	TLS_CHACHA20_POLY1305_SHA256: true,
// }

// aesgcmPreferred returns whether the first known cipher in the preference list
// is an AES-GCM cipher, implying the peer has hardware support for it.
func aesgcmPreferred(ciphers []uint16) bool {
	for _, cID := range ciphers {
		if c := cipherSuiteByID(cID); c != nil {
			return aesgcmCiphers[cID]
		}
		if c := cipherSuiteTLS13ByID(cID); c != nil {
			return aesgcmCiphers[cID]
		}
	}
	return false
}

func cipherRC4(key, iv []byte, isRead bool) any {
	cipher, _ := rc4.NewCipher(key)
	return cipher
}

func cipher3DES(key, iv []byte, isRead bool) any {

	block, _ := des.NewTripleDESCipher(key)
	if isRead {
		return cipher.NewCBCDecrypter(block, iv)
	}
	return cipher.NewCBCEncrypter(block, iv)
}

func cipherAES(key, iv []byte, isRead bool) any {
	block, _ := aes.NewCipher(key)
	if isRead {
		return cipher.NewCBCDecrypter(block, iv)
	}
	return cipher.NewCBCEncrypter(block, iv)
}

// macSHA1 returns a SHA-1 based constant time MAC.
func macSHA1(key []byte) hash.Hash {
	h := sha1.New
	// The BoringCrypto SHA1 does not have a constant-time
	// checksum function, so don't try to use it.
	// if !boring.Enabled {
	// 	h = newConstantTimeHash(h)
	// }
	return hmac.New(h, key)
}

// macSHA256 returns a SHA-256 based MAC. This is only supported in TLS 1.2 and
// is currently only used in disabled-by-default cipher suites.
func macSHA256(key []byte) hash.Hash {
	return hmac.New(sha256.New, key)
}

type aead interface {
	cipher.AEAD

	// explicitNonceLen returns the number of bytes of explicit nonce
	// included in each record. This is eight for older AEADs and
	// zero for modern ones.
	explicitNonceLen() int
}

const (
	aeadNonceLength   = 12
	noncePrefixLength = 4
)

// prefixNonceAEAD wraps an AEAD and prefixes a fixed portion of the nonce to
// each call.
type prefixNonceAEAD struct {
	// nonce contains the fixed part of the nonce in the first four bytes.
	nonce [aeadNonceLength]byte
	aead  cipher.AEAD
}

func (f *prefixNonceAEAD) NonceSize() int        { return aeadNonceLength - noncePrefixLength }
func (f *prefixNonceAEAD) Overhead() int         { return f.aead.Overhead() }
func (f *prefixNonceAEAD) explicitNonceLen() int { return f.NonceSize() }

func (f *prefixNonceAEAD) Seal(out, nonce, plaintext, additionalData []byte) []byte {
	copy(f.nonce[4:], nonce)
	return f.aead.Seal(out, f.nonce[:], plaintext, additionalData)
}

func (f *prefixNonceAEAD) Open(out, nonce, ciphertext, additionalData []byte) ([]byte, error) {
	copy(f.nonce[4:], nonce)
	return f.aead.Open(out, f.nonce[:], ciphertext, additionalData)
}

// xoredNonceAEAD wraps an AEAD by XORing in a fixed pattern to the nonce
// before each call.
type xorNonceAEAD struct {
	nonceMask [aeadNonceLength]byte
	aead      cipher.AEAD
}

func (f *xorNonceAEAD) NonceSize() int        { return 8 } // 64-bit sequence number
func (f *xorNonceAEAD) Overhead() int         { return f.aead.Overhead() }
func (f *xorNonceAEAD) explicitNonceLen() int { return 0 }

func (f *xorNonceAEAD) Seal(out, nonce, plaintext, additionalData []byte) []byte {
	for i, b := range nonce {
		f.nonceMask[4+i] ^= b
	}
	result := f.aead.Seal(out, f.nonceMask[:], plaintext, additionalData)
	for i, b := range nonce {
		f.nonceMask[4+i] ^= b
	}

	return result
}

func (f *xorNonceAEAD) Open(out, nonce, ciphertext, additionalData []byte) ([]byte, error) {
	for i, b := range nonce {
		f.nonceMask[4+i] ^= b
	}
	result, err := f.aead.Open(out, f.nonceMask[:], ciphertext, additionalData)
	for i, b := range nonce {
		f.nonceMask[4+i] ^= b
	}

	return result, err
}

func aeadAESGCM(key, noncePrefix []byte) aead {
	if len(noncePrefix) != noncePrefixLength {
		panic("tls: internal error: wrong nonce length")
	}
	aes, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	var aead cipher.AEAD
	// if boring.Enabled {
	// 	aead, err = boring.NewGCMTLS(aes)
	// } else {
	// 	boring.Unreachable()
	// 	aead, err = cipher.NewGCM(aes)
	// }
	aead, err = cipher.NewGCM(aes)
	if err != nil {
		panic(err)
	}

	ret := &prefixNonceAEAD{aead: aead}
	copy(ret.nonce[:], noncePrefix)
	return ret
}

func aeadAESGCMTLS13(key, nonceMask []byte) aead {
	if len(nonceMask) != aeadNonceLength {
		panic("tls: internal error: wrong nonce length")
	}
	aes, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	aead, err := cipher.NewGCM(aes)
	if err != nil {
		panic(err)
	}

	ret := &xorNonceAEAD{aead: aead}
	copy(ret.nonceMask[:], nonceMask)
	return ret
}

func aeadChaCha20Poly1305(key, nonceMask []byte) aead {
	if len(nonceMask) != aeadNonceLength {
		panic("tls: internal error: wrong nonce length")
	}
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		panic(err)
	}

	ret := &xorNonceAEAD{aead: aead}
	copy(ret.nonceMask[:], nonceMask)
	return ret
}

// type constantTimeHash interface {
// 	hash.Hash
// 	ConstantTimeSum(b []byte) []byte
// }

// cthWrapper wraps any hash.Hash that implements ConstantTimeSum, and replaces
// with that all calls to Sum. It's used to obtain a ConstantTimeSum-based HMAC.
// type cthWrapper struct {
// 	h constantTimeHash
// }

// func (c *cthWrapper) Size() int                   { return c.h.Size() }
// func (c *cthWrapper) BlockSize() int              { return c.h.BlockSize() }
// func (c *cthWrapper) Reset()                      { c.h.Reset() }
// func (c *cthWrapper) Write(p []byte) (int, error) { return c.h.Write(p) }
// func (c *cthWrapper) Sum(b []byte) []byte         { return c.h.ConstantTimeSum(b) }

// func newConstantTimeHash(h func() hash.Hash) func() hash.Hash {
// 	// boring.Unreachable()
// 	return func() hash.Hash {
// 		return &cthWrapper{h().(constantTimeHash)}
// 	}
// }

// tls10MAC implements the TLS 1.0 MAC function. RFC 2246, Section 6.2.3.
func tls10MAC(h hash.Hash, out, seq, header, data, extra []byte) []byte {
	h.Reset()
	h.Write(seq)
	h.Write(header)
	h.Write(data)
	res := h.Sum(out)
	if extra != nil {
		h.Write(extra)
	}
	return res
}

func rsaKA(version uint16) keyAgreement {
	return rsaKeyAgreement{}
}

func ecdheECDSAKA(version uint16) keyAgreement {
	return &ecdheKeyAgreement{
		isRSA:   false,
		version: version,
	}
}

func ecdheRSAKA(version uint16) keyAgreement {
	return &ecdheKeyAgreement{
		isRSA:   true,
		version: version,
	}
}

// mutualCipherSuite returns a cipherSuite given a list of supported
// ciphersuites and the id requested by the peer.
func mutualCipherSuite(have []uint16, want uint16) *cipherSuite {
	for _, id := range have {
		if id == want {
			return cipherSuiteByID(id)
		}
	}
	return nil
}

func cipherSuiteByID(id uint16) *cipherSuite {
	for _, cipherSuite := range cipherSuites {
		if cipherSuite.id == id {
			return cipherSuite
		}
	}
	return nil
}

func mutualCipherSuiteTLS13(have []uint16, want uint16) *cipherSuiteTLS13 {
	for _, id := range have {
		if id == want {
			return cipherSuiteTLS13ByID(id)
		}
	}
	return nil
}

func cipherSuiteTLS13ByID(id uint16) *cipherSuiteTLS13 {
	for _, cipherSuite := range cipherSuitesTLS13 {
		if cipherSuite.id == id {
			return cipherSuite
		}
	}
	return nil
}

// A list of cipher suite IDs that are, or have been, implemented by this
// package.
//
// See https://www.iana.org/assignments/tls-parameters/tls-parameters.xml
const (
	// TLS 1.0 - 1.2 cipher suites.
	TLS_RSA_WITH_NULL_MD5                             uint16 = 0x0001
	TLS_RSA_WITH_NULL_SHA                             uint16 = 0x0002
	TLS_RSA_EXPORT_WITH_RC4_40_MD5                    uint16 = 0x0003
	TLS_RSA_WITH_RC4_128_MD5                          uint16 = 0x0004
	TLS_RSA_WITH_RC4_128_SHA                          uint16 = 0x0005
	TLS_RSA_EXPORT_WITH_RC2_CBC_40_MD5                uint16 = 0x0006
	TLS_RSA_WITH_IDEA_CBC_SHA                         uint16 = 0x0007
	TLS_RSA_EXPORT_WITH_DES40_CBC_SHA                 uint16 = 0x0008
	TLS_RSA_WITH_DES_CBC_SHA                          uint16 = 0x0009
	TLS_RSA_WITH_3DES_EDE_CBC_SHA                     uint16 = 0x000a
	TLS_DH_DSS_EXPORT_WITH_DES40_CBC_SHA              uint16 = 0x000b
	TLS_DH_DSS_WITH_DES_CBC_SHA                       uint16 = 0x000c
	TLS_DH_DSS_WITH_3DES_EDE_CBC_SHA                  uint16 = 0x000d
	TLS_DH_RSA_EXPORT_WITH_DES40_CBC_SHA              uint16 = 0x000e
	TLS_DH_RSA_WITH_DES_CBC_SHA                       uint16 = 0x000f
	TLS_DH_RSA_WITH_3DES_EDE_CBC_SHA                  uint16 = 0x0010
	TLS_DHE_DSS_EXPORT_WITH_DES40_CBC_SHA             uint16 = 0x0011
	TLS_DHE_DSS_WITH_DES_CBC_SHA                      uint16 = 0x0012
	TLS_DHE_DSS_WITH_3DES_EDE_CBC_SHA                 uint16 = 0x0013
	TLS_DHE_RSA_EXPORT_WITH_DES40_CBC_SHA             uint16 = 0x0014
	TLS_DHE_RSA_WITH_DES_CBC_SHA                      uint16 = 0x0015
	TLS_DHE_RSA_WITH_3DES_EDE_CBC_SHA                 uint16 = 0x0016
	TLS_DH_anon_EXPORT_WITH_RC4_40_MD5                uint16 = 0x0017
	TLS_DH_anon_WITH_RC4_128_MD5                      uint16 = 0x0018
	TLS_DH_anon_EXPORT_WITH_DES40_CBC_SHA             uint16 = 0x0019
	TLS_DH_anon_WITH_DES_CBC_SHA                      uint16 = 0x001a
	TLS_DH_anon_WITH_3DES_EDE_CBC_SHA                 uint16 = 0x001b
	TLS_RSA_WITH_AES_128_CBC_SHA                      uint16 = 0x002f
	TLS_DH_DSS_WITH_AES_128_CBC_SHA                   uint16 = 0x0030
	TLS_DH_RSA_WITH_AES_128_CBC_SHA                   uint16 = 0x0031
	TLS_DHE_DSS_WITH_AES_128_CBC_SHA                  uint16 = 0x0032
	TLS_DHE_RSA_WITH_AES_128_CBC_SHA                  uint16 = 0x0033
	TLS_DH_anon_WITH_AES_128_CBC_SHA                  uint16 = 0x0034
	TLS_RSA_WITH_AES_256_CBC_SHA                      uint16 = 0x0035
	TLS_DH_DSS_WITH_AES_256_CBC_SHA                   uint16 = 0x0036
	TLS_DH_RSA_WITH_AES_256_CBC_SHA                   uint16 = 0x0037
	TLS_DHE_DSS_WITH_AES_256_CBC_SHA                  uint16 = 0x0038
	TLS_DHE_RSA_WITH_AES_256_CBC_SHA                  uint16 = 0x0039
	TLS_DH_anon_WITH_AES_256_CBC_SHA                  uint16 = 0x003a
	TLS_RSA_WITH_NULL_SHA256                          uint16 = 0x003b
	TLS_RSA_WITH_AES_128_CBC_SHA256                   uint16 = 0x003c
	TLS_RSA_WITH_AES_256_CBC_SHA256                   uint16 = 0x003d
	TLS_DH_DSS_WITH_AES_128_CBC_SHA256                uint16 = 0x003e
	TLS_DH_RSA_WITH_AES_128_CBC_SHA256                uint16 = 0x003f
	TLS_DHE_DSS_WITH_AES_128_CBC_SHA256               uint16 = 0x0040
	TLS_RSA_WITH_CAMELLIA_128_CBC_SHA                 uint16 = 0x0041
	TLS_DH_DSS_WITH_CAMELLIA_128_CBC_SHA              uint16 = 0x0042
	TLS_DH_RSA_WITH_CAMELLIA_128_CBC_SHA              uint16 = 0x0043
	TLS_DHE_DSS_WITH_CAMELLIA_128_CBC_SHA             uint16 = 0x0044
	TLS_DHE_RSA_WITH_CAMELLIA_128_CBC_SHA             uint16 = 0x0045
	TLS_DH_anon_WITH_CAMELLIA_128_CBC_SHA             uint16 = 0x0046
	TLS_RSA_EXPORT1024_WITH_RC4_56_MD5                uint16 = 0x0060
	TLS_RSA_EXPORT1024_WITH_RC2_CBC_56_MD5            uint16 = 0x0061
	TLS_RSA_EXPORT1024_WITH_DES_CBC_SHA               uint16 = 0x0062
	TLS_DHE_DSS_EXPORT1024_WITH_DES_CBC_SHA           uint16 = 0x0063
	TLS_RSA_EXPORT1024_WITH_RC4_56_SHA                uint16 = 0x0064
	TLS_DHE_DSS_EXPORT1024_WITH_RC4_56_SHA            uint16 = 0x0065
	TLS_DHE_DSS_WITH_RC4_128_SHA                      uint16 = 0x0066
	TLS_DHE_RSA_WITH_AES_128_CBC_SHA256               uint16 = 0x0067
	TLS_DH_DSS_WITH_AES_256_CBC_SHA256                uint16 = 0x0068
	TLS_DH_RSA_WITH_AES_256_CBC_SHA256                uint16 = 0x0069
	TLS_DHE_DSS_WITH_AES_256_CBC_SHA256               uint16 = 0x006a
	TLS_DHE_RSA_WITH_AES_256_CBC_SHA256               uint16 = 0x006b
	TLS_DH_anon_WITH_AES_128_CBC_SHA256               uint16 = 0x006c
	TLS_DH_anon_WITH_AES_256_CBC_SHA256               uint16 = 0x006d
	TLS_RSA_WITH_CAMELLIA_256_CBC_SHA                 uint16 = 0x0084
	TLS_DH_DSS_WITH_CAMELLIA_256_CBC_SHA              uint16 = 0x0085
	TLS_DH_RSA_WITH_CAMELLIA_256_CBC_SHA              uint16 = 0x0086
	TLS_DHE_DSS_WITH_CAMELLIA_256_CBC_SHA             uint16 = 0x0087
	TLS_DHE_RSA_WITH_CAMELLIA_256_CBC_SHA             uint16 = 0x0088
	TLS_DH_anon_WITH_CAMELLIA_256_CBC_SHA             uint16 = 0x0089
	TLS_PSK_WITH_RC4_128_SHA                          uint16 = 0x008a
	TLS_PSK_WITH_3DES_EDE_CBC_SHA                     uint16 = 0x008b
	TLS_PSK_WITH_AES_128_CBC_SHA                      uint16 = 0x008c
	TLS_PSK_WITH_AES_256_CBC_SHA                      uint16 = 0x008d
	TLS_RSA_PSK_WITH_RC4_128_SHA                      uint16 = 0x0092
	TLS_RSA_PSK_WITH_3DES_EDE_CBC_SHA                 uint16 = 0x0093
	TLS_RSA_PSK_WITH_AES_128_CBC_SHA                  uint16 = 0x0094
	TLS_RSA_PSK_WITH_AES_256_CBC_SHA                  uint16 = 0x0095
	TLS_RSA_WITH_SEED_CBC_SHA                         uint16 = 0x0096
	TLS_DH_DSS_WITH_SEED_CBC_SHA                      uint16 = 0x0097
	TLS_DH_RSA_WITH_SEED_CBC_SHA                      uint16 = 0x0098
	TLS_DHE_DSS_WITH_SEED_CBC_SHA                     uint16 = 0x0099
	TLS_DHE_RSA_WITH_SEED_CBC_SHA                     uint16 = 0x009a
	TLS_DH_anon_WITH_SEED_CBC_SHA                     uint16 = 0x009b
	TLS_RSA_WITH_AES_128_GCM_SHA256                   uint16 = 0x009c
	TLS_RSA_WITH_AES_256_GCM_SHA384                   uint16 = 0x009d
	TLS_DHE_RSA_WITH_AES_128_GCM_SHA256               uint16 = 0x009e
	TLS_DHE_RSA_WITH_AES_256_GCM_SHA384               uint16 = 0x009f
	TLS_DH_RSA_WITH_AES_128_GCM_SHA256                uint16 = 0x00a0
	TLS_DH_RSA_WITH_AES_256_GCM_SHA384                uint16 = 0x00a1
	TLS_DHE_DSS_WITH_AES_128_GCM_SHA256               uint16 = 0x00a2
	TLS_DHE_DSS_WITH_AES_256_GCM_SHA384               uint16 = 0x00a3
	TLS_DH_DSS_WITH_AES_128_GCM_SHA256                uint16 = 0x00a4
	TLS_DH_DSS_WITH_AES_256_GCM_SHA384                uint16 = 0x00a5
	TLS_DH_anon_WITH_AES_128_GCM_SHA256               uint16 = 0x00a6
	TLS_DH_anon_WITH_AES_256_GCM_SHA384               uint16 = 0x00a7
	TLS_RSA_WITH_CAMELLIA_128_CBC_SHA256              uint16 = 0x00ba
	TLS_DH_DSS_WITH_CAMELLIA_128_CBC_SHA256           uint16 = 0x00bb
	TLS_DH_RSA_WITH_CAMELLIA_128_CBC_SHA256           uint16 = 0x00bc
	TLS_DHE_DSS_WITH_CAMELLIA_128_CBC_SHA256          uint16 = 0x00bd
	TLS_DHE_RSA_WITH_CAMELLIA_128_CBC_SHA256          uint16 = 0x00be
	TLS_DH_anon_WITH_CAMELLIA_128_CBC_SHA256          uint16 = 0x00bf
	TLS_RSA_WITH_CAMELLIA_256_CBC_SHA256              uint16 = 0x00c0
	TLS_DH_DSS_WITH_CAMELLIA_256_CBC_SHA256           uint16 = 0x00c1
	TLS_DH_RSA_WITH_CAMELLIA_256_CBC_SHA256           uint16 = 0x00c2
	TLS_DHE_DSS_WITH_CAMELLIA_256_CBC_SHA256          uint16 = 0x00c3
	TLS_DHE_RSA_WITH_CAMELLIA_256_CBC_SHA256          uint16 = 0x00c4
	TLS_DH_anon_WITH_CAMELLIA_256_CBC_SHA256          uint16 = 0x00c5
	TLS_ECDH_ECDSA_WITH_NULL_SHA                      uint16 = 0xc001
	TLS_ECDH_ECDSA_WITH_RC4_128_SHA                   uint16 = 0xc002
	TLS_ECDH_ECDSA_WITH_3DES_EDE_CBC_SHA              uint16 = 0xc003
	TLS_ECDH_ECDSA_WITH_AES_128_CBC_SHA               uint16 = 0xc004
	TLS_ECDH_ECDSA_WITH_AES_256_CBC_SHA               uint16 = 0xc005
	TLS_ECDHE_ECDSA_WITH_NULL_SHA                     uint16 = 0xc006
	TLS_ECDHE_ECDSA_WITH_RC4_128_SHA                  uint16 = 0xc007
	TLS_ECDHE_ECDSA_WITH_3DES_EDE_CBC_SHA             uint16 = 0xc008
	TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA              uint16 = 0xc009
	TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA              uint16 = 0xc00a
	TLS_ECDH_RSA_WITH_NULL_SHA                        uint16 = 0xc00b
	TLS_ECDH_RSA_WITH_RC4_128_SHA                     uint16 = 0xc00c
	TLS_ECDH_RSA_WITH_3DES_EDE_CBC_SHA                uint16 = 0xc00d
	TLS_ECDH_RSA_WITH_AES_128_CBC_SHA                 uint16 = 0xc00e
	TLS_ECDH_RSA_WITH_AES_256_CBC_SHA                 uint16 = 0xc00f
	TLS_ECDHE_RSA_WITH_NULL_SHA                       uint16 = 0xc010
	TLS_ECDHE_RSA_WITH_RC4_128_SHA                    uint16 = 0xc011
	TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA               uint16 = 0xc012
	TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA                uint16 = 0xc013
	TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA                uint16 = 0xc014
	TLS_ECDH_anon_WITH_NULL_SHA                       uint16 = 0xc015
	TLS_ECDH_anon_WITH_RC4_128_SHA                    uint16 = 0xc016
	TLS_ECDH_anon_WITH_3DES_EDE_CBC_SHA               uint16 = 0xc017
	TLS_ECDH_anon_WITH_AES_128_CBC_SHA                uint16 = 0xc018
	TLS_ECDH_anon_WITH_AES_256_CBC_SHA                uint16 = 0xc019
	TLS_SRP_SHA_WITH_3DES_EDE_CBC_SHA                 uint16 = 0xc01a
	TLS_SRP_SHA_RSA_WITH_3DES_EDE_CBC_SHA             uint16 = 0xc01b
	TLS_SRP_SHA_DSS_WITH_3DES_EDE_CBC_SHA             uint16 = 0xc01c
	TLS_SRP_SHA_WITH_AES_128_CBC_SHA                  uint16 = 0xc01d
	TLS_SRP_SHA_RSA_WITH_AES_128_CBC_SHA              uint16 = 0xc01e
	TLS_SRP_SHA_DSS_WITH_AES_128_CBC_SHA              uint16 = 0xc01f
	TLS_SRP_SHA_WITH_AES_256_CBC_SHA                  uint16 = 0xc020
	TLS_SRP_SHA_RSA_WITH_AES_256_CBC_SHA              uint16 = 0xc021
	TLS_SRP_SHA_DSS_WITH_AES_256_CBC_SHA              uint16 = 0xc022
	TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256           uint16 = 0xc023
	TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA384           uint16 = 0xc024
	TLS_ECDH_ECDSA_WITH_AES_128_CBC_SHA256            uint16 = 0xc025
	TLS_ECDH_ECDSA_WITH_AES_256_CBC_SHA384            uint16 = 0xc026
	TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256             uint16 = 0xc027
	TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA384             uint16 = 0xc028
	TLS_ECDH_RSA_WITH_AES_128_CBC_SHA256              uint16 = 0xc029
	TLS_ECDH_RSA_WITH_AES_256_CBC_SHA384              uint16 = 0xc02a
	TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256           uint16 = 0xc02b
	TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384           uint16 = 0xc02c
	TLS_ECDH_ECDSA_WITH_AES_128_GCM_SHA256            uint16 = 0xc02d
	TLS_ECDH_ECDSA_WITH_AES_256_GCM_SHA384            uint16 = 0xc02e
	TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256             uint16 = 0xc02f
	TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384             uint16 = 0xc030
	TLS_ECDH_RSA_WITH_AES_128_GCM_SHA256              uint16 = 0xc031
	TLS_ECDH_RSA_WITH_AES_256_GCM_SHA384              uint16 = 0xc032
	TLS_ECDHE_ECDSA_WITH_CAMELLIA_128_CBC_SHA256      uint16 = 0xc072
	TLS_ECDHE_ECDSA_WITH_CAMELLIA_256_CBC_SHA384      uint16 = 0xc073
	TLS_ECDH_ECDSA_WITH_CAMELLIA_128_CBC_SHA256       uint16 = 0xc074
	TLS_ECDH_ECDSA_WITH_CAMELLIA_256_CBC_SHA384       uint16 = 0xc075
	TLS_ECDHE_RSA_WITH_CAMELLIA_128_CBC_SHA256        uint16 = 0xc076
	TLS_ECDHE_RSA_WITH_CAMELLIA_256_CBC_SHA384        uint16 = 0xc077
	TLS_ECDH_RSA_WITH_CAMELLIA_128_CBC_SHA256         uint16 = 0xc078
	TLS_ECDH_RSA_WITH_CAMELLIA_256_CBC_SHA384         uint16 = 0xc079
	TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256_OLD   uint16 = 0xcc13 // draft version of ChaCha ciphers
	TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256_OLD uint16 = 0xcc14 // draft version of ChaCha ciphers
	TLS_DHE_RSA_WITH_CHACHA20_POLY1305_SHA256_OLD     uint16 = 0xcc15 // draft version of ChaCha ciphers
	TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256       uint16 = 0xcca8
	TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256     uint16 = 0xcca9

	// TLS 1.3 cipher suites.
	TLS_AES_128_GCM_SHA256       uint16 = 0x1301
	TLS_AES_256_GCM_SHA384       uint16 = 0x1302
	TLS_CHACHA20_POLY1305_SHA256 uint16 = 0x1303
	TLS_AES_128_CCM_SHA256       uint16 = 0x1304
	TLS_AES_128_CCM_8_SHA256     uint16 = 0x1305

	// TLS_FALLBACK_SCSV isn't a standard cipher suite but an indicator
	// that the client is doing version fallback. See RFC 7507.
	TLS_FALLBACK_SCSV uint16 = 0x5600

	// Legacy names for the corresponding cipher suites with the correct _SHA256
	// suffix, retained for backward compatibility.
	TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305   = TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256
	TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305 = TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256
)
