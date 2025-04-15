package soccer

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_fixture(t *testing.T) {
	var text = `
	{
		"match_id": "394177",
		"country_id": "8",
		"country_name": "Worldcup",
		"league_id": "22",
		"league_name": "AFC World Cup Qualifiers - 3rd Round",
		"match_date": "2025-03-20",
		"match_status": "",
		"match_time": "10:10",
		"match_hometeam_id": "529",
		"match_hometeam_name": "Australia",
		"match_hometeam_score": "",
		"match_awayteam_name": "Indonesia",
		"match_awayteam_id": "484",
		"match_awayteam_score": "",
		"match_hometeam_halftime_score": "",
		"match_awayteam_halftime_score": "",
		"match_hometeam_extra_score": "",
		"match_awayteam_extra_score": "",
		"match_hometeam_penalty_score": "",
		"match_awayteam_penalty_score": "",
		"match_hometeam_ft_score": "",
		"match_awayteam_ft_score": "",
		"match_hometeam_system": "",
		"match_awayteam_system": "",
		"match_live": "0",
		"match_round": "7",
		"match_stadium": "Allianz Stadium (Sydney)",
		"match_referee": "",
		"team_home_badge": ""
	}
`
	fx := Fixture{}
	if err := json.Unmarshal([]byte(text), &fx); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "394177", fx.ID)
	assert.Equal(t, "8", fx.CountryID)
	assert.Equal(t, "Worldcup", fx.CountryName)

}
