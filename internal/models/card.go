package models

type Card struct {
	CardID    string  `json:"card_id" dynamodbav:"card_id"`
	UserID    string  `json:"user_id" dynamodbav:"user_id"`
	CardType  string  `json:"card_type" dynamodbav:"card_type"`
	Balance   float64 `json:"balance" dynamodbav:"balance"`
	Limit     float64 `json:"limit" dynamodbav:"limit"`
	Status    string  `json:"status" dynamodbav:"status"`
	CreatedAt string  `json:"created_at" dynamodbav:"created_at"`
}

type Transaction struct {
	TransactionID string  `json:"transaction_id" dynamodbav:"transaction_id"`
	CardID        string  `json:"card_id" dynamodbav:"card_id"`
	UserID        string  `json:"user_id" dynamodbav:"user_id"`
	Type          string  `json:"type" dynamodbav:"type"`
	Amount        float64 `json:"amount" dynamodbav:"amount"`
	Description   string  `json:"description" dynamodbav:"description"`
	CreatedAt     string  `json:"created_at" dynamodbav:"created_at"`
}

type CardRequest struct {
	UserID  string `json:"userId"`
	Request string `json:"request"`
}

type TransactionRequest struct {
	CardID      string  `json:"card_id"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	Type        string  `json:"type"` 
}

type ReportRequest struct {
	UserID string `json:"user_id"`
	From   string `json:"from"`
	To     string `json:"to"`
}
