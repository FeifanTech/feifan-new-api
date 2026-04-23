package seat

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

func normalizeTenantID(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "default"
	}
	return v
}

func tokenCipherKey() []byte {
	key := []byte(common.CryptoSecret)
	if len(key) >= 32 {
		return key[:32]
	}
	expanded := make([]byte, 32)
	copy(expanded, key)
	for i := len(key); i < 32; i++ {
		expanded[i] = byte(i)
	}
	return expanded
}

func encryptToken(token string) (string, error) {
	block, err := aes.NewCipher(tokenCipherKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	cipherText := gcm.Seal(nonce, nonce, []byte(token), nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

func decryptToken(cipherToken string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(cipherToken)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(tokenCipherKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("invalid encrypted token")
	}
	nonce, payload := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, payload, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func BindSeat(tenantID string, userID int, seatID, githubToken, accountType string) error {
	encToken, err := encryptToken(githubToken)
	if err != nil {
		return err
	}
	return model.UpsertSeatBinding(&model.SeatBinding{
		TenantID:       normalizeTenantID(tenantID),
		UserID:         userID,
		SeatID:         seatID,
		EncryptedToken: encToken,
		AccountType:    accountType,
		Status:         "active",
	})
}

func ResolveGitHubToken(tenantID string, userID int) (string, *model.SeatBinding, error) {
	binding, err := model.GetSeatBindingByTenantAndUser(normalizeTenantID(tenantID), userID)
	if err != nil {
		return "", nil, err
	}
	token, err := decryptToken(binding.EncryptedToken)
	if err != nil {
		return "", nil, err
	}
	_ = model.UpdateSeatBindingLastUsed(binding.Id)
	return token, binding, nil
}
