package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/weave-agent/weave/sdk"
	sdkmodel "github.com/weave-agent/weave/sdk/model"

	tuievents "github.com/weave-agent/weave-tui/internal/events"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildLoginProviders_EmptyWhenNoProviders(t *testing.T) {
	resetAuthTestRegistries(t)

	providers := BuildLoginProviders()
	assert.Empty(t, providers)
}

func TestBuildLoginProviders_IncludesOAuthProviders(t *testing.T) {
	resetAuthTestRegistries(t)

	sdk.RegisterOAuthProvider(sdk.OAuthProvider{
		ID:   "test-oauth-provider",
		Name: "Test OAuth Provider",
	})

	providers := BuildLoginProviders()
	require.Len(t, providers, 1)
	assert.Equal(t, "Test OAuth Provider", providers[0].Name)
	assert.Equal(t, "test-oauth-provider", providers[0].ID)
	assert.True(t, providers[0].IsOAuth)
	assert.False(t, providers[0].HasAuth)
}

func TestBuildLoginProviders_IncludesRegisteredProviders(t *testing.T) {
	resetAuthTestRegistries(t)

	sdk.RegisterProvider[struct{}, struct{}]("test-api-provider", func(_ sdk.Config, _, _ struct{}) (sdk.Provider, error) {
		return nil, errors.New("test provider not implemented")
	})

	providers := BuildLoginProviders()
	require.Len(t, providers, 1)
	assert.Equal(t, "Test-api-provider", providers[0].Name)
	assert.Equal(t, "test-api-provider", providers[0].ID)
	assert.False(t, providers[0].IsOAuth)
	assert.False(t, providers[0].HasAuth)
}

func TestBuildLoginProviders_IncludesModelRegistryProviders(t *testing.T) {
	resetAuthTestRegistries(t)

	sdkmodel.RegisterModel(sdkmodel.ModelDef{
		ID:       "test-model",
		Provider: "model-only-provider",
	})

	providers := BuildLoginProviders()
	require.Len(t, providers, 1)
	assert.Equal(t, "Model-only-provider", providers[0].Name)
	assert.Equal(t, "model-only-provider", providers[0].ID)
	assert.False(t, providers[0].IsOAuth)
	assert.False(t, providers[0].HasAuth)
}

func TestBuildLoginProviders_DeduplicatesProviderSources(t *testing.T) {
	resetAuthTestRegistries(t)

	sdk.RegisterOAuthProvider(sdk.OAuthProvider{ID: "duplicate-provider", Name: "Duplicate OAuth"})
	sdk.RegisterProvider[struct{}, struct{}]("duplicate-provider", func(_ sdk.Config, _, _ struct{}) (sdk.Provider, error) {
		return nil, errors.New("test provider not implemented")
	})
	sdkmodel.RegisterModel(sdkmodel.ModelDef{
		ID:       "duplicate-model",
		Provider: "duplicate-provider",
	})
	sdkmodel.SetProviderAuth("duplicate-provider", true)

	providers := BuildLoginProviders()
	require.Len(t, providers, 1)
	assert.Equal(t, "Duplicate OAuth", providers[0].Name)
	assert.Equal(t, "duplicate-provider", providers[0].ID)
	assert.True(t, providers[0].IsOAuth)
	assert.True(t, providers[0].HasAuth)
}

func TestBuildLoginProviders_SortsByName(t *testing.T) {
	resetAuthTestRegistries(t)

	sdk.RegisterOAuthProvider(sdk.OAuthProvider{ID: "zebra", Name: "Zebra"})
	sdk.RegisterOAuthProvider(sdk.OAuthProvider{ID: "alpha", Name: "Alpha"})

	providers := BuildLoginProviders()
	require.Len(t, providers, 2)
	assert.Equal(t, "alpha", providers[0].ID)
	assert.Equal(t, "zebra", providers[1].ID)
}

func TestBuildLogoutProviders_EmptyWhenNoAuth(t *testing.T) {
	resetAuthTestRegistries(t)

	providers := BuildLogoutProviders()
	assert.Empty(t, providers)
}

