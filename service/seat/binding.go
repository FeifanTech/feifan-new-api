package seat

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

func EncryptGitHubToken(token string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("empty token")
	}
	key := common.Sha256Raw([]byte(common.CryptoSecret))
	block, err := aes.NewCipher(key)
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
	cipherText := gcm.Seal(nil, nonce, []byte(token), nil)
	result := append(nonce, cipherText...)
	return base64.StdEncoding.EncodeToString(result), nil
}

func DecryptGitHubToken(encrypted string) (string, error) {
	if encrypted == "" {
		return "", fmt.Errorf("empty encrypted token")
	}
	raw, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}
	key := common.Sha256Raw([]byte(common.CryptoSecret))
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize {
		return "", fmt.Errorf("invalid encrypted token")
	}
	nonce, cipherText := raw[:nonceSize], raw[nonceSize:]
	plainText, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", err
	}
	return string(plainText), nil
}

func BindSeat(tenantID, userID, seatID, githubToken, accountType string) (*model.SeatBinding, error) {
	encrypted, err := EncryptGitHubToken(githubToken)
	if err != nil {
		return nil, err
	}
	binding := &model.SeatBinding{
		TenantID:       tenantID,
		UserID:         userID,
		SeatID:         seatID,
		EncryptedToken: encrypted,
		AccountType:    accountType,
		Status:         "active",
	}
	if err = model.DB.Where("tenant_id = ? AND user_id = ?", tenantID, userID).Assign(binding).FirstOrCreate(binding).Error; err != nil {
		return nil, err
	}
	return binding, nil
}

func ResolveGitHubToken(tenantID, userID string) (*model.SeatBinding, string, error) {
	binding, err := model.GetSeatBindingByTenantAndUser(tenantID, userID)
	if err != nil {
		return nil, "", err
	}
	token, err := DecryptGitHubToken(binding.EncryptedToken)
	if err != nil {
		return nil, "", err
	}
	return binding, token, nil
}
