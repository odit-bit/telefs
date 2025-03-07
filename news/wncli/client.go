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

var defaultHeader = func() http.Header {
	header := http.Header{}
	header.Set("Accept", "application/json")
	header.Set("Connection", "keep-alive")
	return header
}

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
	// endpoint.Query().Add("date", time.Now().Local().)
	q.Set("headlines-only", "true")
	q.Set("api-key", apiKey)

	u, err := urlpkg.Parse(fmt.Sprintf("%s/%s", _wnBaseUrl, _wnTopNews))
	if err != nil {
		return nil, err
	}
	u.RawQuery = q.Encode()

	// req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	// if err != nil {
	// 	return nil, err
	// }
	// return req, nil

	req := &http.Request{
		// ctx:        ctx,
		Method:     http.MethodGet,
		URL:        u,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     defaultHeader(),
		Body:       nil,
		Host:       u.Host,
	}

	req = req.WithContext(ctx)
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

	if res.StatusCode > 300 {
		return nil, fmt.Errorf("got error: %d", res.StatusCode)
	}
	tn := TopNewsResp{}

	if err := json.NewDecoder(res.Body).Decode(&tn); err != nil {
		return nil, err
	}

	return &tn, nil
}
