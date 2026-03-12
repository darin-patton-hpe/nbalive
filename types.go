package nbalive

import "encoding/json"

// GameStatus represents the state of a game.
type GameStatus int

const (
	GameScheduled  GameStatus = 1
	GameInProgress GameStatus = 2
	GameFinal      GameStatus = 3
)

func (s GameStatus) String() string {
	switch s {
	case GameScheduled:
		return "Scheduled"
	case GameInProgress:
		return "In Progress"
	case GameFinal:
		return "Final"
	default:
		return "Unknown"
	}
}

// BoolString handles NBA's "1"/"0" JSON string booleans.
type BoolString bool

func (b *BoolString) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		// Try as actual bool.
		var raw bool
		if err2 := json.Unmarshal(data, &raw); err2 != nil {
			return err
		}
		*b = BoolString(raw)
		return nil
	}
	*b = BoolString(s == "1")
	return nil
}

// Bool returns the underlying boolean value.
func (b BoolString) Bool() bool { return bool(b) }

// ---- Response Metadata ----

// Meta is the response metadata envelope present in all CDN responses.
type Meta struct {
	Version int    `json:"version"`
	Code    int    `json:"code"`
	Request string `json:"request"`
	Time    string `json:"time"`
}

// ---- Scoreboard ----

// ScoreboardResponse is the top-level response from the scoreboard endpoint.
type ScoreboardResponse struct {
	Meta       Meta       `json:"meta"`
	Scoreboard Scoreboard `json:"scoreboard"`
}

// Scoreboard contains today's games.
type Scoreboard struct {
	GameDate   string `json:"gameDate"`
	LeagueID   string `json:"leagueId"`
	LeagueName string `json:"leagueName"`
	Games      []Game `json:"games"`
}

// Game is a single game entry on the scoreboard.
type Game struct {
	GameID            string     `json:"gameId"`
	GameCode          string     `json:"gameCode"`
	GameStatus        GameStatus `json:"gameStatus"`
	GameStatusText    string     `json:"gameStatusText"`
	Period            int        `json:"period"`
	GameClock         Duration   `json:"gameClock"`
	GameTimeUTC       string     `json:"gameTimeUTC"`
	GameET            string     `json:"gameEt"`
	RegulationPeriods int        `json:"regulationPeriods"`
	SeriesGameNumber  string     `json:"seriesGameNumber"`
	SeriesText        string     `json:"seriesText"`
	HomeTeam          GameTeam   `json:"homeTeam"`
	AwayTeam          GameTeam   `json:"awayTeam"`
	GameLeaders       struct {
		HomeLeaders PlayerLeader `json:"homeLeaders"`
		AwayLeaders PlayerLeader `json:"awayLeaders"`
	} `json:"gameLeaders"`
}

// GameTeam is a team entry on the scoreboard (lighter than BoxTeam).
type GameTeam struct {
	TeamID            int      `json:"teamId"`
	TeamName          string   `json:"teamName"`
	TeamCity          string   `json:"teamCity"`
	TeamTricode       string   `json:"teamTricode"`
	Wins              int      `json:"wins"`
	Losses            int      `json:"losses"`
	Score             int      `json:"score"`
	InBonus           *string  `json:"inBonus"`
	TimeoutsRemaining int      `json:"timeoutsRemaining"`
	Periods           []Period `json:"periods"`
}

// Period is a single period score entry.
type Period struct {
	Period     int    `json:"period"`
	PeriodType string `json:"periodType"`
	Score      int    `json:"score"`
}

// PlayerLeader is the game leader summary shown on the scoreboard.
type PlayerLeader struct {
	PersonID    int    `json:"personId"`
	Name        string `json:"name"`
	JerseyNum   string `json:"jerseyNum"`
	Position    string `json:"position"`
	TeamTricode string `json:"teamTricode"`
	Points      int    `json:"points"`
	Rebounds    int    `json:"rebounds"`
	Assists     int    `json:"assists"`
}

// ---- Play-by-Play ----

// PlayByPlayResponse is the top-level response from the play-by-play endpoint.
type PlayByPlayResponse struct {
	Meta Meta           `json:"meta"`
	Game PlayByPlayGame `json:"game"`
}

// PlayByPlayGame contains the game ID and all actions.
type PlayByPlayGame struct {
	GameID  string   `json:"gameId"`
	Actions []Action `json:"actions"`
}

