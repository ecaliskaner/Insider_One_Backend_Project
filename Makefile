.PHONY: build run test docker-run swagger clean

# Build the application
build:
	go build -o league-simulation.exe .

# Run the application locally
run: build
	./league-simulation.exe

# Run tests with verbose output
test:
	go test ./... -v

# Generate Swagger documentation
swagger:
	swag init

# Run Docker container using docker-compose
docker-run:
	docker-compose up --build -d

# Clean build files
clean:
	rm -f league-simulation.exe
	rm -f league.db*
