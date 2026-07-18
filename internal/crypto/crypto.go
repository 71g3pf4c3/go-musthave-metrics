package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
)

// LoadPublicKey reads a PEM-encoded X25519 public key from path.
// Expected PEM type: "X25519 PUBLIC KEY", raw 32-byte key in the block.
func LoadPublicKey(path string) (*ecdh.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in %s", path)
	}
	pub, err := ecdh.X25519().NewPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse X25519 public key: %w", err)
	}
	return pub, nil
}

// LoadPrivateKey reads a PEM-encoded X25519 private key from path.
// Expected PEM type: "X25519 PRIVATE KEY", raw 32-byte seed in the block.
func LoadPrivateKey(path string) (*ecdh.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in %s", path)
	}
	priv, err := ecdh.X25519().NewPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse X25519 private key: %w", err)
	}
	return priv, nil
}

// Encrypt encrypts data using X25519 ECDH + AES-256-GCM.
// Wire format: [32 bytes ephemeral public key][12 bytes nonce][ciphertext+tag]
func Encrypt(serverPub *ecdh.PublicKey, data []byte) ([]byte, error) {
	// Generate ephemeral X25519 keypair.
	ephemeral, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ephemeral key: %w", err)
	}

	// ECDH shared secret.
	shared, err := ephemeral.ECDH(serverPub)
	if err != nil {
		return nil, fmt.Errorf("ecdh: %w", err)
	}

	// Derive AES-256 key from shared secret.
	aesKey := sha256.Sum256(shared)

	block, err := aes.NewCipher(aesKey[:])
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, data, nil)

	// [ephemeral pub 32] [nonce 12] [ciphertext+tag]
	result := make([]byte, 0, 32+len(nonce)+len(ciphertext))
	result = append(result, ephemeral.PublicKey().Bytes()...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)
	return result, nil
}

// Decrypt decrypts data using X25519 ECDH + AES-256-GCM.
func Decrypt(serverPriv *ecdh.PrivateKey, data []byte) ([]byte, error) {
	if len(data) < 32+12+16 { // pub + nonce + min tag
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract ephemeral public key.
	ephPubBytes := data[:32]
	ephPub, err := ecdh.X25519().NewPublicKey(ephPubBytes)
	if err != nil {
		return nil, fmt.Errorf("parse ephemeral public key: %w", err)
	}

	// ECDH shared secret.
	shared, err := serverPriv.ECDH(ephPub)
	if err != nil {
		return nil, fmt.Errorf("ecdh: %w", err)
	}

	aesKey := sha256.Sum256(shared)

	block, err := aes.NewCipher(aesKey[:])
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}

	nonce := data[32 : 32+gcm.NonceSize()]
	ciphertext := data[32+gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plaintext, nil
}

// DecryptMiddleware returns HTTP middleware that decrypts request bodies
// using the given X25519 private key.
func DecryptMiddleware(priv *ecdh.PrivateKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			r.Body.Close()
			if err != nil {
				http.Error(w, "failed to read body", http.StatusBadRequest)
				return
			}

			if len(body) == 0 {
				r.Body = io.NopCloser(bytes.NewReader(nil))
				next.ServeHTTP(w, r)
				return
			}

			decrypted, err := Decrypt(priv, body)
			if err != nil {
				http.Error(w, "decryption failed", http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(decrypted))
			r.ContentLength = int64(len(decrypted))
			next.ServeHTTP(w, r)
		})
	}
}
