// Modified by Hundredfold AI; see HUNDREDFOLD_MODIFICATIONS.md.

package vertex

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/maximhq/bifrost/core/schemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func accessTokenKey(token string) schemas.Key {
	return schemas.Key{
		Value: *schemas.NewSecretVar(""),
		VertexKeyConfig: &schemas.VertexKeyConfig{
			ProjectID:       *schemas.NewSecretVar("demo-project"),
			Region:          *schemas.NewSecretVar("global"),
			AuthCredentials: *schemas.NewSecretVar(`{"type":"access_token","access_token":"` + token + `"}`),
		},
	}
}

func TestAPresentedAccessTokenIsUsedAsItIsAndNeverPooled(t *testing.T) {
	t.Parallel()
	source, err := getAuthTokenSource(accessTokenKey("token-a"))
	require.NoError(t, err)
	token, err := source.Token()
	require.NoError(t, err)
	assert.Equal(t, "token-a", token.AccessToken)

	_, pooled := vertexTokenSourcePool.Load(getClientKey(`{"type":"access_token","access_token":"token-a"}`))
	assert.False(t, pooled, "a presented access token must not outlive its request in the pool")

	// A second token is a second token, not the first one read back out of a cache.
	source, err = getAuthTokenSource(accessTokenKey("token-b"))
	require.NoError(t, err)
	token, err = source.Token()
	require.NoError(t, err)
	assert.Equal(t, "token-b", token.AccessToken)

	_, err = getAuthTokenSource(accessTokenKey(""))
	require.Error(t, err)
}

func TestARerankFollowsTheConfiguredOriginAndPresentsTheToken(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var path, authorization, project string
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		raw, _ := io.ReadAll(request.Body)
		mu.Lock()
		path, authorization, project = request.URL.Path, request.Header.Get("Authorization"), request.Header.Get("X-Goog-User-Project")
		_ = json.Unmarshal(raw, &body)
		mu.Unlock()
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"records":[{"id":"idx:1","score":0.9},{"id":"idx:0","score":0.2}]}`))
	}))
	t.Cleanup(server.Close)

	provider, err := NewVertexProvider(&schemas.ProviderConfig{
		NetworkConfig: schemas.NetworkConfig{
			BaseURL:                        server.URL,
			DefaultRequestTimeoutInSeconds: 5,
			AllowPrivateNetwork:            true,
		},
	}, nil)
	require.NoError(t, err)
	ctx := schemas.NewBifrostContext(t.Context(), schemas.NoDeadline)
	response, bifrostErr := provider.Rerank(ctx, accessTokenKey("token-a"), &schemas.BifrostRerankRequest{
		Provider: schemas.Vertex,
		Model:    "semantic-ranker-default-004",
		Query:    "create an order",
		Documents: []schemas.RerankDocument{
			{Text: "Lists orders."},
			{Text: "Creates an order."},
		},
	})
	require.Nil(t, bifrostErr)
	require.Len(t, response.Results, 2)
	assert.Equal(t, 1, response.Results[0].Index)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, "/v1/projects/demo-project/locations/global/rankingConfigs/default_ranking_config:rank", path)
	assert.Equal(t, "Bearer token-a", authorization)
	assert.Equal(t, "demo-project", project)
	assert.Equal(t, "semantic-ranker-default-004", body["model"])
}
