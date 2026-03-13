package live

import (
	"context"
	"fmt"

	"github.com/darin-patton-hpe/nbalive"
)

func (c *Client) BoxScore(ctx context.Context, gameID string) (*nbalive.BoxScoreResponse, error) {
	var resp nbalive.BoxScoreResponse
	if err := c.get(ctx, fmt.Sprintf("boxscore/boxscore_%s.json", gameID), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
