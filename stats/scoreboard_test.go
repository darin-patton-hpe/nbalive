package stats

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const scoreboardV3Fixture = `
{
    "meta": {"version": 1, "code": 200, "request": "scoreboardv3", "time": "2024-11-15"},
    "scoreboard": {
        "gameDate": "2024-11-15",
        "leagueId": "00",
        "leagueName": "National Basketball Association",
        "games": [
            {
                "gameId": "0022400171",
                "gameCode": "20241115/PHICLE",
                "gameStatus": 3,
                "gameStatusText": "Final",
                "period": 4,
                "gameClock": "PT0M0.00S",
                "gameTimeUTC": "2024-11-16T00:00:00Z",
                "gameEt": "2024-11-15T19:00:00-05:00",
                "regulationPeriods": 4,
                "seriesGameNumber": "",
                "seriesText": "",
                "homeTeam": {
                    "teamId": 1610612739,
                    "teamName": "Cavaliers",
                    "teamCity": "Cleveland",
                    "teamTricode": "CLE",
                    "wins": 15,
                    "losses": 0,
                    "score": 114,
                    "periods": [{"period": 1, "periodType": "REGULAR", "score": 28}]
                },
                "awayTeam": {
                    "teamId": 1610612755,
                    "teamName": "76ers",
                    "teamCity": "Philadelphia",
                    "teamTricode": "PHI",
                    "wins": 3,
                    "losses": 12,
                    "score": 106,
                    "periods": [{"period": 1, "periodType": "REGULAR", "score": 22}]
                },
                "gameLeaders": {
                    "homeLeaders": {
                        "personId": 1628386,
                        "name": "Jarrett Allen",
                        "jerseyNum": "31",
                        "position": "C",
                        "teamTricode": "CLE",
                        "points": 25,
                        "rebounds": 10,
                        "assists": 2
                    },
                    "awayLeaders": {
                        "personId": 203954,
                        "name": "Joel Embiid",
                        "jerseyNum": "21",
                        "position": "C",
                        "teamTricode": "PHI",
                        "points": 30,
                        "rebounds": 8,
                        "assists": 3
                    }
                }
            }
        ]
    }
}
`

func TestScoreboardByDateSuccess(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scoreboardv3" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/scoreboardv3")
		}
		if got := r.URL.Query().Get("GameDate"); got != "2024-11-15" {
			t.Fatalf("GameDate = %q, want %q", got, "2024-11-15")
		}
		if got := r.URL.Query().Get("LeagueID"); got != "00" {
			t.Fatalf("LeagueID = %q, want %q", got, "00")
		}
		if got := r.Header.Get("Origin"); got != "https://www.nba.com" {
			t.Fatalf("Origin = %q, want %q", got, "https://www.nba.com")
		}
		if got := r.Header.Get("Referer"); got != "https://www.nba.com/" {
			t.Fatalf("Referer = %q, want %q", got, "https://www.nba.com/")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(scoreboardV3Fixture))
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL))
	resp, err := c.ScoreboardByDate(context.Background(), "2024-11-15")
	if err != nil {
		t.Fatalf("ScoreboardByDate() error = %v", err)
	}
	if resp == nil || len(resp.Scoreboard.Games) != 1 {
		t.Fatalf("games length = %d, want 1", len(resp.Scoreboard.Games))
	}
	game := resp.Scoreboard.Games[0]
	if game.GameID != "0022400171" {
		t.Fatalf("gameID = %q, want %q", game.GameID, "0022400171")
	}
	if game.HomeTeam.TeamTricode != "CLE" || game.AwayTeam.TeamTricode != "PHI" {
		t.Fatalf("unexpected tricodes: home=%q away=%q", game.HomeTeam.TeamTricode, game.AwayTeam.TeamTricode)
	}
}

func TestScoreboardByDateInvalidDateFormat(t *testing.T) {
	t.Parallel()

	c := NewClient()
	badDates := []string{"2024/01/01", "01-15-2024", "20240115", "", "abc"}
	for _, d := range badDates {
		d := d
		t.Run(d, func(t *testing.T) {
			t.Parallel()
			_, err := c.ScoreboardByDate(context.Background(), d)
			if err == nil {
				t.Fatal("expected invalid date format error, got nil")
			}
			if !strings.Contains(err.Error(), "invalid date format") {
				t.Fatalf("error = %q, want substring %q", err.Error(), "invalid date format")
			}
		})
	}
}

func TestScoreboardByDateServer404(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL))
	_, err := c.ScoreboardByDate(context.Background(), "2024-11-15")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "status 404") {
		t.Fatalf("error = %q, want substring %q", err.Error(), "status 404")
	}
}

func TestScoreboardByDateServer500(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL))
	_, err := c.ScoreboardByDate(context.Background(), "2024-11-15")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("error = %q, want substring %q", err.Error(), "status 500")
	}
}

func TestScoreboardByDateInvalidJSON(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"meta":`))
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL))
	_, err := c.ScoreboardByDate(context.Background(), "2024-11-15")
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Fatalf("error = %q, want substring %q", err.Error(), "decode")
	}
}

func TestScoreboardByDateContextCancellation(t *testing.T) {
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

	_, err := c.ScoreboardByDate(ctx, "2024-11-15")
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
	if !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("error = %v, want context canceled", err)
	}
}
