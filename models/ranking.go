package models

import "sort"

// RankStandings applies the same league ranking rules everywhere standings are ordered.
func RankStandings(standings []Standing, matches []Match) []Standing {
	if len(standings) == 0 {
		return standings
	}

	sort.Slice(standings, func(i, j int) bool {
		a, b := standings[i], standings[j]
		if a.Points != b.Points {
			return a.Points > b.Points
		}
		if a.GD != b.GD {
			return a.GD > b.GD
		}
		if a.GF != b.GF {
			return a.GF > b.GF
		}
		return a.TeamName < b.TeamName
	})

	groups := groupPerfectTies(standings)
	var ranked []Standing
	position := 1
	for _, group := range groups {
		if len(group) > 1 {
			sortTiedStandings(group, matches)
		}
		for _, standing := range group {
			standing.Position = position
			position++
			ranked = append(ranked, *standing)
		}
	}

	return ranked
}

func groupPerfectTies(standings []Standing) [][]*Standing {
	var groups [][]*Standing
	var current []*Standing

	for i := range standings {
		if len(current) == 0 {
			current = append(current, &standings[i])
			continue
		}

		first := current[0]
		if standings[i].Points == first.Points && standings[i].GD == first.GD && standings[i].GF == first.GF {
			current = append(current, &standings[i])
			continue
		}

		groups = append(groups, current)
		current = []*Standing{&standings[i]}
	}

	if len(current) > 0 {
		groups = append(groups, current)
	}
	return groups
}

func sortTiedStandings(group []*Standing, matches []Match) {
	tiedIDs := make(map[int]bool, len(group))
	for _, standing := range group {
		tiedIDs[standing.TeamID] = true
	}

	type headToHeadStats struct {
		points    int
		awayGoals int
	}
	stats := make(map[int]*headToHeadStats, len(group))
	for id := range tiedIDs {
		stats[id] = &headToHeadStats{}
	}

	for _, match := range matches {
		if !tiedIDs[match.HomeTeamID] || !tiedIDs[match.AwayTeamID] || match.HomeScore == nil || match.AwayScore == nil {
			continue
		}

		homeGoals, awayGoals := *match.HomeScore, *match.AwayScore
		if homeGoals > awayGoals {
			stats[match.HomeTeamID].points += 3
		} else if homeGoals < awayGoals {
			stats[match.AwayTeamID].points += 3
		} else {
			stats[match.HomeTeamID].points++
			stats[match.AwayTeamID].points++
		}
		stats[match.AwayTeamID].awayGoals += awayGoals
	}

	sort.Slice(group, func(i, j int) bool {
		left := stats[group[i].TeamID]
		right := stats[group[j].TeamID]
		if left.points != right.points {
			return left.points > right.points
		}
		if left.awayGoals != right.awayGoals {
			return left.awayGoals > right.awayGoals
		}
		return group[i].TeamName < group[j].TeamName
	})
}
