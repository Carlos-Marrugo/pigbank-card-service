# Pig Bank - Card Service (Go)

## Description

Event-driven microservice for automatic financial card management. Listens to SQS events triggered by user registration and automatically creates debit and credit cards with dynamic credit scoring.

---

## Features

### Automatic Card Creation (SQS Worker)

* Listens to `create-request-card-sqs` queue
* Generates random credit score (0–100)
* Calculates credit limit: `100 + (score/100) * (10,000,000 - 100)`
* Creates debit card (active, balance = 0)
* Creates credit card with calculated limit
* Sends welcome event to notification queue

---

### Card Management (Planned)

* Credit limit validation for purchases
* Balance updates for deposits and payments
* Transaction history

---

### Transaction Processing (Planned)

* Purchase validation (debit balance / credit limit)
* Deposit processing
* Credit card payment
* Monthly statement generation
* CSV report export to S3

---

## Architecture

```
User Service → SQS (card-request) → Card Service → DynamoDB (cards/transactions)
                                              ↓
                                      SQS (notification) → Notification Service
```

---

## Tech Stack

* Language: Go 1.25
* Database: DynamoDB (`pigbank-cards`, `pigbank-transactions`)
* Messaging: AWS SQS (with DLQ)
* Infrastructure: Terraform + LocalStack

---

## API Endpoints (Planned)

| Method | Endpoint                          | Description                           |
| ------ | --------------------------------- | ------------------------------------- |
| POST   | `/api/v1/cards/credit`            | Create credit card manually           |
| POST   | `/api/v1/transactions`            | Process purchase, deposit, or payment |
| GET    | `/api/v1/transactions/report`     | Generate CSV report to S3             |
| GET    | `/api/v1/cards/{card_id}/balance` | Check balance or credit limit         |

---

## Database Schema

### Cards Table (`pigbank-cards`)

```json
{
  "card_id": "uuid (PK)",
  "user_id": "uuid (GSI)",
  "card_type": "DEBIT | CREDIT",
  "balance": 0.00,
  "limit": 1000000.00,
  "status": "ACTIVE | BLOCKED",
  "created_at": "timestamp"
}
```

### Transactions Table (`pigbank-transactions`)

```json
{
  "transaction_id": "uuid (PK)",
  "card_id": "uuid (GSI)",
  "user_id": "uuid (GSI)",
  "type": "PURCHASE | DEPOSIT | PAYMENT",
  "amount": 500.00,
  "description": "Store purchase",
  "created_at": "timestamp"
}
```

---

## SQS Events

### Input (from User Service)

```json
{
  "userId": "uuid",
  "request": "DEBIT"
}
```

### Output (to Notification Service)

```json
{
  "user_id": "uuid",
  "type": "WELCOME",
  "score": 18,
  "credit_limit": 1800082.00,
  "email": "user@example.com"
}
```

---

## Local Development

### Prerequisites

* LocalStack running on port 4566
* DynamoDB tables: `pigbank-cards`, `pigbank-transactions`
* SQS queues: `create-request-card-sqs`, `notification-email-sqs`

---

### Environment Variables (`.env`)

```env
CARDS_TABLE=pigbank-cards
TRANSACTIONS_TABLE=pigbank-transactions
CARD_REQUEST_QUEUE_URL=http://sqs.localhost:4566/000000000000/create-request-card-sqs
NOTIFICATION_QUEUE_URL=http://sqs.localhost:4566/000000000000/notification-email-sqs
```

---

### Run Service

```bash
go mod tidy
go run main.go
```

---

### Test with User Registration

```bash
curl -X POST http://localhost:8081/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{"name":"John","last_name":"Doe","email":"john@test.com","password":"123","document":"123456"}'
```

---

## Resilience

* Dead Letter Queue (DLQ) for failed messages (3 retries)
* Long polling in SQS (10 seconds)
* Graceful shutdown handling
* Automatic retry on DynamoDB errors

---

## Credit Scoring Formula

```go
score = random(0, 100)
creditLimit = 100 + (score/100) * (10_000_000 - 100)
```

Examples:

* Score 0 → 100
* Score 50 → 5,000,050
* Score 100 → 10,000,000

---
