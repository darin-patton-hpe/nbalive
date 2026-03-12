package nbalive

import (
	"context"
	"fmt"
)

// PlayByPlay fetches the play-by-play for a game.
func (c *Client) PlayByPlay(ctx context.Context, gameID string) (*PlayByPlayResponse, error) {
	var resp PlayByPlayResponse
	if err := c.get(ctx, fmt.Sprintf("playbyplay/playbyplay_%s.json", gameID), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
