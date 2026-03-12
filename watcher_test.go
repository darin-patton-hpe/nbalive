package nbalive

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

const testGameID = "0022400001"

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResponse(status int, etag string, v any) *http.Response {
	b, _ := json.Marshal(v)
	resp := &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(b)),
	}
	if etag != "" {
		resp.Header.Set("ETag", etag)
	}
	return resp
}

func TestWatchNewActionsAreEmittedAndDeduplicated(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var reqCount atomic.Int32

		client := NewClient(
			WithHTTPClient(&http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					n := reqCount.Add(1)
					if n == 1 {
						return jsonResponse(200, "p1", PlayByPlayResponse{
							Game: PlayByPlayGame{GameID: testGameID, Actions: []Action{
								{OrderNumber: 1}, {OrderNumber: 2}, {OrderNumber: 3},
							}},
						}), nil
					}
					return jsonResponse(200, "p2", PlayByPlayResponse{
						Game: PlayByPlayGame{GameID: testGameID, Actions: []Action{
							{OrderNumber: 1}, {OrderNumber: 2}, {OrderNumber: 3},
							{OrderNumber: 4}, {OrderNumber: 5},
						}},
					}), nil
				}),
			}),
		)

		ch := client.Watch(t.Context(), testGameID, WatchConfig{PollInterval: 5 * time.Second})

		synctest.Wait()
		var orders []int
		for range 3 {
			ev := <-ch
			if ev.Kind != EventAction {
				t.Fatalf("expected EventAction, got %v", ev.Kind)
			}
			orders = append(orders, ev.Action.OrderNumber)
		}

		synctest.Wait()
		for range 2 {
			ev := <-ch
			if ev.Kind != EventAction {
				t.Fatalf("expected EventAction, got %v", ev.Kind)
			}
			orders = append(orders, ev.Action.OrderNumber)
		}

		want := []int{1, 2, 3, 4, 5}
		for i := range want {
			if orders[i] != want[i] {
				t.Fatalf("orders[%d]=%d, want %d; full=%v", i, orders[i], want[i], orders)
			}
		}
	})
}

func TestWatchETag304NoDuplicateEvents(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var reqCount atomic.Int32
		var ifNoneMatchSeen atomic.Bool

		client := NewClient(
			WithHTTPClient(&http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					n := reqCount.Add(1)
					if n == 1 {
						return jsonResponse(200, "etag-pbp", PlayByPlayResponse{
							Game: PlayByPlayGame{GameID: testGameID, Actions: []Action{{OrderNumber: 11}}},
						}), nil
					}
					if r.Header.Get("If-None-Match") == "etag-pbp" {
						ifNoneMatchSeen.Store(true)
					}
					return &http.Response{
						StatusCode: http.StatusNotModified,
						Header:     http.Header{"ETag": {"etag-pbp"}},
						Body:       io.NopCloser(bytes.NewReader(nil)),
					}, nil
				}),
			}),
		)

		const poll = 5 * time.Second
		ch := client.Watch(t.Context(), testGameID, WatchConfig{PollInterval: poll})

		synctest.Wait()
		ev := <-ch
		if ev.Kind != EventAction || ev.Action == nil || ev.Action.OrderNumber != 11 {
			t.Fatalf("unexpected first event: %+v", ev)
		}

		// Advance fake clock past the next tick, then wait for the
		// goroutine to finish its HTTP round-trip and block again.
		time.Sleep(poll)
		synctest.Wait()

		select {
		case ev := <-ch:
			t.Fatalf("unexpected event after 304: %+v", ev)
		default:
		}

		if reqCount.Load() < 2 {
			t.Fatalf("expected at least 2 requests, got %d", reqCount.Load())
		}
		if !ifNoneMatchSeen.Load() {
			t.Fatal("expected If-None-Match header on second request")
		}
	})
}

