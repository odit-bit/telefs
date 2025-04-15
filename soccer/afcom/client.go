package afcom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/odit-bit/tdrive/soccer"

	"github.com/gorilla/websocket"
)

var mapLeagueID = map[soccer.IntID]string{
	soccer.ID_Champions_League: "3",
}

// client manage apifootball.com api call
type Client struct {
	baseURL string
	key     string
	cli     http.Client
}

func New(endpoint, key string) *Client {
	c := Client{
		baseURL: endpoint,
		key:     key,
		cli:     http.Client{Timeout: 5 * time.Second},
	}
	return &c
}

func newEventRequest(ctx context.Context, uri string, apiKey string, from, to time.Time, param soccer.EventParam) (*http.Request, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	val := url.Values{}
	val.Set("action", "get_events")
	val.Set("APIkey", apiKey)
	val.Set("from", from.Format(time.DateOnly))
	val.Set("to", to.Format(time.DateOnly))

	// optional param
	val.Set("timezone", string(param.Timezone))
	val.Set("country_id", param.CountryID.String())
	val.Set("league_id", mapLeagueID[param.LeagueID])
	val.Set("match_id", param.MacthID.String())
	val.Set("team_id", param.TeamID.String())
	val.Set("match_live", param.MacthID.String())

	parsed.RawQuery = val.Encode()
	return http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)

}

func (api *Client) Events(ctx context.Context, from, to time.Time, param soccer.EventParam) ([]soccer.Fixture, error) {

	req, err := newEventRequest(ctx, api.baseURL, api.key, from, to, param)
	if err != nil {
		return nil, fmt.Errorf("apiFootballCom failed create request: %v", err)
	}

	res, err := api.cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("apiFootballCom get request err: %v", err)
	}

	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("apiFootballCom get response err: %v", res.Status)
	}

	//parse the payload
	defer res.Body.Close()

	payload := []soccer.Fixture{}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("apiFootballCom failed parse response payload: %v", err)
	}
	return payload, nil
}

func (api *Client) WSLivescore(ctx context.Context, param soccer.EventParam) (<-chan []soccer.Fixture, <-chan error) {
	URL := url.URL{Scheme: "wss", Host: "apifootball.com", Path: "livescore"}

	val := url.Values{}
	val.Set("country_id", param.CountryID.String())
	val.Set("league_id", mapLeagueID[param.LeagueID])
	val.Set("match_id", param.MacthID.String())

	errC := make(chan error, 1)
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, URL.String(), http.Header(val))
	if err != nil {
		errC <- err
		close(errC)
		return nil, errC
	}

	resC := make(chan []soccer.Fixture, 1)
	go func() {
		defer func() {
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			conn.Close()
		}()

		var err error
	readLoop:
		for {
			list := []soccer.Fixture{}
			err = conn.ReadJSON(&list)
			if err != nil {
				break readLoop
			}

			select {
			case resC <- list:
			case <-ctx.Done():
				err = ctx.Err()
				break readLoop
			}
		}

		errC <- err
		close(errC)
		close(resC)
	}()

	return resC, errC

}
