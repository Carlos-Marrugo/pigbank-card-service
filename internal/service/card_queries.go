package service

import (
	"context"
	"fmt"

	"github.com/Carlos-Marrugo/pigbank-card-service/internal/models"
)

func GetUserCards(ctx context.Context, userID string) ([]models.Card, error) {
	if cardRepo == nil {
		return nil, fmt.Errorf("repository not initialized")
	}
	return cardRepo.FindCardsByUser(ctx, userID)
}

func GetCardBalance(ctx context.Context, cardID, userID string) (float64, error) {
	if cardRepo == nil {
		return 0, fmt.Errorf("repository not initialized")
	}

	card, err := cardRepo.FindCardByID(ctx, cardID)
	if err != nil {
		return 0, err
	}

	if card.UserID != userID {
		return 0, fmt.Errorf("card does not belong to user")
	}

	return card.Balance, nil
}
