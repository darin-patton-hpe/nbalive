package nbalive

import (
	"context"
	"fmt"
)

// BoxScore fetches the box score for a game.
func (c *Client) BoxScore(ctx context.Context, gameID string) (*BoxScoreResponse, error) {
	var resp BoxScoreResponse
	if err := c.get(ctx, fmt.Sprintf("boxscore/boxscore_%s.json", gameID), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
