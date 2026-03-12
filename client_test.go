package nbalive

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientNewClientDefaults(t *testing.T) {
	t.Parallel()

	c := NewClient()
	if c.baseURL != DefaultBaseURL {
		t.Fatalf("baseURL = %q, want %q", c.baseURL, DefaultBaseURL)
	}
	if c.httpClient == nil {
		t.Fatal("httpClient is nil, want non-nil")
	}
}

func TestClientOptions(t *testing.T) {
	t.Parallel()

	hc := &http.Client{Timeout: 5 * time.Second}
	base := "http://example.test"
	c := NewClient(WithHTTPClient(hc), WithBaseURL(base))

	if c.httpClient != hc {
		t.Fatal("WithHTTPClient did not set provided client")
	}
	if c.baseURL != base {
		t.Fatalf("baseURL = %q, want %q", c.baseURL, base)
	}
}

func TestClientGetViaScoreboard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		statusCode  int
		body        string
		wantErrPart string
		wantGameID  string
	}{
		{
			name:       "success decodes minimal json",
			statusCode: http.StatusOK,
			body: `{
				"meta": {"version": 1, "code": 200, "request": "ok", "time": "now"},
				"scoreboard": {
					"games": [
						{"gameId": "0001", "gameStatus": 2, "gameClock": "PT0M0.00S"}
					]
				}
			}`,
			wantGameID: "0001",
		},
		{
			name:        "404 status returns error",
			statusCode:  http.StatusNotFound,
			body:        `{"error":"missing"}`,
			wantErrPart: "status 404",
		},
		{
			name:        "500 status returns error",
			statusCode:  http.StatusInternalServerError,
			body:        `{"error":"boom"}`,
			wantErrPart: "status 500",
		},
		{
			name:        "invalid json returns decode error",
			statusCode:  http.StatusOK,
			body:        `{"meta":`,
			wantErrPart: "decode",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/scoreboard/todaysScoreboard_00.json" {
					t.Fatalf("path = %q, want %q", r.URL.Path, "/scoreboard/todaysScoreboard_00.json")
				}
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			c := NewClient(WithBaseURL(srv.URL))
			resp, err := c.Scoreboard(context.Background())

			if tc.wantErrPart != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErrPart)
				}
				if !strings.Contains(err.Error(), tc.wantErrPart) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tc.wantErrPart)
				}
				return
			}

			if err != nil {
				t.Fatalf("Scoreboard() error = %v", err)
			}
			if resp == nil || len(resp.Scoreboard.Games) != 1 {
				t.Fatalf("games length = %d, want 1", len(resp.Scoreboard.Games))
			}
			if got := resp.Scoreboard.Games[0].GameID; got != tc.wantGameID {
				t.Fatalf("gameId = %q, want %q", got, tc.wantGameID)
			}
		})
	}
}

func TestClientGetViaScoreboardContextCancellation(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"meta":{"version":1,"code":200},"scoreboard":{"games":[]}}`))
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.Scoreboard(ctx)
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
	if !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("error = %v, want context canceled", err)
	}
}

func TestClientGetIfModified(t *testing.T) {
	t.Parallel()

	path := "scoreboard/todaysScoreboard_00.json"

	t.Run("first request no etag returns modified true and etag", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/"+path {
				t.Fatalf("path = %q, want %q", r.URL.Path, "/"+path)
			}
			if got := r.Header.Get("If-None-Match"); got != "" {
				t.Fatalf("If-None-Match = %q, want empty", got)
			}
			w.Header().Set("ETag", "\"v1\"")
			_, _ = w.Write([]byte(`{"meta":{"version":1,"code":200},"scoreboard":{"games":[{"gameId":"g1","gameClock":"PT0M0.00S"}]}}`))
		}))
		defer srv.Close()

		c := NewClient(WithBaseURL(srv.URL))
		var dst ScoreboardResponse
		etag, modified, err := c.getIfModified(context.Background(), path, "", &dst)
		if err != nil {
			t.Fatalf("getIfModified() error = %v", err)
		}
		if !modified {
			t.Fatal("modified = false, want true")
		}
		if etag != "\"v1\"" {
			t.Fatalf("etag = %q, want %q", etag, "\"v1\"")
		}
		if len(dst.Scoreboard.Games) != 1 || dst.Scoreboard.Games[0].GameID != "g1" {
			t.Fatalf("decoded dst mismatch: %#v", dst.Scoreboard.Games)
		}
	})

	t.Run("second request with etag returns 304 and leaves dst untouched", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/"+path {
				t.Fatalf("path = %q, want %q", r.URL.Path, "/"+path)
			}
			if got := r.Header.Get("If-None-Match"); got != "\"v1\"" {
				t.Fatalf("If-None-Match = %q, want %q", got, "\"v1\"")
			}
			w.WriteHeader(http.StatusNotModified)
		}))
		defer srv.Close()

		c := NewClient(WithBaseURL(srv.URL))
		dst := ScoreboardResponse{Scoreboard: Scoreboard{Games: []Game{{GameID: "sentinel"}}}}
		etag, modified, err := c.getIfModified(context.Background(), path, "\"v1\"", &dst)
		if err != nil {
			t.Fatalf("getIfModified() error = %v", err)
		}
		if modified {
			t.Fatal("modified = true, want false")
		}
		if etag != "\"v1\"" {
			t.Fatalf("etag = %q, want original %q", etag, "\"v1\"")
		}
		if len(dst.Scoreboard.Games) != 1 || dst.Scoreboard.Games[0].GameID != "sentinel" {
			t.Fatalf("dst was unexpectedly modified: %#v", dst.Scoreboard.Games)
		}
	})

	t.Run("request with stale etag gets new data and etag", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/"+path {
				t.Fatalf("path = %q, want %q", r.URL.Path, "/"+path)
			}
			if got := r.Header.Get("If-None-Match"); got != "\"v1\"" {
				t.Fatalf("If-None-Match = %q, want %q", got, "\"v1\"")
			}
			w.Header().Set("ETag", "\"v2\"")
			_, _ = w.Write([]byte(`{"meta":{"version":1,"code":200},"scoreboard":{"games":[{"gameId":"g2","gameClock":"PT0M0.00S"}]}}`))
		}))
		defer srv.Close()

		c := NewClient(WithBaseURL(srv.URL))
		var dst ScoreboardResponse
		etag, modified, err := c.getIfModified(context.Background(), path, "\"v1\"", &dst)
		if err != nil {
			t.Fatalf("getIfModified() error = %v", err)
		}
		if !modified {
			t.Fatal("modified = false, want true")
		}
		if etag != "\"v2\"" {
			t.Fatalf("etag = %q, want %q", etag, "\"v2\"")
		}
		if len(dst.Scoreboard.Games) != 1 || dst.Scoreboard.Games[0].GameID != "g2" {
			t.Fatalf("decoded dst mismatch: %#v", dst.Scoreboard.Games)
		}
	})

	t.Run("server error returns err and modified false", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/"+path {
				t.Fatalf("path = %q, want %q", r.URL.Path, "/"+path)
			}
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"boom"}`))
		}))
		defer srv.Close()

		c := NewClient(WithBaseURL(srv.URL))
		var dst ScoreboardResponse
		etag, modified, err := c.getIfModified(context.Background(), path, "\"v1\"", &dst)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "status 500") {
			t.Fatalf("error = %q, want substring %q", err.Error(), "status 500")
		}
		if modified {
			t.Fatal("modified = true, want false")
		}
		if etag != "" {
			t.Fatalf("etag = %q, want empty", etag)
		}
	})
}

