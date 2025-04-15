package soccer

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	ID_Champions_League IntID = iota
)

var (
	local_zone TZ = ""
)

func init() {
	loc, err := time.LoadLocation("")
	if err != nil {
		panic(err)
	}
	zone, _ := time.Now().In(loc).Zone()
	local_zone = TZ(zone)
}

type LiveScoreVendor interface {
	WSLivescore(ctx context.Context, param EventParam) (<-chan []Fixture, <-chan error)
}

type LiveScore struct {
	HomeTeam string
	AwayTeam string
	Score    string
	Minutes  string
}

type ChampionsLeague struct {
	vendor LiveScoreVendor
	logger *logrus.Logger

	*publisher[LiveScore]
}

func NewLiveChampionLeague(api LiveScoreVendor) *ChampionsLeague {
	pub := NewPublisher[LiveScore]()
	return &ChampionsLeague{
		publisher: pub,
		vendor:    api,
		logger:    logrus.New(),
	}
}

// // perpetual polling until error or context is done
// func (cl *ChampionsLeague) PollContext(ctx context.Context) error {

// 	// sync into next minute
// 	now := time.Now()
// 	nextMinute := now.Truncate(time.Minute).Add(time.Minute)
// 	delay := nextMinute.Sub(now)
// 	time.Sleep(delay)

// 	ticker := time.NewTicker(1 * time.Minute)
// 	for {
// 		select {
// 		case <-ctx.Done():
// 			ticker.Stop()
// 			cl.logger.Infof("publisher exit poll: %v \n", ctx.Err())
// 			return ctx.Err()
// 		case <-ticker.C:
// 			now := time.Now().Local()
// 			update, err := cl.vendor.Events(ctx, now, now, EventParam{
// 				Timezone: local_zone,
// 				LeagueID: ID_Champions_League,
// 				IsLive:   true,
// 			})
// 			if err != nil {
// 				return err
// 			}
// 			fx := update
// 			for _, f := range fx {
// 				if err := cl.Publish(ctx, LiveScore{
// 					AwayTeam: f.AwayTeam,
// 					HomeTeam: f.HomeTeam,
// 					Score:    f.GetScore(),
// 					Minutes:  f.Status,
// 				}); err != nil {
// 					cl.logger.Errorf("publisher failed publish: %v \n", err)
// 				}
// 			}

// 		}
// 	}
// }

// perpetual polling until error or context is done
func (cl *ChampionsLeague) PollContext(ctx context.Context) error {

	// // sync into next minute
	// now := time.Now()
	// nextMinute := now.Truncate(time.Minute).Add(time.Minute)
	// delay := nextMinute.Sub(now)
	// time.Sleep(delay)

	// ticker := time.NewTicker(1 * time.Minute)

	resC, errC := cl.vendor.WSLivescore(ctx, EventParam{
		Timezone: local_zone,
		LeagueID: ID_Champions_League,
		IsLive:   true,
	})

	for {
		select {
		case <-ctx.Done():
			cl.logger.Infof("publisher exit poll: %v \n", ctx.Err())
			return ctx.Err()
		case err := <-errC:
			return err
		case update := <-resC:
			fx := update
			for _, f := range fx {
				if err := cl.Publish(ctx, LiveScore{
					AwayTeam: f.AwayTeam,
					HomeTeam: f.HomeTeam,
					Score:    f.GetScore(),
					Minutes:  f.Status,
				}); err != nil {
					cl.logger.Errorf("publisher failed publish: %v \n", err)
				}
			}

		}
	}
}
