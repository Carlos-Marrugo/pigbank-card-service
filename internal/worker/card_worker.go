package worker

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"github.com/Carlos-Marrugo/pigbank-card-service/internal/models"
	"github.com/Carlos-Marrugo/pigbank-card-service/internal/repository"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
)

type CardWorker struct {
	sqsClient         *sqs.Client
	repo              *repository.CardRepository
	queueURL          string
	notificationQueue string
}

func NewCardWorker(sqsClient *sqs.Client, repo *repository.CardRepository, queueURL, notificationQueue string) *CardWorker {
	return &CardWorker{
		sqsClient:         sqsClient,
		repo:              repo,
		queueURL:          queueURL,
		notificationQueue: notificationQueue,
	}
}

func (w *CardWorker) Start(ctx context.Context) {
	log.Println("Card Worker started, listening for messages...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Card Worker stopped")
			return
		default:
			w.pollMessages(ctx)
			time.Sleep(5 * time.Second)
		}
	}
}

func (w *CardWorker) pollMessages(ctx context.Context) {
	result, err := w.sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(w.queueURL),
		MaxNumberOfMessages: 5,
		WaitTimeSeconds:     10,
	})
	if err != nil {
		log.Printf("Error receiving messages: %v", err)
		return
	}

	for _, msg := range result.Messages {
		w.processMessage(ctx, msg)
		w.deleteMessage(ctx, msg)
	}
}

func (w *CardWorker) processMessage(ctx context.Context, msg types.Message) {
	var req models.CardRequest
	if err := json.Unmarshal([]byte(*msg.Body), &req); err != nil {
		log.Printf("Error parsing message: %v", err)
		return
	}

	log.Printf("Processing card request for user: %s", req.UserID)

	score := rand.Intn(101)
	creditLimit := 100 + (float64(score)/100)*(10000000-100)

	debitCard := models.Card{
		CardID:   uuid.New().String(),
		UserID:   req.UserID,
		CardType: "DEBIT",
		Balance:  0,
		Limit:    0,
		Status:   "ACTIVE",
	}

	if err := w.repo.CreateCard(ctx, debitCard); err != nil {
		log.Printf("Error creating debit card: %v", err)
		return
	}

	creditCard := models.Card{
		CardID:   uuid.New().String(),
		UserID:   req.UserID,
		CardType: "CREDIT",
		Balance:  0,
		Limit:    creditLimit,
		Status:   "ACTIVE",
	}

	if err := w.repo.CreateCard(ctx, creditCard); err != nil {
		log.Printf("Error creating credit card: %v", err)
		return
	}

	w.sendNotification(ctx, req.UserID, score, creditLimit)

	log.Printf("Cards created for user %s: Debit=%s, Credit=%s (limit: %.2f, score: %d)",
		req.UserID, debitCard.CardID, creditCard.CardID, creditLimit, score)
}

func (w *CardWorker) sendNotification(ctx context.Context, userID string, score int, limit float64) {
	notification := map[string]interface{}{
		"user_id":      userID,
		"type":         "WELCOME",
		"score":        score,
		"credit_limit": limit,
		"email":        "user@example.com",
	}

	body, _ := json.Marshal(notification)

	_, err := w.sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(w.notificationQueue),
		MessageBody: aws.String(string(body)),
	})

	if err != nil {
		log.Printf("Error sending notification: %v", err)
	}
}

func (w *CardWorker) deleteMessage(ctx context.Context, msg types.Message) {
	_, err := w.sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(w.queueURL),
		ReceiptHandle: msg.ReceiptHandle,
	})
	if err != nil {
		log.Printf("Error deleting message: %v", err)
	}
}
