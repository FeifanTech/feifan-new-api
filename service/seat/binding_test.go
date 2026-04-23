package seat

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupSeatTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	require.NoError(t, db.AutoMigrate(&model.SeatBinding{}))
}

func TestEncryptDecryptGitHubToken(t *testing.T) {
	common.CryptoSecret = "unit-test-secret"
	enc, err := EncryptGitHubToken("ghp_example_token")
	require.NoError(t, err)
	require.NotEmpty(t, enc)
	dec, err := DecryptGitHubToken(enc)
	require.NoError(t, err)
	require.Equal(t, "ghp_example_token", dec)
}

func TestEncryptDecryptErrorCases(t *testing.T) {
	common.CryptoSecret = "unit-test-secret"
	_, err := EncryptGitHubToken("")
	require.Error(t, err)
	_, err = DecryptGitHubToken("")
	require.Error(t, err)
	_, err = DecryptGitHubToken("%%%invalid-base64%%%")
	require.Error(t, err)
}

func TestBindAndResolveSeat(t *testing.T) {
	setupSeatTestDB(t)
	common.CryptoSecret = "unit-test-secret"

	binding, err := BindSeat("tenant-a", "user-a", "seat-a", "ghp_token_a", "enterprise")
	require.NoError(t, err)
	require.Equal(t, "tenant-a", binding.TenantID)
	require.Equal(t, "seat-a", binding.SeatID)

	resolvedBinding, token, err := ResolveGitHubToken("tenant-a", "user-a")
	require.NoError(t, err)
	require.Equal(t, binding.SeatID, resolvedBinding.SeatID)
	require.Equal(t, "ghp_token_a", token)

	updated, err := BindSeat("tenant-a", "user-b", "seat-b", "ghp_token_b", "business")
	require.NoError(t, err)
	require.Equal(t, "seat-b", updated.SeatID)
	_, updatedToken, err := ResolveGitHubToken("tenant-a", "user-b")
	require.NoError(t, err)
	require.Equal(t, "ghp_token_b", updatedToken)
}

func TestResolveGitHubTokenErrors(t *testing.T) {
	setupSeatTestDB(t)
	common.CryptoSecret = "unit-test-secret"
	_, _, err := ResolveGitHubToken("tenant-missing", "user-missing")
	require.Error(t, err)

	require.NoError(t, model.DB.Create(&model.SeatBinding{
		TenantID:       "t2",
		UserID:         "u2",
		SeatID:         "s2",
		EncryptedToken: "not-valid-base64",
		Status:         "active",
	}).Error)
	_, _, err = ResolveGitHubToken("t2", "u2")
	require.Error(t, err)

	_, err = BindSeat("t3", "u3", "s3", "", "individual")
	require.Error(t, err)
}
