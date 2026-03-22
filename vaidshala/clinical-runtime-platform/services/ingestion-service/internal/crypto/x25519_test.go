package crypto

import (
	"bytes"
	"testing"
	"time"
)

func TestGenerateKeyPair(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	var zeroKey [32]byte
	if kp.PublicKey == zeroKey {
		t.Error("public key is all zeros")
	}
	if kp.PrivateKey == zeroKey {
		t.Error("private key is all zeros")
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	sender, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("sender GenerateKeyPair() error: %v", err)
	}
	receiver, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("receiver GenerateKeyPair() error: %v", err)
	}

	plaintext := []byte(`{"resourceType":"Bundle","entry":[]}`)

	encrypted, nonce, err := sender.Encrypt(plaintext, receiver.PublicKey)
	if err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}

	decrypted, err := receiver.Decrypt(encrypted, sender.PublicKey, nonce)
	if err != nil {
		t.Fatalf("Decrypt() error: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("round-trip mismatch:\n  want: %s\n  got:  %s", plaintext, decrypted)
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	sender, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("sender GenerateKeyPair() error: %v", err)
	}
	receiver, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("receiver GenerateKeyPair() error: %v", err)
	}
	wrongKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("wrongKey GenerateKeyPair() error: %v", err)
	}

	plaintext := []byte("secret health data")
	encrypted, nonce, err := sender.Encrypt(plaintext, receiver.PublicKey)
	if err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}

	_, err = wrongKey.Decrypt(encrypted, sender.PublicKey, nonce)
	if err == nil {
		t.Error("Decrypt() with wrong key should have failed, but succeeded")
	}
}

func TestVerifyConsentArtifact_Valid(t *testing.T) {
	artifact := ConsentArtifact{
		ConsentID:    "consent-001",
		PatientID:    "patient-001",
		HIURequestID: "req-001",
		Purpose:      "CAREMGT",
		HITypes:      []string{"OPConsultation", "DiagnosticReport"},
		DateFrom:     time.Now().Add(-24 * time.Hour),
		DateTo:       time.Now().Add(24 * time.Hour),
		ExpiresAt:    time.Now().Add(72 * time.Hour),
		Signature:    "MEUCIQD...base64sig",
		Status:       "GRANTED",
	}

	if err := VerifyConsentArtifact(artifact); err != nil {
		t.Errorf("VerifyConsentArtifact() unexpected error: %v", err)
	}
}

func TestVerifyConsentArtifact_Expired(t *testing.T) {
	artifact := ConsentArtifact{
		ConsentID:    "consent-002",
		PatientID:    "patient-001",
		HIURequestID: "req-002",
		Purpose:      "CAREMGT",
		HITypes:      []string{"OPConsultation"},
		DateFrom:     time.Now().Add(-48 * time.Hour),
		DateTo:       time.Now().Add(-24 * time.Hour),
		ExpiresAt:    time.Now().Add(-1 * time.Hour),
		Signature:    "MEUCIQD...base64sig",
		Status:       "GRANTED",
	}

	err := VerifyConsentArtifact(artifact)
	if err == nil {
		t.Error("VerifyConsentArtifact() should fail for expired consent")
	}
}

func TestVerifyConsentArtifact_Revoked(t *testing.T) {
	artifact := ConsentArtifact{
		ConsentID:    "consent-003",
		PatientID:    "patient-001",
		HIURequestID: "req-003",
		Purpose:      "CAREMGT",
		HITypes:      []string{"OPConsultation"},
		DateFrom:     time.Now().Add(-24 * time.Hour),
		DateTo:       time.Now().Add(24 * time.Hour),
		ExpiresAt:    time.Now().Add(72 * time.Hour),
		Signature:    "MEUCIQD...base64sig",
		Status:       "REVOKED",
	}

	err := VerifyConsentArtifact(artifact)
	if err == nil {
		t.Error("VerifyConsentArtifact() should fail for REVOKED consent")
	}
}
