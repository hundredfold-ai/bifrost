// Modified by Hundredfold AI; see HUNDREDFOLD_MODIFICATIONS.md.

package gemini

import (
	"github.com/maximhq/bifrost/core/schemas"
)

// ToGeminiEmbeddingRequest converts a BifrostRequest with embedding input to Gemini's batch embedding request format
// GeminiGenerationRequest contains requests array for batch embed content endpoint
func ToGeminiEmbeddingRequest(bifrostReq *schemas.BifrostEmbeddingRequest) *GeminiBatchEmbeddingRequest {
	if bifrostReq == nil || bifrostReq.Input == nil || (bifrostReq.Input.Text == nil && bifrostReq.Input.Texts == nil) {
		return nil
	}

	embeddingInput := bifrostReq.Input

	// Collect all texts to embed
	var texts []string
	if embeddingInput.Text != nil {
		texts = append(texts, *embeddingInput.Text)
	}
	if len(embeddingInput.Texts) > 0 {
		texts = append(texts, embeddingInput.Texts...)
	}

	if len(texts) == 0 {
		return nil
	}

	// Create batch embedding request with one request per text
	batchRequest := &GeminiBatchEmbeddingRequest{
		Requests: make([]GeminiEmbeddingRequest, len(texts)),
	}
	if bifrostReq.Params != nil {
		batchRequest.ExtraParams = bifrostReq.Params.ExtraParams
	}

	// Create individual embedding requests for each text
	for i, text := range texts {
		embeddingReq := GeminiEmbeddingRequest{
			Model: "models/" + bifrostReq.Model,
			Content: &Content{
				Parts: []*Part{
					{
						Text: text,
					},
				},
			},
		}

		// Add parameters if available
		if bifrostReq.Params != nil {
			if bifrostReq.Params.Dimensions != nil {
				embeddingReq.OutputDimensionality = bifrostReq.Params.Dimensions
			}

			// Handle extra parameters
			if bifrostReq.Params.ExtraParams != nil {
				if taskType, ok := schemas.SafeExtractStringPointer(bifrostReq.Params.ExtraParams["taskType"]); ok {
					delete(batchRequest.ExtraParams, "taskType")
					embeddingReq.TaskType = taskType
				}
				if title, ok := schemas.SafeExtractStringPointer(bifrostReq.Params.ExtraParams["title"]); ok {
					delete(batchRequest.ExtraParams, "title")
					embeddingReq.Title = title
				}
			}
		}

		batchRequest.Requests[i] = embeddingReq
	}

	return batchRequest
}

// ToGeminiEmbedContentResponse converts a BifrostEmbeddingResponse to the single :embedContent wire format.
func ToGeminiEmbedContentResponse(bifrostResp *schemas.BifrostEmbeddingResponse) *GeminiEmbedContentResponse {
	if bifrostResp == nil || len(bifrostResp.Data) == 0 {
		return nil
	}
	values := bifrostResp.Data[0].Embedding.EmbeddingArray
	if values == nil && len(bifrostResp.Data[0].Embedding.Embedding2DArray) > 0 {
		values = bifrostResp.Data[0].Embedding.Embedding2DArray[0]
	}
	embedding := GeminiEmbedding{
		Values: append([]float64(nil), values...),
	}
	if bifrostResp.Usage != nil {
		embedding.Statistics = &ContentEmbeddingStatistics{
			TokenCount: int32(bifrostResp.Usage.PromptTokens),
		}
	}
	return &GeminiEmbedContentResponse{Embedding: embedding}
}

// ToGeminiEmbeddingResponse converts a BifrostResponse with embedding data to Gemini's embedding response format
func ToGeminiEmbeddingResponse(bifrostResp *schemas.BifrostEmbeddingResponse) *GeminiEmbeddingResponse {
	if bifrostResp == nil || len(bifrostResp.Data) == 0 {
		return nil
	}

	geminiResp := &GeminiEmbeddingResponse{
		Embeddings: make([]GeminiEmbedding, len(bifrostResp.Data)),
	}

	// Convert each embedding from Bifrost format to Gemini format
	for i, embedding := range bifrostResp.Data {
		var values []float64

		// Extract embedding values from BifrostEmbeddingResponse
		if embedding.Embedding.EmbeddingArray != nil {
			values = append([]float64(nil), embedding.Embedding.EmbeddingArray...)
		} else if len(embedding.Embedding.Embedding2DArray) > 0 {
			// If it's a 2D array, take the first array
			values = append([]float64(nil), embedding.Embedding.Embedding2DArray[0]...)
		}

		geminiEmbedding := GeminiEmbedding{
			Values: values,
		}

		// Add statistics if available (token count from usage metadata)
		if bifrostResp.Usage != nil {
			geminiEmbedding.Statistics = &ContentEmbeddingStatistics{
				TokenCount: int32(bifrostResp.Usage.PromptTokens),
			}
		}

		geminiResp.Embeddings[i] = geminiEmbedding
	}

	// Set metadata if available (for Vertex API compatibility)
	if bifrostResp.Usage != nil {
		geminiResp.Metadata = &EmbedContentMetadata{
			BillableCharacterCount: int32(bifrostResp.Usage.PromptTokens),
		}
	}

	return geminiResp
}