// Action is a single play-by-play event (shot, foul, turnover, etc.).
type Action struct {
	ActionNumber int      `json:"actionNumber"`
	Clock        Duration `json:"clock"`
	TimeActual   string   `json:"timeActual"`
	Period       int      `json:"period"`
	PeriodType   string   `json:"periodType"`
	ActionType   string   `json:"actionType"`
	SubType      string   `json:"subType"`
	Descriptor   string   `json:"descriptor"`
	Qualifiers   []string `json:"qualifiers"`
	Description  string   `json:"description"`

	TeamID      int    `json:"teamId"`
	TeamTricode string `json:"teamTricode"`
	PersonID    int    `json:"personId"`
	PlayerName  string `json:"playerName"`
	PlayerNameI string `json:"playerNameI"`

	// Shot fields — nil for non-shot actions (check IsFieldGoal).
	X            *float64 `json:"x"`
	Y            *float64 `json:"y"`
	XLegacy      *int     `json:"xLegacy"`
	YLegacy      *int     `json:"yLegacy"`
	IsFieldGoal  int      `json:"isFieldGoal"`
	ShotResult   string   `json:"shotResult"`
	ShotDistance *float64 `json:"shotDistance"`

	ScoreHome string `json:"scoreHome"`
	ScoreAway string `json:"scoreAway"`

	Possession      int   `json:"possession"`
	OrderNumber     int   `json:"orderNumber"`
	PersonIDsFilter []int `json:"personIdsFilter"`
}

// IsMade returns true if this is a made field goal or free throw.
func (a Action) IsMade() bool { return a.ShotResult == "Made" }

// HasCoords returns true if this action has shot location data.
func (a Action) HasCoords() bool { return a.X != nil && a.Y != nil }

// ---- Box Score ----

// BoxScoreResponse is the top-level response from the box score endpoint.
type BoxScoreResponse struct {
	Meta Meta         `json:"meta"`
	Game BoxScoreGame `json:"game"`
}

// BoxScoreGame contains full game details and team/player statistics.
type BoxScoreGame struct {
	GameID            string     `json:"gameId"`
	GameTimeLocal     string     `json:"gameTimeLocal"`
	GameTimeUTC       string     `json:"gameTimeUTC"`
	GameET            string     `json:"gameEt"`
	Duration          int        `json:"duration"`
	GameCode          string     `json:"gameCode"`
	GameStatus        GameStatus `json:"gameStatus"`
	GameStatusText    string     `json:"gameStatusText"`
	Period            int        `json:"period"`
	GameClock         Duration   `json:"gameClock"`
	RegulationPeriods int        `json:"regulationPeriods"`
	Attendance        int        `json:"attendance"`
	Arena             Arena      `json:"arena"`
	Officials         []Official `json:"officials"`
	HomeTeam          BoxTeam    `json:"homeTeam"`
	AwayTeam          BoxTeam    `json:"awayTeam"`
}

// Arena is the game venue.
type Arena struct {
	ArenaID       int    `json:"arenaId"`
	ArenaName     string `json:"arenaName"`
	ArenaCity     string `json:"arenaCity"`
	ArenaState    string `json:"arenaState"`
	ArenaCountry  string `json:"arenaCountry"`
	ArenaTimezone string `json:"arenaTimezone"`
}

// Official is a game referee.
type Official struct {
	PersonID   int    `json:"personId"`
	Name       string `json:"name"`
	NameI      string `json:"nameI"`
	FirstName  string `json:"firstName"`
	FamilyName string `json:"familyName"`
	JerseyNum  string `json:"jerseyNum"`
	Assignment string `json:"assignment"`
}

// BoxTeam is a team's box score data including roster and statistics.
type BoxTeam struct {
	TeamID            int         `json:"teamId"`
	TeamName          string      `json:"teamName"`
	TeamCity          string      `json:"teamCity"`
	TeamTricode       string      `json:"teamTricode"`
	Score             int         `json:"score"`
	InBonus           *string     `json:"inBonus"`
	TimeoutsRemaining int         `json:"timeoutsRemaining"`
	Periods           []Period    `json:"periods"`
	Players           []BoxPlayer `json:"players"`
	Statistics        TeamStats   `json:"statistics"`
}

// BoxPlayer is a player entry in the box score.
type BoxPlayer struct {
	Status     string      `json:"status"`
	Order      int         `json:"order"`
	PersonID   int         `json:"personId"`
	JerseyNum  string      `json:"jerseyNum"`
	Position   string      `json:"position"`
	Starter    BoolString  `json:"starter"`
	OnCourt    BoolString  `json:"oncourt"`
	Played     BoolString  `json:"played"`
	Name       string      `json:"name"`
	NameI      string      `json:"nameI"`
	FirstName  string      `json:"firstName"`
	FamilyName string      `json:"familyName"`
	Statistics PlayerStats `json:"statistics"`
}

