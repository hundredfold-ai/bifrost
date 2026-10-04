package vertex

import (
	"encoding/json"
	"testing"

	"github.com/maximhq/bifrost/core/schemas"
)

// Gemini Embedding 2 is reached at the global location through :embedContent, and nothing else is.
func TestGeminiEmbeddingTakesTheGlobalEmbedContentRoute(t *testing.T) {
	for model, want := range map[string]bool{
		"gemini-embedding-2":   true,
		"Gemini-Embedding-2":   true,
		"gemini-embedding-001": false,
		"text-embedding-005":   false,
		"multimodalembedding":  false,
	} {
		if got := isVertexGeminiEmbeddingModel(model); got != want {
			t.Fatalf("%s: %v", model, got)
		}
	}
	got := getCompleteURLForGeminiEndpoint("gemini-embedding-2", "global", "p", "", ":embedContent")
	want := "https://aiplatform.googleapis.com/v1/projects/p/locations/global/publishers/google/models/gemini-embedding-2:embedContent"
	if got != want {
		t.Fatalf("url %s", got)
	}
}

// One content per call, in the shape :embedContent reads; more than one input is refused rather than
// truncated to the first.
func TestGeminiEmbeddingRequestCarriesExactlyOneInput(t *testing.T) {
	dimensions := 768
	request, err := ToVertexGeminiEmbeddingRequest(&schemas.BifrostEmbeddingRequest{
		Model:  "gemini-embedding-2",
		Input:  &schemas.EmbeddingInput{Texts: []string{"hello"}},
		Params: &schemas.EmbeddingParameters{Dimensions: &dimensions},
	})
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"content":{"parts":[{"text":"hello"}]},"outputDimensionality":768}` {
		t.Fatalf("body %s", body)
	}
	if _, err := ToVertexGeminiEmbeddingRequest(&schemas.BifrostEmbeddingRequest{
		Input: &schemas.EmbeddingInput{Texts: []string{"a", "b"}},
	}); err == nil {
		t.Fatal("two inputs were accepted")
	}
	if _, err := ToVertexGeminiEmbeddingRequest(&schemas.BifrostEmbeddingRequest{
		Input: &schemas.EmbeddingInput{Embedding: []int{1}},
	}); err == nil {
		t.Fatal("a token input was accepted")
	}
}

// The answer measured from Vertex, converted with its usage.
func TestGeminiEmbeddingResponseCarriesItsUsage(t *testing.T) {
	var response VertexGeminiEmbeddingResponse
	if err := json.Unmarshal([]byte(`{"embedding":{"values":[0.5,-0.25]},`+
		`"usageMetadata":{"promptTokenCount":3,"totalTokenCount":3,"trafficType":"ON_DEMAND"}}`,
	), &response); err != nil {
		t.Fatal(err)
	}
	converted := response.ToBifrostEmbeddingResponse()
	if converted == nil || len(converted.Data) != 1 || converted.Data[0].Index != 0 ||
		len(converted.Data[0].Embedding.EmbeddingArray) != 2 ||
		converted.Usage == nil || converted.Usage.PromptTokens != 3 {
		t.Fatalf("converted %+v", converted)
	}
	empty := VertexGeminiEmbeddingResponse{}
	if empty.ToBifrostEmbeddingResponse() != nil {
		t.Fatal("an answer with no vector converted")
	}
}
