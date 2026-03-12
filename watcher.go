package nbalive

import (
	"context"
	"fmt"
	"time"
)

// EventKind discriminates events from Watch.
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

// Event is a tagged union emitted by Watch.
// Exactly one of Action, BoxScore, or Err is non-nil depending on Kind.
type Event struct {
	Kind     EventKind
	GameID   string
	Action   *Action       // non-nil when Kind == EventAction
	BoxScore *BoxScoreGame // non-nil when Kind == EventBoxScore or EventGameOver
	Err      error         // non-nil when Kind == EventError
}

// WatchConfig configures a game watcher.
type WatchConfig struct {
	// PollInterval between fetches. Default: 15s. Minimum enforced: 5s.
	PollInterval time.Duration
	// BoxScore controls whether box scores are fetched each tick.
	// When true, EventBoxScore events are emitted alongside actions.
	BoxScore bool
}

func (cfg *WatchConfig) withDefaults() {
	if cfg.PollInterval < 5*time.Second {
		cfg.PollInterval = 15 * time.Second
	}
}

// Watch polls a live game and emits events on the returned channel.
// The channel is closed when the game ends or the context is cancelled.
//
// Event stream semantics:
//   - EventAction:   one per new play-by-play action (deduplicated by orderNumber)
//   - EventBoxScore: one per poll tick (if WatchConfig.BoxScore is true)
//   - EventGameOver: emitted once when gameStatus == Final, then channel closes
//   - EventError:    transient fetch/decode errors; watcher continues polling
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

			// --- Play-by-play ---
			var pbp PlayByPlayResponse
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

			// --- Box Score (optional) ---
			if cfg.BoxScore {
				var bs BoxScoreResponse
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
					if bs.Game.GameStatus == GameFinal {
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

// emit sends an event, respecting context cancellation. Returns false if ctx is done.
func emit(ctx context.Context, ch chan<- Event, e Event) bool {
	select {
	case ch <- e:
		return true
	case <-ctx.Done():
		return false
	}
}
