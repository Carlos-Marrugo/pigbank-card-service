package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Carlos-Marrugo/pigbank-card-service/internal/api"
	"github.com/Carlos-Marrugo/pigbank-card-service/internal/repository"
	"github.com/Carlos-Marrugo/pigbank-card-service/internal/service"
	"github.com/Carlos-Marrugo/pigbank-card-service/internal/worker"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	ctx := context.Background()

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == "s3" {
			return aws.Endpoint{
				URL:               "http://localhost:4566",
				SigningRegion:     "us-east-1",
				HostnameImmutable: true,
			}, nil
		}
		return aws.Endpoint{
			URL:           "http://localhost:4566",
			SigningRegion: "us-east-1",
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	dbClient := dynamodb.NewFromConfig(cfg)
	sqsClient := sqs.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	cardsTable := os.Getenv("CARDS_TABLE")
	if cardsTable == "" {
		cardsTable = "pigbank-cards"
	}

	txTable := os.Getenv("TRANSACTIONS_TABLE")
	if txTable == "" {
		txTable = "pigbank-transactions"
	}

	reportsBucket := os.Getenv("REPORTS_BUCKET_NAME")
	if reportsBucket == "" {
		reportsBucket = "pigbank-reports-501e63ae"
	}

	queueURL := os.Getenv("CARD_REQUEST_QUEUE_URL")
	if queueURL == "" {
		queueURL = "http://sqs.us-east-1.localhost.localstack.cloud:4566/000000000000/create-request-card-sqs"
	}

	notificationQueue := os.Getenv("NOTIFICATION_QUEUE_URL")
	if notificationQueue == "" {
		notificationQueue = "http://sqs.us-east-1.localhost.localstack.cloud:4566/000000000000/notification-email-sqs"
	}

	repo := repository.NewCardRepository(dbClient, cardsTable, txTable)
	cardWorker := worker.NewCardWorker(sqsClient, repo, queueURL, notificationQueue)

	service.SetCardRepository(repo, s3Client, reportsBucket)

	r := gin.Default()
	cardHandler := &api.CardHandler{}

	v1 := r.Group("/api/v1")
	{
		v1.Use(authMiddleware())
		{
			v1.POST("/transactions", cardHandler.ProcessTransaction)
			v1.GET("/transactions/report", cardHandler.GenerateReport)
			v1.GET("/cards", cardHandler.GetCards)
			v1.GET("/cards/:card_id/balance", cardHandler.GetBalance)
		}
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go cardWorker.Start(ctx)
	go r.Run(":8082")

	log.Println("Card Service started on port 8082. Waiting for messages...")
	<-sigChan
	log.Println("Shutting down...")
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(401, gin.H{"error": "Bearer token required"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte("JWT_SECRET"), nil
		})

		if err != nil || !token.Valid {
			c.JSON(401, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(401, gin.H{"error": "Invalid claims"})
			c.Abort()
			return
		}

		userUUID, ok := claims["uuid"].(string)
		if !ok {
			c.JSON(401, gin.H{"error": "User ID not found in token"})
			c.Abort()
			return
		}

		c.Set("user_id", userUUID)
		c.Next()
	}
}
