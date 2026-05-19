package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRankStandings_AppliesHeadToHeadForPerfectTies(t *testing.T) {
	homeGoals, awayGoals := 1, 0
	standings := []Standing{
		{TeamID: 1, TeamName: "Beta", Played: 3, Won: 1, Lost: 1, Drawn: 1, GF: 2, GA: 2, GD: 0, Points: 4},
		{TeamID: 2, TeamName: "Alpha", Played: 3, Won: 1, Lost: 1, Drawn: 1, GF: 2, GA: 2, GD: 0, Points: 4},
	}
	matches := []Match{
		{
			HomeTeamID: 2,
			AwayTeamID: 1,
			HomeScore:  &homeGoals,
			AwayScore:  &awayGoals,
			Status:     "played",
		},
	}

	ranked := RankStandings(standings, matches)

	assert.Equal(t, 2, ranked[0].TeamID)
	assert.Equal(t, 1, ranked[0].Position)
	assert.Equal(t, 1, ranked[1].TeamID)
	assert.Equal(t, 2, ranked[1].Position)
}
