package services

import (
	"context"
	"testing"

	"github.com/insider/league-simulation/database"
)

func BenchmarkLeagueService_GetPredictions(b *testing.B) {
	ctx := context.Background()
	service := newBenchmarkedLeagueService(b, ctx)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		service.invalidateCache()
		predictions, err := service.GetPredictions(ctx)
		if err != nil {
			b.Fatalf("get predictions: %v", err)
		}
		if len(predictions) != 4 {
			b.Fatalf("expected 4 predictions, got %d", len(predictions))
		}
	}
}

func BenchmarkLeagueService_GetPredictionsCached(b *testing.B) {
	ctx := context.Background()
	service := newBenchmarkedLeagueService(b, ctx)

	if _, err := service.GetPredictions(ctx); err != nil {
		b.Fatalf("warm cache: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		predictions, err := service.GetPredictions(ctx)
		if err != nil {
			b.Fatalf("get predictions: %v", err)
		}
		if len(predictions) != 4 {
			b.Fatalf("expected 4 predictions, got %d", len(predictions))
		}
	}
}

func newBenchmarkedLeagueService(b *testing.B, ctx context.Context) *LeagueServiceImpl {
	b.Helper()

	db, err := database.NewDB(":memory:")
	if err != nil {
		b.Fatalf("new db: %v", err)
	}
	b.Cleanup(func() {
		if err := db.Close(); err != nil {
			b.Fatalf("close db: %v", err)
		}
	})

	if err := db.RunMigrations(); err != nil {
		b.Fatalf("run migrations: %v", err)
	}
	if err := database.SeedTeams(db); err != nil {
		b.Fatalf("seed teams: %v", err)
	}
	if err := database.SeedPlayers(db); err != nil {
		b.Fatalf("seed players: %v", err)
	}
	if err := database.SeedStandings(db); err != nil {
		b.Fatalf("seed standings: %v", err)
	}
	if err := database.GenerateSchedule(db); err != nil {
		b.Fatalf("generate schedule: %v", err)
	}

	service := NewLeagueService(db, NewMatchEngineWithSeed(42), NewWeatherAdapterWithSeed(43))
	for week := 0; week < 4; week++ {
		if _, err := service.PlayNextWeek(ctx); err != nil {
			b.Fatalf("play week %d: %v", week+1, err)
		}
	}

	return service
}
