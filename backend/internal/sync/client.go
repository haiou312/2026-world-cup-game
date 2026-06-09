package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const fdBase = "https://api.football-data.org/v4"

type Client struct {
	token string
	http  *http.Client
}

func NewClient(token string) *Client {
	return &Client{token: token, http: &http.Client{Timeout: 20 * time.Second}}
}

type fdTeam struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Tla   string `json:"tla"`
	Crest string `json:"crest"`
}

type fdMatch struct {
	ID       int       `json:"id"`
	UtcDate  time.Time `json:"utcDate"`
	Status   string    `json:"status"`
	Stage    string    `json:"stage"`
	Group    string    `json:"group"`
	HomeTeam fdTeam    `json:"homeTeam"`
	AwayTeam fdTeam    `json:"awayTeam"`
	Score    struct {
		Winner   string `json:"winner"`
		Duration string `json:"duration"`
		FullTime struct {
			Home *int `json:"home"`
			Away *int `json:"away"`
		} `json:"fullTime"`
	} `json:"score"`
}

type matchesResponse struct {
	Matches []fdMatch `json:"matches"`
}

type fdStandingRow struct {
	Position       int    `json:"position"`
	Team           fdTeam `json:"team"`
	PlayedGames    int    `json:"playedGames"`
	Won            int    `json:"won"`
	Draw           int    `json:"draw"`
	Lost           int    `json:"lost"`
	Points         int    `json:"points"`
	GoalsFor       int    `json:"goalsFor"`
	GoalsAgainst   int    `json:"goalsAgainst"`
	GoalDifference int    `json:"goalDifference"`
}

type fdStanding struct {
	Stage string          `json:"stage"`
	Type  string          `json:"type"`
	Group string          `json:"group"`
	Table []fdStandingRow `json:"table"`
}

type standingsResponse struct {
	Standings []fdStanding `json:"standings"`
}

// Matches fetches a competition's matches. If date (YYYY-MM-DD) is non-empty it
// filters to that single day; otherwise the whole competition is returned.
func (c *Client) Matches(ctx context.Context, competition, date string) ([]fdMatch, error) {
	q := url.Values{}
	if date != "" {
		q.Set("dateFrom", date)
		q.Set("dateTo", date)
	}
	var out matchesResponse
	if err := c.get(ctx, "/competitions/"+competition+"/matches", q, &out); err != nil {
		return nil, err
	}
	return out.Matches, nil
}

// Standings fetches the competition's group tables (already sorted by the
// official tie-breakers via each row's position).
func (c *Client) Standings(ctx context.Context, competition string) ([]fdStanding, error) {
	var out standingsResponse
	if err := c.get(ctx, "/competitions/"+competition+"/standings", url.Values{}, &out); err != nil {
		return nil, err
	}
	return out.Standings, nil
}

func (c *Client) get(ctx context.Context, path string, q url.Values, out any) error {
	u := fdBase + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Auth-Token", c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("football-data status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
