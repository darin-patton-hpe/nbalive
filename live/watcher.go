package live

import (
	"context"
	"fmt"
	"time"

	"github.com/darin-patton-hpe/nbalive"
)

type EventKind int

const (
	EventAction EventKind = iota
	EventBoxScore
	EventGameOver
	EventError
)

func (k EventKind) String() string {
	switch k {
	case EventAction:
		return "Action"
	case EventBoxScore:
		return "BoxScore"
	case EventGameOver:
		return "GameOver"
	case EventError:
		return "Error"
	default:
		return "Unknown"
	}
}

type Event struct {
	Kind     EventKind
	GameID   string
	Action   *nbalive.Action
	BoxScore *nbalive.BoxScoreGame
	Err      error
}

type WatchConfig struct {
	PollInterval time.Duration
	BoxScore     bool
}

func (cfg *WatchConfig) withDefaults() {
	if cfg.PollInterval < 5*time.Second {
		cfg.PollInterval = 15 * time.Second
	}
}

func (c *Client) Watch(ctx context.Context, gameID string, cfg WatchConfig) <-chan Event {
	cfg.withDefaults()
	ch := make(chan Event, 64)

	go func() {
		defer close(ch)

		ticker := time.NewTicker(cfg.PollInterval)
		defer ticker.Stop()

		var (
			lastOrder int
			pbpETag   string
			bsETag    string
		)

		pbpPath := fmt.Sprintf("playbyplay/playbyplay_%s.json", gameID)
		bsPath := fmt.Sprintf("boxscore/boxscore_%s.json", gameID)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			var pbp nbalive.PlayByPlayResponse
			newETag, modified, err := c.getIfModified(ctx, pbpPath, pbpETag, &pbp)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				emit(ctx, ch, Event{Kind: EventError, GameID: gameID, Err: err})
				continue
			}
			if modified {
				pbpETag = newETag
				for i := range pbp.Game.Actions {
					a := &pbp.Game.Actions[i]
					if a.OrderNumber > lastOrder {
						if !emit(ctx, ch, Event{Kind: EventAction, GameID: gameID, Action: a}) {
							return
						}
						lastOrder = a.OrderNumber
					}
				}
			}

			if cfg.BoxScore {
				var bs nbalive.BoxScoreResponse
				newETag, modified, err := c.getIfModified(ctx, bsPath, bsETag, &bs)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					emit(ctx, ch, Event{Kind: EventError, GameID: gameID, Err: err})
					continue
				}
				if modified {
					bsETag = newETag
					if bs.Game.GameStatus == nbalive.GameFinal {
						emit(ctx, ch, Event{Kind: EventGameOver, GameID: gameID, BoxScore: &bs.Game})
						return
					}
					emit(ctx, ch, Event{Kind: EventBoxScore, GameID: gameID, BoxScore: &bs.Game})
				}
			}
		}
	}()

	return ch
}

func emit(ctx context.Context, ch chan<- Event, e Event) bool {
	select {
	case ch <- e:
		return true
	case <-ctx.Done():
		return false
	}
}
