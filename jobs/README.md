# Jobs Module Testing

## Requirements

### Local Development
- **Docker must be running** - Tests will fail with a clear error if Docker is not available
- Tests automatically start a Redis container for each test run
- No fallbacks or mocks - real Redis only

### GitHub Actions
- Tests automatically use the Redis service provided by GitHub Actions
- No additional setup required - it just works

## Running Tests

```bash
# Run all jobs tests (Docker must be running)
cd jobs
go test -v

# Run specific test scenarios
go test -v -run TestJobsFeatures
```

## Test Coverage

Current coverage: ~38% of runtime.go

To view coverage report:
```bash
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Architecture

The jobs module uses [Asynq](https://github.com/hibiken/asynq) for background job processing with Redis as the backend.

### Test Infrastructure

- `test_helpers.go` - Manages Redis container lifecycle
- `runtime_test.go` - BDD tests using Cucumber/Godog
- `redis_test.go` - Simple connectivity test

### Key Features Tested

- Runtime initialization with/without Redis
- Job enqueueing and processing
- Welcome email jobs
- Session cleanup jobs
- Error handling and retries
- Worker management

## Docker Compose

For local development with persistent Redis:

```bash
docker-compose up -d
```

This starts Redis on port 6379 with data persistence.

## Troubleshooting

### "Docker must be running" error
Start Docker Desktop or the Docker daemon:
```bash
# macOS
open -a Docker

# Linux
sudo systemctl start docker
```

### "Redis container failed to start" error
Check if port 6379 is already in use:
```bash
lsof -i :6379
```

Kill any existing Redis processes or containers:
```bash
docker ps -q --filter ancestor=redis | xargs docker stop
```

## CI/CD

GitHub Actions workflow automatically:
1. Starts Redis service on port 6379
2. Runs all tests including jobs tests
3. Reports coverage

No special configuration needed - tests detect GHA environment and use the provided Redis service.