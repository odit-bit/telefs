package wncli

import (
	"math/rand/v2"
	"time"
)

var (
	TN_ID countryID = "id"
)

type countryID string

type Article struct {
	ID          int64
	Title       string
	Text        string
	Summary     string
	URL         string
	Video       string
	PublishDate time.Time
	Author      string
	Authors     []string
}

type News struct {
	Articles []Article `json:"news"`
}

func (n *News) RandomArticle() Article {
	if len(n.Articles) == 0 {
		return Article{}
	}

	num := rand.IntN(len(n.Articles)-0) - 0
	return n.Articles[num]
}

type TopNewsRequest struct {
	CountryID countryID
}

type TopNewsResp struct {
	TopNews []News `json:"top_news"`
}