func TestBuildLogoutProviders_IncludesAuthenticatedProviders(t *testing.T) {
	resetAuthTestRegistries(t)

	sdk.RegisterProvider[struct{}, struct{}]("authed-provider", func(_ sdk.Config, _, _ struct{}) (sdk.Provider, error) {
		return nil, errors.New("test provider not implemented")
	})
	sdkmodel.SetProviderAuth("authed-provider", true)

	providers := BuildLogoutProviders()
	require.Len(t, providers, 1)
	assert.Equal(t, "authed-provider", providers[0].ID)
	assert.Equal(t, "Authed-provider", providers[0].Name)
}

func TestBuildLogoutProviders_IncludesOAuthAndModelRegistryProviders(t *testing.T) {
	resetAuthTestRegistries(t)

	sdk.RegisterOAuthProvider(sdk.OAuthProvider{ID: "oauth-logout", Name: "OAuth Logout"})
	sdkmodel.RegisterModel(sdkmodel.ModelDef{
		ID:       "test-model",
		Provider: "model-logout",
	})
	sdkmodel.SetProviderAuth("oauth-logout", true)
	sdkmodel.SetProviderAuth("model-logout", true)

	providers := BuildLogoutProviders()
	require.Len(t, providers, 2)
	assert.Equal(t, "model-logout", providers[0].ID)
	assert.Equal(t, "Model-logout", providers[0].Name)
	assert.Equal(t, "oauth-logout", providers[1].ID)
	assert.Equal(t, "OAuth Logout", providers[1].Name)
}

func TestDisplayNameForProvider(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"anthropic", "Anthropic"},
		{"openai", "Openai"},
		{"", ""},
		{"a", "A"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, DisplayNameForProvider(tt.input))
		})
	}
}

func TestRunOAuthFlowCmd_StartError(t *testing.T) {
	cmd := RunOAuthFlowCmd(context.Background(), sdk.OAuthProvider{
		ID:          "bad-oauth",
		RedirectURI: "://bad",
	}, 3)

	msg := cmd()
	result, ok := msg.(tuievents.LoginFlowResultMsg)
	require.True(t, ok)
	assert.Equal(t, "bad-oauth", result.Provider)
	assert.Equal(t, 3, result.Gen)
	require.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "start callback server")
}

func TestPollDeviceCodeCmd_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"access_token":  "at-test",
			"refresh_token": "rt-test",
			"token_type":    "bearer",
			"expires_in":    3600,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := PollDeviceCodeCmd(context.Background(), "test-provider", "dc-123", 1, server.URL, "client-id", 1)
	msg := cmd()

	result, ok := msg.(tuievents.LoginFlowResultMsg)
	require.True(t, ok)
	assert.Equal(t, "test-provider", result.Provider)
	assert.Equal(t, 1, result.Gen)
	require.NoError(t, result.Error)
	assert.Equal(t, "at-test", result.Credential.AccessToken)
	assert.Equal(t, "rt-test", result.Credential.RefreshToken)
	assert.Equal(t, "bearer", result.Credential.TokenType)
	assert.False(t, result.Credential.ExpiresAt.IsZero())
}

func TestPollDeviceCodeCmd_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)

		resp := map[string]any{"error": "access_denied"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cmd := PollDeviceCodeCmd(context.Background(), "test-provider", "dc-123", 1, server.URL, "client-id", 2)
	msg := cmd()

	result, ok := msg.(tuievents.LoginFlowResultMsg)
	require.True(t, ok)
	assert.Equal(t, "test-provider", result.Provider)
	assert.Equal(t, 2, result.Gen)
	require.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "access_denied")
}

func resetAuthTestRegistries(t *testing.T) {
	t.Helper()

	sdk.ResetProviderRegistry()
	sdk.ResetOAuthRegistry()
	sdkmodel.ResetModelRegistry()
	sdkmodel.ResetAuthRegistry()

	t.Cleanup(func() {
		sdk.ResetProviderRegistry()
		sdk.ResetOAuthRegistry()
		sdkmodel.ResetModelRegistry()
		sdkmodel.ResetAuthRegistry()
	})
}
