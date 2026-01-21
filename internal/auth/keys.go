package auth

import (
	"crypto/rsa"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// KeyStore manages JWT signing keys by issuer and kid
type KeyStore struct {
	hs256Keys map[string]map[string][]byte         // issuer -> kid -> secret
	rs256Keys map[string]map[string]*rsa.PublicKey // issuer -> kid -> public key
}

// NewKeyStore creates a new KeyStore
func NewKeyStore() *KeyStore {
	return &KeyStore{
		hs256Keys: make(map[string]map[string][]byte),
		rs256Keys: make(map[string]map[string]*rsa.PublicKey),
	}
}

// LoadHS256Key adds an HS256 secret key for an issuer and kid
func (ks *KeyStore) LoadHS256Key(issuer, kid string, secret []byte) {
	if _, ok := ks.hs256Keys[issuer]; !ok {
		ks.hs256Keys[issuer] = make(map[string][]byte)
	}
	ks.hs256Keys[issuer][kid] = secret
}

// LoadRS256Key adds an RS256 public key for an issuer and kid
func (ks *KeyStore) LoadRS256Key(issuer, kid string, publicKeyPEM string) error {
	// CORREÇÃO: Normalizar \n literais para quebras de linha reais
	// Isso resolve o problema quando a chave vem de variáveis de ambiente
	normalizedPEM := strings.ReplaceAll(publicKeyPEM, `\n`, "\n")

	// Também limpar espaços extras no início e fim
	normalizedPEM = strings.TrimSpace(normalizedPEM)

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(normalizedPEM))
	if err != nil {
		return fmt.Errorf("failed to parse RSA public key: %w", err)
	}

	if _, ok := ks.rs256Keys[issuer]; !ok {
		ks.rs256Keys[issuer] = make(map[string]*rsa.PublicKey)
	}
	ks.rs256Keys[issuer][kid] = publicKey
	return nil
}

// GetHS256Key retrieves an HS256 secret for an issuer and kid
func (ks *KeyStore) GetHS256Key(issuer, kid string) ([]byte, bool) {
	if keys, ok := ks.hs256Keys[issuer]; ok {
		if secret, ok := keys[kid]; ok {
			return secret, true
		}
	}
	return nil, false
}

// GetRS256Key retrieves an RS256 public key for an issuer and kid
func (ks *KeyStore) GetRS256Key(issuer, kid string) (*rsa.PublicKey, bool) {
	if keys, ok := ks.rs256Keys[issuer]; ok {
		if publicKey, ok := keys[kid]; ok {
			return publicKey, true
		}
	}
	return nil, false
}
