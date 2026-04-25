package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"

	"github.com/Carlos-Marrugo/pigbank-card-service/internal/models"
	"github.com/Carlos-Marrugo/pigbank-card-service/internal/repository"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

var (
	cardRepo      *repository.CardRepository
	s3Client      *s3.Client
	reportsBucket string
)

func SetCardRepository(repo *repository.CardRepository, s3 *s3.Client, bucket string) {
	cardRepo = repo
	s3Client = s3
	reportsBucket = bucket
}

func ProcessTransaction(ctx context.Context, req models.TransactionRequest, userID string) (*models.Transaction, error) {
	card, err := cardRepo.FindCardByID(ctx, req.CardID)
	if err != nil {
		return nil, fmt.Errorf("card not found: %v", err)
	}

	if card.UserID != userID {
		return nil, fmt.Errorf("card does not belong to user")
	}

	var newBalance float64

	switch req.Type {
	case "DEPOSIT":
		newBalance = card.Balance + req.Amount
		if err := cardRepo.UpdateCardBalance(ctx, req.CardID, newBalance); err != nil {
			return nil, err
		}

	case "PURCHASE":
		if card.CardType == "DEBIT" {
			if card.Balance < req.Amount {
				return nil, fmt.Errorf("insufficient balance")
			}
			newBalance = card.Balance - req.Amount
		} else if card.CardType == "CREDIT" {
			if card.Balance+req.Amount > card.Limit {
				return nil, fmt.Errorf("credit limit exceeded")
			}
			newBalance = card.Balance + req.Amount
		}
		if err := cardRepo.UpdateCardBalance(ctx, req.CardID, newBalance); err != nil {
			return nil, err
		}

	case "PAYMENT":
		if card.CardType != "CREDIT" {
			return nil, fmt.Errorf("payments only allowed for credit cards")
		}
		if req.Amount > card.Balance {
			return nil, fmt.Errorf("payment amount exceeds balance")
		}
		newBalance = card.Balance - req.Amount
		if err := cardRepo.UpdateCardBalance(ctx, req.CardID, newBalance); err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("invalid transaction type")
	}

	tx := models.Transaction{
		TransactionID: uuid.New().String(),
		CardID:        req.CardID,
		UserID:        userID,
		Type:          req.Type,
		Amount:        req.Amount,
		Description:   req.Description,
	}

	if err := cardRepo.SaveTransaction(ctx, tx); err != nil {
		return nil, err
	}

	return &tx, nil
}

func GenerateReport(ctx context.Context, userID string, from, to string) (string, error) {
	transactions, err := cardRepo.GetTransactionsByUser(ctx, userID, from, to)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	writer.Write([]string{"Transaction ID", "Card ID", "Type", "Amount", "Description", "Date"})

	for _, tx := range transactions {
		writer.Write([]string{
			tx.TransactionID,
			tx.CardID,
			tx.Type,
			fmt.Sprintf("%.2f", tx.Amount),
			tx.Description,
			tx.CreatedAt,
		})
	}
	writer.Flush()

	key := fmt.Sprintf("reports/%s/%s_%s.csv", userID, from, to)
	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(reportsBucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String("text/csv"),
	})
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("http://localhost:4566/%s/%s", reportsBucket, key)
	return url, nil
}
