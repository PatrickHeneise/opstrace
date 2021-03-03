// Copyright 2021 Opstrace, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package authenticator

import (
	"crypto/rsa"
	// Disable warning for using sha1: a cryptographically secure hash is not
	// needed here: an cluster admin generates and manages key pairs, and only
	// trusted admin is supposed to add or remove keys from the key set.
	//nolint: gosec
	"crypto/sha1"
	"encoding/hex"
	"os"
	"strings"

	json "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

// Use a single key for now. Further down the road there should be support
// for multiple public keys, each identified by a key id.
var authtokenVerificationPubKeyFallback *rsa.PublicKey

// map for key set. Key of map: key ID (sha1 of PEM bytes?)
var authtokenVerificationPubKeys map[string]*rsa.PublicKey

func keyIDfromPEM(pemstring string) string {
	//nolint: gosec // a strong hash is not needed here, md5 would also do it.
	h := sha1.New()
	// Trim leading and trailing whitespace from PEM string, take underlying
	// bytes and build the SHA1 hash from it -- represent it in hex form as
	// a string.
	h.Write([]byte(strings.TrimSpace(pemstring)))
	return hex.EncodeToString(h.Sum(nil))
}

/*
Read set of public keys from environment variable
API_AUTHTOKEN_VERIFICATION_PUBKEYS. If key deserialization fails, log an error
and exit the process with a non-zero exit code.

*/
func ReadKeySetJSONFromEnvOrCrash() {
	data, present := os.LookupEnv("API_AUTHTOKEN_VERIFICATION_PUBKEYS")

	if !present {
		log.Errorf("API_AUTHTOKEN_VERIFICATION_PUBKEYS must be set. Exit.")
		os.Exit(1)
	}

	if data == "" {
		log.Errorf("API_AUTHTOKEN_VERIFICATION_PUBKEYS must not be empty. Exit.")
		os.Exit(1)
	}

	log.Infof("API_AUTHTOKEN_VERIFICATION_PUBKEYS value: %s", data)

	// Declared an empty interface
	var keys map[string]string
	// Unmarshal or Decode the JSON to the interface.
	jerr := json.Unmarshal([]byte(data), &keys)
	if jerr != nil {
		log.Errorf("error while JSON-parsing API_AUTHTOKEN_VERIFICATION_PUBKEYS: %s", jerr)
		os.Exit(1)
	}

	// Initialize map
	authtokenVerificationPubKeys = make(map[string]*rsa.PublicKey)

	for kidFromConfig, pemstring := range keys {
		log.Infof("parse PEM bytes for key with ID %s", kidFromConfig)
		// We're interested in processing the (PEM) bytes underneath the string
		// value.
		pubkey, err := deserializeRSAPubKeyFromPEMBytes([]byte(pemstring))
		if err != nil {
			log.Errorf("%s", err)
			os.Exit(1)
		}

		kidFromKey := keyIDfromPEM(pemstring)
		log.Infof("calculated key ID from PEM data: %s", kidFromKey)
		if kidFromKey != kidFromConfig {
			log.Errorf("key ID from config (%s) does not match key ID calculated from key (%s)", kidFromConfig, kidFromKey)
			os.Exit(1)
		}
		log.Infof("key ID confirmed")

		log.Infof(
			"Parsed RSA public key. Modulus size: %d bits",
			pubkey.Size()*8)

		// Store in global authenticator key set.
		authtokenVerificationPubKeys[kidFromConfig] = pubkey
	}
}

func LegacyReadAuthTokenVerificationKeyFromEnv() {
	// Upgrade consideration: support for one pubkey -> support for multiple
	// pubkeys: legacy auth tokens don't encode a key id. Read legacy env var,
	// do not fail if not set. If set: store key as fallback, for tokens that
	// do not encode a key id. That way, an Opstrace cluster state is supported
	// that is in mixed state (with valid auth tokens of both types).
	data, present := os.LookupEnv("API_AUTHTOKEN_VERIFICATION_PUBKEY")

	if !present {
		log.Infof("API_AUTHTOKEN_VERIFICATION_PUBKEY is not set")
		return
	}

	if data == "" {
		log.Infof("API_AUTHTOKEN_VERIFICATION_PUBKEY is empty")
		return
	}

	log.Infof("API_AUTHTOKEN_VERIFICATION_PUBKEY value: %s", data)

	// `os.LookupEnv` returns a string. We're interested in processing the
	// bytes underneath it.
	pubkey, err := deserializeRSAPubKeyFromPEMBytes([]byte(data))
	if err != nil {
		log.Errorf("%s", err)
		os.Exit(1)
	}

	// Set module global for subsequent consumption by authenticator logic.
	authtokenVerificationPubKeyFallback = pubkey
	log.Infof("Successfully read RSA public key from legacy env var API_AUTHTOKEN_VERIFICATION_PUBKEY")
}