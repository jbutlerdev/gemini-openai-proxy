package api

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
)

func Register(router *gin.Engine, client *genai.Client) {
	// Configure CORS to allow all methods and all origins
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = []string{"*"}
	config.AllowCredentials = true
	config.OptionsResponseStatusCode = http.StatusOK
	router.Use(cors.New(config))

	// Define a route and its handler
	router.GET("/", IndexHandler)
	// openai model
	router.GET("/v1/models", func(c *gin.Context) {
		ModelListHandler(c, client)
	})
	router.GET("/v1/models/:model", func(c *gin.Context) {
		ModelRetrieveHandler(c, client)
	})

	// openai chat
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		ChatProxyHandler(c, client)
	})

	// openai embeddings
	router.POST("/v1/embeddings", func(c *gin.Context) {
		EmbeddingProxyHandler(c, client)
	})
}
