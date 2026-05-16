package services

import (
	"log"
)

// StartListeners initializes background workers to process events
func StartListeners(eb *EventBus, s *LeagueServiceImpl) {
	matchFinishedCh := eb.Subscribe("match_finished")

	go func() {
		for event := range matchFinishedCh {
			if payload, ok := event.(MatchFinishedEvent); ok {
				// 1. Update Morale and Fatigue
				s.updateTeamMetrics(&payload.HomeTeam, &payload.AwayTeam, payload.HomeGoals, payload.AwayGoals)
				
				// 2. We don't strictly *need* to update standings per match instantly here if we recalculate them
				// at the end of the week, but doing so fits the Event-Driven approach.
				// For the sake of the existing logic, we recalculate standings in PlayNextWeek.
				// However, if we wanted to fully decouple, we'd trigger a recalculation event here.
				log.Printf("Listener: Processed match %d (%s %d - %d %s)", 
					payload.MatchID, payload.HomeTeam.Name, payload.HomeGoals, payload.AwayGoals, payload.AwayTeam.Name)
			}
		}
	}()
}