// PlayerStats contains individual player statistics.
type PlayerStats struct {
	Points                  int      `json:"points"`
	Assists                 int      `json:"assists"`
	ReboundsTotal           int      `json:"reboundsTotal"`
	ReboundsDefensive       int      `json:"reboundsDefensive"`
	ReboundsOffensive       int      `json:"reboundsOffensive"`
	Steals                  int      `json:"steals"`
	Blocks                  int      `json:"blocks"`
	BlocksReceived          int      `json:"blocksReceived"`
	Turnovers               int      `json:"turnovers"`
	FieldGoalsMade          int      `json:"fieldGoalsMade"`
	FieldGoalsAttempted     int      `json:"fieldGoalsAttempted"`
	FieldGoalsPercentage    float64  `json:"fieldGoalsPercentage"`
	ThreePointersMade       int      `json:"threePointersMade"`
	ThreePointersAttempted  int      `json:"threePointersAttempted"`
	ThreePointersPercentage float64  `json:"threePointersPercentage"`
	TwoPointersMade         int      `json:"twoPointersMade"`
	TwoPointersAttempted    int      `json:"twoPointersAttempted"`
	TwoPointersPercentage   float64  `json:"twoPointersPercentage"`
	FreeThrowsMade          int      `json:"freeThrowsMade"`
	FreeThrowsAttempted     int      `json:"freeThrowsAttempted"`
	FreeThrowsPercentage    float64  `json:"freeThrowsPercentage"`
	FoulsPersonal           int      `json:"foulsPersonal"`
	FoulsOffensive          int      `json:"foulsOffensive"`
	FoulsTechnical          int      `json:"foulsTechnical"`
	FoulsDrawn              int      `json:"foulsDrawn"`
	PlusMinusPoints         float64  `json:"plusMinusPoints"`
	Plus                    float64  `json:"plus"`
	Minus                   float64  `json:"minus"`
	Minutes                 Duration `json:"minutes"`
	MinutesCalculated       Duration `json:"minutesCalculated"`
	PointsFastBreak         int      `json:"pointsFastBreak"`
	PointsInThePaint        int      `json:"pointsInThePaint"`
	PointsSecondChance      int      `json:"pointsSecondChance"`
}

// TeamStats contains team-level aggregate statistics.
// This intentionally duplicates some PlayerStats fields rather than embedding,
// because the JSON shape is flat and the types serve different purposes.
type TeamStats struct {
	Points                      int      `json:"points"`
	Assists                     int      `json:"assists"`
	ReboundsTotal               int      `json:"reboundsTotal"`
	ReboundsDefensive           int      `json:"reboundsDefensive"`
	ReboundsOffensive           int      `json:"reboundsOffensive"`
	ReboundsTeam                int      `json:"reboundsTeam"`
	Steals                      int      `json:"steals"`
	Blocks                      int      `json:"blocks"`
	BlocksReceived              int      `json:"blocksReceived"`
	Turnovers                   int      `json:"turnovers"`
	TurnoversTotal              int      `json:"turnoversTotal"`
	TurnoversTeam               int      `json:"turnoversTeam"`
	FieldGoalsMade              int      `json:"fieldGoalsMade"`
	FieldGoalsAttempted         int      `json:"fieldGoalsAttempted"`
	FieldGoalsPercentage        float64  `json:"fieldGoalsPercentage"`
	FieldGoalsEffectiveAdjusted float64  `json:"fieldGoalsEffectiveAdjusted"`
	ThreePointersMade           int      `json:"threePointersMade"`
	ThreePointersAttempted      int      `json:"threePointersAttempted"`
	ThreePointersPercentage     float64  `json:"threePointersPercentage"`
	TwoPointersMade             int      `json:"twoPointersMade"`
	TwoPointersAttempted        int      `json:"twoPointersAttempted"`
	TwoPointersPercentage       float64  `json:"twoPointersPercentage"`
	FreeThrowsMade              int      `json:"freeThrowsMade"`
	FreeThrowsAttempted         int      `json:"freeThrowsAttempted"`
	FreeThrowsPercentage        float64  `json:"freeThrowsPercentage"`
	FoulsPersonal               int      `json:"foulsPersonal"`
	FoulsOffensive              int      `json:"foulsOffensive"`
	FoulsTechnical              int      `json:"foulsTechnical"`
	FoulsTeam                   int      `json:"foulsTeam"`
	FoulsTeamTechnical          int      `json:"foulsTeamTechnical"`
	FoulsDrawn                  int      `json:"foulsDrawn"`
	Minutes                     Duration `json:"minutes"`
	MinutesCalculated           Duration `json:"minutesCalculated"`
	PointsFastBreak             int      `json:"pointsFastBreak"`
	PointsInThePaint            int      `json:"pointsInThePaint"`
	PointsSecondChance          int      `json:"pointsSecondChance"`
	PointsFromTurnovers         int      `json:"pointsFromTurnovers"`
	PointsAgainst               int      `json:"pointsAgainst"`
	BenchPoints                 int      `json:"benchPoints"`
	BiggestLead                 int      `json:"biggestLead"`
	BiggestLeadScore            string   `json:"biggestLeadScore"`
	BiggestScoringRun           int      `json:"biggestScoringRun"`
	BiggestScoringRunScore      string   `json:"biggestScoringRunScore"`
	LeadChanges                 int      `json:"leadChanges"`
	TimesTied                   int      `json:"timesTied"`
	TimeLeading                 Duration `json:"timeLeading"`
	TrueShootingPercentage      float64  `json:"trueShootingPercentage"`
	TrueShootingAttempts        float64  `json:"trueShootingAttempts"`
	AssistsTurnoverRatio        float64  `json:"assistsTurnoverRatio"`
}
