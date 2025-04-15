package soccer

import (
	"strconv"
	"time"
)

// return available nearest time fixture +- 1 day
func NearestFixtures(events []Fixture) []Fixture {
	result := []Fixture{}
	if len(events) == 0 {
		return result
	}

	earliest := events[0]
	now := time.Now()
	date := earliest.Date.Time()
	min := date.Sub(now)

	result = append(result, earliest)
	for _, f := range events {

		date := f.Date.Time()
		dur := date.Sub(now)

		if dur-min <= 24*time.Hour {
			if min >= dur {
				earliest = f
				min = dur
			}
			result = append(result, f)
		}
	}

	return result
}

type Fixture struct {
	ID          string `json:"match_id"`
	CountryID   string `json:"country_id"`
	CountryName string `json:"country_name"`

	League_ID  string `json:"league_id"`
	LeagueName string `json:"league_name"`
	Stadium    string `json:"match_stadium"`

	HomeTeamID string `json:"match_hometeam_id"`
	HomeTeam   string `json:"match_hometeam_name"`
	HomeScore  string `jason:"match_hometeam_score"`

	AwayTeamID string `json:"match_awayteam_id"`
	AwayTeam   string `json:"match_awayteam_name"`
	AwayScore  string `jason:"match_awayteam_score"`

	LeagueYear string `json:"league_year"`
	StageName  string `json:"stage_name"`

	Status string  `json:"match_status"` // on going match minute or finished
	Date   DateVar `json:"match_date"`
	Start  string  `json:"match_time"`

	IsLive string `json:"match_live"`

	Scorer []Scorer `json:"goalscorer"`
}

func (f *Fixture) GetScore() string {
	if len(f.Scorer) != 0 {
		return f.Scorer[len(f.Scorer)-1].Score
	}
	return "0 - 0"
}

type Scorer struct {
	Score string `json:"score"`
}

type TZ string

type IntID int

func (id *IntID) String() string {
	return strconv.Itoa(int(*id))
}

type BoolVar bool

func (b *BoolVar) String() string {
	if *b {
		return "true"
	}
	return "false"
}

type DateVar string

func (d *DateVar) Time() time.Time {
	t, err := time.Parse(time.DateOnly, string(*d))
	if err != nil {
		panic(err)
	}
	return t
}

type EventParam struct {
	Timezone  TZ
	CountryID IntID
	LeagueID  IntID
	MacthID   IntID
	TeamID    IntID
	IsLive    BoolVar
}
