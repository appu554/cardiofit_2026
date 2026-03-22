package crypto

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/nacl/box"
)

// X25519KeyPair holds a Curve25519 key pair used for ABDM health data
// encryption and decryption following the ECDH key agreement protocol.
type X25519KeyPair struct {
	PublicKey  [32]byte
	PrivateKey [32]byte
}

// GenerateKeyPair creates a new random X25519 key pair suitable for
// ABDM health information exchange encryption.
func GenerateKeyPair() (*X25519KeyPair, error) {
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("x25519: key generation failed: %w", err)
	}
	return &X25519KeyPair{
		PublicKey:  *pub,
		PrivateKey: *priv,
	}, nil
}

// Decrypt decrypts a message that was encrypted by a sender using our
// public key. The caller must supply the sender's public key and the
// nonce that was used during encryption.
func (kp *X25519KeyPair) Decrypt(encrypted []byte, senderPublicKey [32]byte, nonce [24]byte) ([]byte, error) {
	plaintext, ok := box.Open(nil, encrypted, &nonce, &senderPublicKey, &kp.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("x25519: decryption failed — invalid ciphertext, key, or nonce")
	}
	return plaintext, nil
}

// Encrypt encrypts plaintext for a recipient identified by their public
// key. A random nonce is generated for each call. The caller must
// transmit the nonce alongside the ciphertext.
func (kp *X25519KeyPair) Encrypt(plaintext []byte, recipientPublicKey [32]byte) (encrypted []byte, nonce [24]byte, err error) {
	if _, err = rand.Read(nonce[:]); err != nil {
		return nil, nonce, fmt.Errorf("x25519: nonce generation failed: %w", err)
	}
	encrypted = box.Seal(nil, plaintext, &nonce, &recipientPublicKey, &kp.PrivateKey)
	return encrypted, nonce, nil
}

// LoadKeyPair constructs an X25519KeyPair from pre-existing key material,
// typically loaded from secure storage or a configuration vault.
func LoadKeyPair(publicKey, privateKey [32]byte) *X25519KeyPair {
	return &X25519KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}
}
