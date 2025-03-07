package wncli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	urlpkg "net/url"
)

const (
	_wnBaseUrl = "https://api.worldnewsapi.com"
	_wnTopNews = "top-news"
)

// wrap the world news API client call
type Client struct {
	apiKey string
}

func New(apiKey string) *Client {
	return &Client{apiKey: apiKey}
}

func httpTopNewsRequest(ctx context.Context, apiKey string, id string) (*http.Request, error) {

	q := urlpkg.Values{}
	q.Set("source-country", id)
	q.Set("language", id)
	q.Set("headlines-only", "true")
	q.Set("api-key", apiKey)

	u, err := urlpkg.Parse(fmt.Sprintf("%s/%s", _wnBaseUrl, _wnTopNews))
	if err != nil {
		return nil, err
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Connection", "keep-alive")

	return req, nil
}

func (c *Client) TopNews(pCtx context.Context, cid countryID) (*TopNewsResp, error) {
	//  "https://api.worldnewsapi.com/top-news?source-country=id&language=id&date=U&headlines-only=%3Cboolean%3E"
	ctx, cancel := context.WithCancel(pCtx)
	defer cancel()

	req, err := httpTopNewsRequest(ctx, c.apiKey, string(cid))
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("got error: %d", res.StatusCode)
	}
	tn := TopNewsResp{}

	if err := json.NewDecoder(res.Body).Decode(&tn); err != nil {
		return nil, err
	}

	return &tn, nil
}
