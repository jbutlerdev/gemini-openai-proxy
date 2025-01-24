package api

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	"github.com/pkg/errors"
	openai "github.com/sashabaranov/go-openai"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"

	"github.com/zhu327/gemini-openai-proxy/pkg/adapter"
)

func IndexHandler(c *gin.Context) {
	c.JSON(http.StatusMisdirectedRequest, gin.H{
		"message": "Welcome to the OpenAI API! Documentation is available at https://platform.openai.com/docs/api-reference",
	})
}

func ModelListHandler(c *gin.Context, client *genai.Client) {
	ctx := c.Request.Context()
	iter := client.ListModels(ctx)
	models := make([]openai.Model, 0)
	for {
		m, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			panic(err)
		}
		models = append(models, openai.Model{
			CreatedAt: 1686935002,
			ID:        m.Name,
			Object:    "model",
			OwnedBy:   "gemini",
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   models,
	})
}

func ModelRetrieveHandler(c *gin.Context, client *genai.Client) {
	model := c.Param("model")
	c.JSON(http.StatusOK, openai.Model{
		CreatedAt: 1686935002,
		ID:        model,
		Object:    "model",
		OwnedBy:   "gemini",
	})
}

func ChatProxyHandler(c *gin.Context, client *genai.Client) {
	// Retrieve the Authorization header value
	authorizationHeader := c.GetHeader("Authorization")
	// Declare a variable to store the OPENAI_API_KEY
	var openaiAPIKey string
	// Use fmt.Sscanf to extract the Bearer token
	_, err := fmt.Sscanf(authorizationHeader, "Bearer %s", &openaiAPIKey)
	if err != nil {
		handleGenerateContentError(c, err)
		return
	}

	req := &adapter.ChatCompletionRequest{}
	// Bind the JSON data from the request to the struct
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, openai.APIError{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	messages, err := req.ToGenaiMessages()
	if err != nil {
		c.JSON(http.StatusBadRequest, openai.APIError{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	gemini := adapter.NewGeminiAdapter(client, req.Model)

	if !req.Stream {
		resp, err := gemini.GenerateContent(ctx, req, messages)
		if err != nil {
			handleGenerateContentError(c, err)
			return
		}

		c.JSON(http.StatusOK, resp)
		return
	}

	dataChan, err := gemini.GenerateStreamContent(ctx, req, messages)
	if err != nil {
		handleGenerateContentError(c, err)
		return
	}

	setEventStreamHeaders(c)
	c.Stream(func(w io.Writer) bool {
		if data, ok := <-dataChan; ok {
			c.Render(-1, adapter.Event{Data: "data: " + data})
			return true
		}
		c.Render(-1, adapter.Event{Data: "data: [DONE]"})
		return false
	})
}

func handleGenerateContentError(c *gin.Context, err error) {
	log.Printf("genai generate content error %v\n", err)

	// Try OpenAI API error first
	var openaiErr *openai.APIError
	if errors.As(err, &openaiErr) {

		// Convert the code to an HTTP status code
		statusCode := http.StatusInternalServerError
		if code, ok := openaiErr.Code.(int); ok {
			statusCode = code
		}

		c.AbortWithStatusJSON(statusCode, openaiErr)
		return
	}

	// Try Google API error
	var googleErr *googleapi.Error
	if errors.As(err, &googleErr) {
		log.Printf("Handling Google API error with code: %d\n", googleErr.Code)
		statusCode := googleErr.Code
		if statusCode == http.StatusTooManyRequests {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, openai.APIError{
				Code:    http.StatusTooManyRequests,
				Message: "Rate limit exceeded",
				Type:    "rate_limit_error",
			})
			return
		}

		c.AbortWithStatusJSON(statusCode, openai.APIError{
			Code:    statusCode,
			Message: googleErr.Message,
			Type:    "server_error",
		})
		return
	}

	// For all other errors
	log.Printf("Handling unknown error: %v\n", err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, openai.APIError{
		Code:    http.StatusInternalServerError,
		Message: err.Error(),
		Type:    "server_error",
	})
}

func setEventStreamHeaders(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
}

func EmbeddingProxyHandler(c *gin.Context, client *genai.Client) {
	// Retrieve the Authorization header value
	authorizationHeader := c.GetHeader("Authorization")
	// Declare a variable to store the OPENAI_API_KEY
	var openaiAPIKey string
	// Use fmt.Sscanf to extract the Bearer token
	_, err := fmt.Sscanf(authorizationHeader, "Bearer %s", &openaiAPIKey)
	if err != nil {
		handleGenerateContentError(c, err)
		return
	}

	req := &adapter.EmbeddingRequest{}
	// Bind the JSON data from the request to the struct
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, openai.APIError{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	messages, err := req.ToGenaiMessages()
	if err != nil {
		c.JSON(http.StatusBadRequest, openai.APIError{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	gemini := adapter.NewGeminiAdapter(client, req.Model)

	resp, err := gemini.GenerateEmbedding(ctx, messages)
	if err != nil {
		handleGenerateContentError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
