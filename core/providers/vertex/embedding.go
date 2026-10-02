package vertex

import (
	"fmt"
	"strings"

	"github.com/maximhq/bifrost/core/schemas"
)

// ToVertexEmbeddingRequest converts a Bifrost embedding request to Vertex AI format
func ToVertexEmbeddingRequest(bifrostReq *schemas.BifrostEmbeddingRequest) *VertexEmbeddingRequest {
	if bifrostReq == nil || bifrostReq.Input == nil || (bifrostReq.Input.Text == nil && bifrostReq.Input.Texts == nil) {
		return nil
	}
	// Create the request
	vertexReq := &VertexEmbeddingRequest{}
	if bifrostReq.Params != nil {
		vertexReq.ExtraParams = bifrostReq.Params.ExtraParams
	}
	var texts []string
	if bifrostReq.Input.Text != nil {
		texts = []string{*bifrostReq.Input.Text}
	} else {
		texts = bifrostReq.Input.Texts
	}

	// Create instances for each text
	instances := make([]VertexEmbeddingInstance, 0, len(texts))
	for _, text := range texts {
		instance := VertexEmbeddingInstance{
			Content: text,
		}

		// Add optional task_type and title from params
		if bifrostReq.Params != nil {
			if taskTypeStr, ok := schemas.SafeExtractStringPointer(bifrostReq.Params.ExtraParams["task_type"]); ok {
				delete(vertexReq.ExtraParams, "task_type")
				instance.TaskType = taskTypeStr
			}
			if title, ok := schemas.SafeExtractStringPointer(bifrostReq.Params.ExtraParams["title"]); ok {
				delete(vertexReq.ExtraParams, "title")
				instance.Title = title
			}
		}

		instances = append(instances, instance)
	}
	vertexReq.Instances = instances
	// Add parameters if present
	if bifrostReq.Params != nil {
		parameters := &VertexEmbeddingParameters{}

		// Set autoTruncate (defaults to true)
		autoTruncate := true
		if bifrostReq.Params.ExtraParams != nil {
			if autoTruncateVal, ok := schemas.SafeExtractBool(bifrostReq.Params.ExtraParams["autoTruncate"]); ok {
				delete(vertexReq.ExtraParams, "autoTruncate")
				autoTruncate = autoTruncateVal
			}
		}
		parameters.AutoTruncate = &autoTruncate

		// Add outputDimensionality if specified
		if bifrostReq.Params.Dimensions != nil {
			delete(vertexReq.ExtraParams, "dimensions")
			parameters.OutputDimensionality = bifrostReq.Params.Dimensions
		}

		vertexReq.Parameters = parameters
	}

	return vertexReq
}

// ToBifrostEmbeddingResponse converts a Vertex AI embedding response to Bifrost format
func (response *VertexEmbeddingResponse) ToBifrostEmbeddingResponse() *schemas.BifrostEmbeddingResponse {
	if response == nil || len(response.Predictions) == 0 {
		return nil
	}

	// Convert predictions to Bifrost embeddings
	embeddings := make([]schemas.EmbeddingData, 0, len(response.Predictions))
	var usage *schemas.BifrostLLMUsage

	for i, prediction := range response.Predictions {
		if prediction.Embeddings == nil || len(prediction.Embeddings.Values) == 0 {
			continue
		}

		// Create embedding object
		embedding := schemas.EmbeddingData{
			Object: "embedding",
			Embedding: schemas.EmbeddingStruct{
				EmbeddingArray: append([]float64(nil), prediction.Embeddings.Values...),
			},
			Index: i,
		}

		// Extract statistics if available
		if prediction.Embeddings.Statistics != nil {
			if usage == nil {
				usage = &schemas.BifrostLLMUsage{}
			}
			usage.TotalTokens += prediction.Embeddings.Statistics.TokenCount
			usage.PromptTokens += prediction.Embeddings.Statistics.TokenCount
		}

		embeddings = append(embeddings, embedding)
	}

	return &schemas.BifrostEmbeddingResponse{
		Object: "list",
		Data:   embeddings,
		Usage:  usage,
		ExtraFields: schemas.BifrostResponseExtraFields{
		},
	}
}

// isVertexGeminiEmbeddingModel reports whether Vertex serves a model only through :embedContent.
//
// Gemini Embedding 2 is not a :predict model on Vertex: a regional :predict answers 404 for it, and
// it is served at the global location alone. The predicate is upstream's (maximhq/bifrost#2559), so
// a later rebase onto a release carrying that change replaces this one without changing which models
// take which route.
func isVertexGeminiEmbeddingModel(model string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(model)), "gemini-embedding-2")
}

// ToVertexGeminiEmbeddingRequest converts a Bifrost embedding request to Vertex's :embedContent
// body. The endpoint embeds exactly one content per call, so a request carrying more than one input
// is refused here rather than silently embedding the first: a caller with several inputs makes
// several calls, which is what upstream requires as well.
func ToVertexGeminiEmbeddingRequest(bifrostReq *schemas.BifrostEmbeddingRequest) (*VertexGeminiEmbeddingRequest, error) {
	if bifrostReq == nil || bifrostReq.Input == nil {
		return nil, fmt.Errorf("embedding input is not provided")
	}
	var text string
	switch {
	case bifrostReq.Input.Text != nil:
		text = *bifrostReq.Input.Text
	case len(bifrostReq.Input.Texts) == 1:
		text = bifrostReq.Input.Texts[0]
	case len(bifrostReq.Input.Texts) > 1:
		return nil, fmt.Errorf("vertex gemini embedding takes one input per request, got %d", len(bifrostReq.Input.Texts))
	default:
		return nil, fmt.Errorf("vertex gemini embedding takes text input")
	}
	request := &VertexGeminiEmbeddingRequest{
		Content: VertexGeminiEmbeddingContent{Parts: []VertexGeminiEmbeddingPart{{Text: text}}},
	}
	if bifrostReq.Params != nil {
		request.OutputDimensionality = bifrostReq.Params.Dimensions
		request.ExtraParams = bifrostReq.Params.ExtraParams
	}
	return request, nil
}

// ToBifrostEmbeddingResponse converts one :embedContent answer. Usage is the call's own
// usageMetadata, which :embedContent reports for the whole call rather than per embedding.
func (response *VertexGeminiEmbeddingResponse) ToBifrostEmbeddingResponse() *schemas.BifrostEmbeddingResponse {
	if response == nil || len(response.Embedding.Values) == 0 {
		return nil
	}
	converted := &schemas.BifrostEmbeddingResponse{
		Object: "list",
		Data: []schemas.EmbeddingData{{
			Object: "embedding",
			Embedding: schemas.EmbeddingStruct{
				EmbeddingArray: append([]float64(nil), response.Embedding.Values...),
			},
			Index: 0,
		}},
	}
	if response.UsageMetadata != nil {
		converted.Usage = &schemas.BifrostLLMUsage{
			PromptTokens: response.UsageMetadata.PromptTokenCount,
			TotalTokens:  response.UsageMetadata.TotalTokenCount,
		}
	}
	return converted
}