func TestClientScoreboard(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scoreboard/todaysScoreboard_00.json" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/scoreboard/todaysScoreboard_00.json")
		}
		_, _ = w.Write([]byte(`{
			"meta": {"version": 1, "code": 200},
			"scoreboard": {
				"games": [{"gameId":"0022300001","gameStatus":2,"gameClock":"PT0M0.00S"}]
			}
		}`))
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL))
	resp, err := c.Scoreboard(context.Background())
	if err != nil {
		t.Fatalf("Scoreboard() error = %v", err)
	}
	if got := len(resp.Scoreboard.Games); got != 1 {
		t.Fatalf("len(games) = %d, want 1", got)
	}
	if got := resp.Scoreboard.Games[0].GameID; got != "0022300001" {
		t.Fatalf("gameId = %q, want %q", got, "0022300001")
	}
}

func TestClientPlayByPlay(t *testing.T) {
	t.Parallel()

	const gameID = "0022300002"
	expectedPath := fmt.Sprintf("/playbyplay/playbyplay_%s.json", gameID)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != expectedPath {
			t.Fatalf("path = %q, want %q", r.URL.Path, expectedPath)
		}
		_, _ = w.Write([]byte(`{"meta":{"version":1,"code":200},"game":{"gameId":"0022300002","actions":[]}}`))
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL))
	resp, err := c.PlayByPlay(context.Background(), gameID)
	if err != nil {
		t.Fatalf("PlayByPlay() error = %v", err)
	}
	if resp.Game.GameID != gameID {
		t.Fatalf("gameId = %q, want %q", resp.Game.GameID, gameID)
	}
}

func TestClientBoxScore(t *testing.T) {
	t.Parallel()

	const gameID = "0022300003"
	expectedPath := fmt.Sprintf("/boxscore/boxscore_%s.json", gameID)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != expectedPath {
			t.Fatalf("path = %q, want %q", r.URL.Path, expectedPath)
		}
		_, _ = w.Write([]byte(`{"meta":{"version":1,"code":200},"game":{"gameId":"0022300003","gameClock":"PT0M0.00S"}}`))
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL))
	resp, err := c.BoxScore(context.Background(), gameID)
	if err != nil {
		t.Fatalf("BoxScore() error = %v", err)
	}
	if resp.Game.GameID != gameID {
		t.Fatalf("gameId = %q, want %q", resp.Game.GameID, gameID)
	}
}

func TestClientLiveGames(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scoreboard/todaysScoreboard_00.json" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/scoreboard/todaysScoreboard_00.json")
		}
		_, _ = w.Write([]byte(`{
			"meta": {"version": 1, "code": 200},
			"scoreboard": {
				"games": [
					{"gameId":"g1","gameStatus":1,"gameClock":"PT0M0.00S"},
					{"gameId":"g2","gameStatus":2,"gameClock":"PT11M0.00S"},
					{"gameId":"g3","gameStatus":3,"gameClock":"PT0M0.00S"}
				]
			}
		}`))
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL))
	live, err := c.LiveGames(context.Background())
	if err != nil {
		t.Fatalf("LiveGames() error = %v", err)
	}
	if len(live) != 1 {
		t.Fatalf("len(live) = %d, want 1", len(live))
	}
	if live[0].GameID != "g2" {
		t.Fatalf("live[0].gameId = %q, want %q", live[0].GameID, "g2")
	}
	if live[0].GameStatus != GameInProgress {
		t.Fatalf("live[0].gameStatus = %d, want %d", live[0].GameStatus, GameInProgress)
	}
}
