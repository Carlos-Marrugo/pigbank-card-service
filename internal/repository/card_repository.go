package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Carlos-Marrugo/pigbank-card-service/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

type CardRepository struct {
	client     *dynamodb.Client
	cardsTable string
	txTable    string
}

func NewCardRepository(client *dynamodb.Client, cardsTable, txTable string) *CardRepository {
	return &CardRepository{
		client:     client,
		cardsTable: cardsTable,
		txTable:    txTable,
	}
}

func (r *CardRepository) CreateCard(ctx context.Context, card models.Card) error {
	card.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	item, err := attributevalue.MarshalMap(card)
	if err != nil {
		return err
	}
	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.cardsTable),
		Item:      item,
	})
	return err
}

func (r *CardRepository) SaveTransaction(ctx context.Context, tx models.Transaction) error {
	tx.TransactionID = uuid.New().String()
	tx.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	item, err := attributevalue.MarshalMap(tx)
	if err != nil {
		return err
	}
	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.txTable),
		Item:      item,
	})
	return err
}

func (r *CardRepository) UpdateCardBalance(ctx context.Context, cardID string, newBalance float64) error {
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(r.cardsTable),
		Key: map[string]types.AttributeValue{
			"card_id": &types.AttributeValueMemberS{Value: cardID},
		},
		UpdateExpression: aws.String("SET balance = :b"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":b": &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", newBalance)},
		},
	}
	_, err := r.client.UpdateItem(ctx, input)
	return err
}

func (r *CardRepository) FindCardByID(ctx context.Context, cardID string) (*models.Card, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(r.cardsTable),
		Key: map[string]types.AttributeValue{
			"card_id": &types.AttributeValueMemberS{Value: cardID},
		},
	}

	result, err := r.client.GetItem(ctx, input)
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, fmt.Errorf("card not found")
	}

	var card models.Card
	err = attributevalue.UnmarshalMap(result.Item, &card)
	return &card, err
}

func (r *CardRepository) FindCardsByUser(ctx context.Context, userID string) ([]models.Card, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.cardsTable),
		IndexName:              aws.String("UserIndex"),
		KeyConditionExpression: aws.String("user_id = :u"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":u": &types.AttributeValueMemberS{Value: userID},
		},
	}

	result, err := r.client.Query(ctx, input)
	if err != nil {
		return nil, err
	}

	var cards []models.Card
	err = attributevalue.UnmarshalListOfMaps(result.Items, &cards)
	return cards, err
}

func (r *CardRepository) GetTransactionsByUser(ctx context.Context, userID string, from, to string) ([]models.Transaction, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.txTable),
		IndexName:              aws.String("UserIndex"),
		KeyConditionExpression: aws.String("user_id = :u AND created_at BETWEEN :from AND :to"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":u":    &types.AttributeValueMemberS{Value: userID},
			":from": &types.AttributeValueMemberS{Value: from},
			":to":   &types.AttributeValueMemberS{Value: to},
		},
	}

	result, err := r.client.Query(ctx, input)
	if err != nil {
		return nil, err
	}

	var transactions []models.Transaction
	err = attributevalue.UnmarshalListOfMaps(result.Items, &transactions)
	return transactions, err
}
