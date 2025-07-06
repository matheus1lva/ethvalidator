# Ethereum Validator API

A high-performance RESTful API for querying Ethereum validator information, including block rewards and sync committee duties.

## Features

- **Block Rewards**: Query block rewards by slot with MEV detection
- **Sync Committee Duties**: Get validator sync committee assignments
- **High Performance**: Built-in caching, connection pooling, and concurrent request handling
- **Production Ready**: Comprehensive logging, metrics, health checks, and graceful shutdown
- **Clean Architecture**: Following Go best practices with clear separation of concerns
- **Observability**: Prometheus metrics and structured logging with request tracing

## Architecture

The project follows clean architecture principles:

```
├── cmd/api/              # Application entrypoint
├── internal/             # Private application code
│   ├── api/             # HTTP layer
│   │   ├── handlers/    # Request handlers
│   │   └── middleware/  # HTTP middleware
│   ├── config/          # Configuration management
│   ├── domain/          # Business entities
│   └── service/         # Business logic
├── pkg/                 # Public packages
│   ├── cache/          # Caching implementation
│   ├── errors/         # Error definitions
│   ├── ethereum/       # Ethereum client
│   └── logger/         # Structured logging
└── test/               # Integration tests
```

## Requirements

- Go 1.21 or higher
- Docker (optional, for containerized deployment)
- Access to Ethereum RPC endpoint (provided in assignment)

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/matheus/eth-validator-api.git
cd eth-validator-api

# Install dependencies
go mod download

# Build the application
go build -o api ./cmd/api

# Run the application
./api
```

### Using Docker

```bash
# Build the Docker image
docker build -t eth-validator-api .

# Run with Docker
docker run -p 8080:8080 --env-file .env eth-validator-api

# Or use Docker Compose
docker-compose up
```

## Configuration

The application uses environment variables for configuration. Copy `.env.example` to `.env` and update as needed:

```bash
cp .env.example .env
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8080` |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` |
| `ETH_RPC_ENDPOINT` | Ethereum RPC endpoint URL | Required |
| `REQUEST_TIMEOUT` | HTTP request timeout | `30s` |
| `CACHE_TTL` | Cache time-to-live | `5m` |
| `CACHE_MAX_SIZE` | Maximum cache entries | `1000` |
| `MAX_CONCURRENT_REQUESTS` | Max concurrent RPC requests | `10` |
| `METRICS_ENABLED` | Enable Prometheus metrics | `true` |

## API Endpoints

### Get Block Reward

Retrieves block reward information for a given slot.

```bash
GET /blockreward/{slot}
```

**Parameters:**
- `slot` (integer): The slot number in the Ethereum blockchain

**Response:**
```json
{
  "data": {
    "status": "mev",
    "reward": "1000000000000000000"
  }
}
```

**Status Codes:**
- `200 OK`: Success
- `400 Bad Request`: Invalid slot or future slot
- `404 Not Found`: Slot not found/missed
- `500 Internal Server Error`: Server error

**Example:**
```bash
curl http://localhost:8080/blockreward/7890123
```

### Get Sync Committee Duties

Retrieves validators with sync committee duties for a given slot.

```bash
GET /syncduties/{slot}
```

**Parameters:**
- `slot` (integer): The slot number in the Ethereum blockchain

**Response:**
```json
{
  "data": {
    "validators": [
      "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a",
      "0x8831234f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a"
    ]
  }
}
```

**Status Codes:**
- `200 OK`: Success
- `400 Bad Request`: Invalid slot or slot too far in future
- `404 Not Found`: Slot not found
- `500 Internal Server Error`: Server error

**Example:**
```bash
curl http://localhost:8080/syncduties/7890123
```

### Health Check

```bash
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": "2h30m15s",
  "timestamp": "2024-01-15T10:30:00Z",
  "system": {
    "go_version": "go1.21.5",
    "num_goroutine": 10,
    "num_cpu": 8
  }
}
```

### Metrics

```bash
GET /metrics
```

Prometheus-formatted metrics including:
- HTTP request duration
- Request counts by endpoint and status
- Go runtime metrics

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with race detector
go test -race ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Linting

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linters
golangci-lint run
```

### Building

```bash
# Build for current platform
go build -o api ./cmd/api

# Build with version information
go build -ldflags="-X main.version=1.0.0 -X main.commit=$(git rev-parse HEAD) -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o api ./cmd/api

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -o api-linux ./cmd/api
```

## Monitoring

The application includes built-in monitoring capabilities:

### Prometheus Metrics

When `METRICS_ENABLED=true`, metrics are exposed at `/metrics`:

- `http_requests_total`: Total HTTP requests by path, method, and status
- `http_duration_seconds`: HTTP request duration histogram
- Standard Go runtime metrics

### Structured Logging

All logs include:
- Request ID for tracing
- Structured fields for easy parsing
- Configurable log levels

Example log output:
```json
{
  "level": "info",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "GET",
  "path": "/blockreward/7890123",
  "status": 200,
  "duration": "125ms",
  "time": "2024-01-15T10:30:00Z",
  "message": "request completed"
}
```

## Performance Considerations

1. **Caching**: Responses are cached to reduce RPC calls
2. **Connection Pooling**: HTTP client reuses connections
3. **Concurrent Requests**: Configurable concurrency limits
4. **Timeouts**: Request timeouts prevent hanging
5. **Graceful Shutdown**: Proper cleanup on termination

## Security

- No sensitive data in logs
- Request validation and sanitization
- Proper error handling without information leakage
- Non-root Docker container
- Health check endpoint separate from metrics

## Deployment

### Docker Compose

The included `docker-compose.yml` provides a complete stack with:
- API service
- Prometheus for metrics collection
- Grafana for visualization

```bash
docker-compose up -d
```

Access:
- API: http://localhost:8080
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin/admin)

### Kubernetes

Example deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: eth-validator-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: eth-validator-api
  template:
    metadata:
      labels:
        app: eth-validator-api
    spec:
      containers:
      - name: api
        image: eth-validator-api:latest
        ports:
        - containerPort: 8080
        env:
        - name: ETH_RPC_ENDPOINT
          valueFrom:
            secretKeyRef:
              name: eth-config
              key: rpc-endpoint
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

## Troubleshooting

### Common Issues

1. **Connection Refused**
   - Check if the service is running: `curl http://localhost:8080/health`
   - Verify the PORT environment variable

2. **RPC Errors**
   - Verify ETH_RPC_ENDPOINT is set correctly
   - Check network connectivity to RPC endpoint
   - Increase REQUEST_TIMEOUT if needed

3. **High Memory Usage**
   - Reduce CACHE_MAX_SIZE
   - Lower MAX_CONCURRENT_REQUESTS

### Debug Mode

Enable debug logging:
```bash
LOG_LEVEL=debug ./api
```

## License

MIT License - see LICENSE file for details

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Contact

For questions about this assignment, please contact Thomas from the Tech Team.