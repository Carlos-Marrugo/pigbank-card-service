package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Carlos-Marrugo/pigbank-card-service/internal/repository"
	"github.com/Carlos-Marrugo/pigbank-card-service/internal/worker"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	ctx := context.Background()

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
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

	cardsTable := os.Getenv("CARDS_TABLE")
	if cardsTable == "" {
		cardsTable = "pigbank-cards"
	}

	txTable := os.Getenv("TRANSACTIONS_TABLE")
	if txTable == "" {
		txTable = "pigbank-transactions"
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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go cardWorker.Start(ctx)

	log.Println("Card Service started. Waiting for messages...")
	<-sigChan
	log.Println("Shutting down...")
}
