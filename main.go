package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"github.com/zhu327/gemini-openai-proxy/api"
)

func main() {
	// Define a flag for the port
	port := flag.Int("port", 8080, "Port to listen on")
	apiKey := flag.String("api-key", "", "API key")
	flag.Parse()

	if *apiKey == "" {
		panic("api-key is required")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(*apiKey))
	if err != nil {
		log.Printf("new genai client error %v\n", err)
		panic(err)
	}
	defer client.Close()

	// Create a new Gin router
	router := gin.Default()
	api.Register(router, client)

	// Run the server on port 8080
	err = router.Run(fmt.Sprintf(":%d", *port))
	if err != nil {
		panic(err)
	}
}
