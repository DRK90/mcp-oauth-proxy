package token

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/obot-platform/mcp-oauth-proxy/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type refreshGrantStore struct {
	client       *types.ClientInfo
	grant        *types.Grant
	token        *types.TokenData
	storedToken  *types.TokenData
	revokedToken string
}

func (s *refreshGrantStore) GetClient(clientID string) (*types.ClientInfo, error) {
	return s.client, nil
}

func (s *refreshGrantStore) StoreToken(token *types.TokenData) error {
	s.storedToken = token
	return nil
}

func (s *refreshGrantStore) ValidateAuthCode(code string) (string, string, error) {
	return "", "", nil
}

func (s *refreshGrantStore) GetGrant(grantID string, userID string) (*types.Grant, error) {
	return s.grant, nil
}

func (s *refreshGrantStore) DeleteAuthCode(code string) error {
	return nil
}

func (s *refreshGrantStore) GetTokenByRefreshToken(refreshToken string) (*types.TokenData, error) {
	s.token.RefreshToken = refreshToken
	return s.token, nil
}

func (s *refreshGrantStore) RevokeToken(token string) error {
	s.revokedToken = token
	return nil
}

func TestRefreshTokenGrantRevokesOldRefreshToken(t *testing.T) {
	const oldRefreshToken = "user:grant:old-refresh"

	store := &refreshGrantStore{
		client: &types.ClientInfo{
			ClientID:                "client",
			TokenEndpointAuthMethod: "none",
		},
		grant: &types.Grant{
			ID:       "grant",
			ClientID: "client",
			UserID:   "user",
			Scope:    []string{"openid", "authoring/propose"},
		},
		token: &types.TokenData{
			ClientID:              "client",
			UserID:                "user",
			GrantID:               "grant",
			Scope:                 "openid authoring/propose",
			ExpiresAt:             time.Now().Add(time.Hour),
			RefreshTokenExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		},
	}

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {"client"},
		"refresh_token": {oldRefreshToken},
	}
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	NewHandler(store).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, store.storedToken)
	assert.Equal(t, oldRefreshToken, store.revokedToken)
	assert.NotEqual(t, oldRefreshToken, store.storedToken.RefreshToken)

	var response types.TokenResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.Equal(t, store.storedToken.RefreshToken, response.RefreshToken)
}