func TestWatchGameOverEmitsAndClosesChannel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := NewClient(
			WithBaseURL("http://fake"),
			WithHTTPClient(&http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					switch {
					case r.URL.Path == "/playbyplay/playbyplay_"+testGameID+".json":
						return jsonResponse(200, "p1", PlayByPlayResponse{
							Game: PlayByPlayGame{GameID: testGameID},
						}), nil
					case r.URL.Path == "/boxscore/boxscore_"+testGameID+".json":
						return jsonResponse(200, "b1", BoxScoreResponse{
							Game: BoxScoreGame{GameID: testGameID, GameStatus: GameFinal},
						}), nil
					default:
						return &http.Response{StatusCode: 404, Body: io.NopCloser(bytes.NewReader(nil))}, nil
					}
				}),
			}),
		)

		ch := client.Watch(t.Context(), testGameID, WatchConfig{
			PollInterval: 5 * time.Second,
			BoxScore:     true,
		})

		synctest.Wait()
		ev := <-ch
		if ev.Kind != EventGameOver {
			t.Fatalf("expected EventGameOver, got %v", ev.Kind)
		}
		if ev.BoxScore == nil || ev.BoxScore.GameStatus != GameFinal {
			t.Fatalf("expected final box score payload, got %+v", ev.BoxScore)
		}

		_, ok := <-ch
		if ok {
			t.Fatal("expected channel to be closed after game over")
		}
	})
}

func TestWatchContextCancellationClosesChannel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := NewClient(
			WithHTTPClient(&http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: 404, Body: io.NopCloser(bytes.NewReader(nil))}, nil
				}),
			}),
		)

		ctx, cancel := context.WithCancel(t.Context())
		ch := client.Watch(ctx, testGameID, WatchConfig{PollInterval: 5 * time.Second})

		cancel()
		synctest.Wait()

		_, ok := <-ch
		if ok {
			t.Fatal("expected closed channel after cancellation")
		}
	})
}

func TestWatchErrorEventThenContinuesPolling(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var reqCount atomic.Int32

		client := NewClient(
			WithHTTPClient(&http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					n := reqCount.Add(1)
					if n == 1 {
						return &http.Response{
							StatusCode: http.StatusInternalServerError,
							Body:       io.NopCloser(bytes.NewReader(nil)),
						}, nil
					}
					return jsonResponse(200, "p2", PlayByPlayResponse{
						Game: PlayByPlayGame{GameID: testGameID, Actions: []Action{{OrderNumber: 21}}},
					}), nil
				}),
			}),
		)

		ch := client.Watch(t.Context(), testGameID, WatchConfig{PollInterval: 5 * time.Second})

		synctest.Wait()
		ev := <-ch
		if ev.Kind != EventError || ev.Err == nil {
			t.Fatalf("expected EventError on first tick, got %+v", ev)
		}

		synctest.Wait()
		ev = <-ch
		if ev.Kind != EventAction || ev.Action == nil || ev.Action.OrderNumber != 21 {
			t.Fatalf("expected EventAction after error recovery, got %+v", ev)
		}
	})
}

func TestWatchConfigWithDefaults(t *testing.T) {
	t.Run("below minimum uses default", func(t *testing.T) {
		cfg := WatchConfig{PollInterval: 50 * time.Millisecond}
		cfg.withDefaults()
		if cfg.PollInterval != 15*time.Second {
			t.Fatalf("PollInterval=%v, want %v", cfg.PollInterval, 15*time.Second)
		}
	})

	t.Run("minimum and above are preserved", func(t *testing.T) {
		cfg := WatchConfig{PollInterval: 5 * time.Second}
		cfg.withDefaults()
		if cfg.PollInterval != 5*time.Second {
			t.Fatalf("PollInterval=%v, want %v", cfg.PollInterval, 5*time.Second)
		}

		cfg = WatchConfig{PollInterval: 7 * time.Second}
		cfg.withDefaults()
		if cfg.PollInterval != 7*time.Second {
			t.Fatalf("PollInterval=%v, want %v", cfg.PollInterval, 7*time.Second)
		}
	})
}
