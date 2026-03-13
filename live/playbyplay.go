package live

import (
	"context"
	"fmt"

	"github.com/darin-patton-hpe/nbalive"
)

func (c *Client) PlayByPlay(ctx context.Context, gameID string) (*nbalive.PlayByPlayResponse, error) {
	var resp nbalive.PlayByPlayResponse
	if err := c.get(ctx, fmt.Sprintf("playbyplay/playbyplay_%s.json", gameID), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
