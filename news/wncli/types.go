package wncli

import (
	"time"
)

var (
	TN_ID countryID = "id"
)

type countryID string

type Article struct {
	ID          int64    `json:"id"`
	Title       string   `json:"title"`
	Text        string   `json:"text"`
	Summary     string   `json:"summary"`
	URL         string   `json:"url"`
	Image       string   `json:"image"`
	Video       string   `json:"video"`
	PublishDate string   `json:"publish_date"`
	Author      string   `json:"author"`
	Authors     []string `json:"authors"`
}

type News struct {
	Articles []Article `json:"news"`
}

type TopNewsRequest struct {
	CountryID countryID
}

type TopNewsResp struct {
	TopNews  []News `json:"top_news"`
	Language string `json:"language"`
	Country  string `json:"country"`
	CachedAt time.Time
}