// ToBifrostEmbeddingResponse converts a Gemini embedding response to BifrostEmbeddingResponse format
func ToBifrostEmbeddingResponse(geminiResp *GeminiEmbeddingResponse, model string) *schemas.BifrostEmbeddingResponse {
	if geminiResp == nil || len(geminiResp.Embeddings) == 0 {
		return nil
	}

	bifrostResp := &schemas.BifrostEmbeddingResponse{
		Data:   make([]schemas.EmbeddingData, len(geminiResp.Embeddings)),
		Model:  model,
		Object: "list",
	}

	// Convert each embedding from Gemini format to Bifrost format
	for i, geminiEmbedding := range geminiResp.Embeddings {
		embeddingData := schemas.EmbeddingData{
			Index:  i,
			Object: "embedding",
			Embedding: schemas.EmbeddingStruct{
				EmbeddingArray: geminiEmbedding.Values,
			},
		}

		bifrostResp.Data[i] = embeddingData
	}

	// Token usage, in the order a provider is most likely to report it.
	//
	// The Gemini API reports one usageMetadata for the whole batch. Vertex reports a statistics
	// block on each embedding instead, and those are summed: an embedding call is billed for every
	// input, so taking the first input's count understates a batch by everything after it.
	//
	// metadata.billableCharacterCount is deliberately not used. It counts characters, not tokens,
	// and reporting it as prompt tokens overstated usage by the characters-per-token ratio. Where no
	// token count is reported, Usage stays nil so a caller can tell "unreported" from "zero".
	var promptTokens int
	reported := false
	if geminiResp.UsageMetadata != nil && geminiResp.UsageMetadata.PromptTokenCount > 0 {
		promptTokens, reported = int(geminiResp.UsageMetadata.PromptTokenCount), true
	} else {
		for _, embedding := range geminiResp.Embeddings {
			if embedding.Statistics != nil && embedding.Statistics.TokenCount > 0 {
				promptTokens += int(embedding.Statistics.TokenCount)
				reported = true
			}
		}
	}
	if reported {
		bifrostResp.Usage = &schemas.BifrostLLMUsage{
			PromptTokens: promptTokens,
			TotalTokens:  promptTokens,
		}
	}

	return bifrostResp
}

// ToBifrostEmbeddingRequest converts a GeminiGenerationRequest to BifrostEmbeddingRequest format
func (request *GeminiGenerationRequest) ToBifrostEmbeddingRequest(ctx *schemas.BifrostContext) *schemas.BifrostEmbeddingRequest {
	if request == nil {
		return nil
	}

	provider, model := schemas.ParseModelString(request.Model, "")

	// Create the embedding request
	bifrostReq := &schemas.BifrostEmbeddingRequest{
		Provider:  provider,
		Model:     model,
		Fallbacks: schemas.ParseFallbacks(request.Fallbacks),
	}

	// SDK batch embedding request contains multiple embedding requests with same parameters but different text fields.
	if len(request.Requests) > 0 {
		var texts []string
		for _, req := range request.Requests {
			if req.Content != nil && len(req.Content.Parts) > 0 {
				for _, part := range req.Content.Parts {
					if part != nil && part.Text != "" {
						texts = append(texts, part.Text)
					}
				}
			}
		}
		if len(texts) > 0 {
			bifrostReq.Input = &schemas.EmbeddingInput{}
			if len(texts) == 1 {
				bifrostReq.Input.Text = &texts[0]
			} else {
				bifrostReq.Input.Texts = texts
			}
		}

		embeddingRequest := request.Requests[0]

		// Convert parameters
		if embeddingRequest.OutputDimensionality != nil || embeddingRequest.TaskType != nil || embeddingRequest.Title != nil {
			bifrostReq.Params = &schemas.EmbeddingParameters{}

			if embeddingRequest.OutputDimensionality != nil {
				bifrostReq.Params.Dimensions = embeddingRequest.OutputDimensionality
			}

			// Handle extra parameters
			if embeddingRequest.TaskType != nil || embeddingRequest.Title != nil {
				bifrostReq.Params.ExtraParams = make(map[string]interface{})
				if embeddingRequest.TaskType != nil {
					bifrostReq.Params.ExtraParams["taskType"] = embeddingRequest.TaskType
				}
				if embeddingRequest.Title != nil {
					bifrostReq.Params.ExtraParams["title"] = embeddingRequest.Title
				}
			}
		}
	}

	// Generation-style requests (e.g., non-Imagen :predict) carry text in contents[].parts[].
	// If no SDK requests[] were provided, derive embedding input from contents.
	if bifrostReq.Input == nil {
		var texts []string
		for _, content := range request.Contents {
			for _, part := range content.Parts {
				if part != nil && part.Text != "" {
					texts = append(texts, part.Text)
				}
			}
		}
		if len(texts) > 0 {
			bifrostReq.Input = &schemas.EmbeddingInput{}
			if len(texts) == 1 {
				bifrostReq.Input.Text = &texts[0]
			} else {
				bifrostReq.Input.Texts = texts
			}
		}
	}

	return bifrostReq
}
