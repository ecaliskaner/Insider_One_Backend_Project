.PHONY: build run test bench vet verify docker-run swagger clean

# Build the application
build:
	go build -o league-simulation.exe .

# Run the application locally
run: build
	./league-simulation.exe migrate up
	./league-simulation.exe seed
	./league-simulation.exe serve

# Run tests with verbose output
test:
	go test ./... -v

# Run Oracle benchmarks
bench:
	go test ./services -bench=BenchmarkLeagueService_GetPredictions -benchmem -run '^$$'

# Run go vet
vet:
	go vet ./...

# Run the core local verification suite
verify:
	gofmt -l .
	go vet ./...
	go test ./...
	go build ./...

# Generate Swagger documentation
swagger:
	go run github.com/swaggo/swag/cmd/swag init

# Run Docker container using docker-compose
docker-run:
	docker-compose up --build -d

# Clean build files
clean:
	rm -f league-simulation.exe
	rm -f league.db*
