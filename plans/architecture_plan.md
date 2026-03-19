# Architecture Plan

## 1. Architectural Components and Interactions

The system consists of two main parts:
- A Go backend that uses BadgerDB v4 for persistent storage and provides an API for the frontend.
- A WebAssembly frontend that runs in the browser and communicates with the Go backend to display real-time IoT metrics.

The communication between the frontend and backend is via [to be specified] (REST/WebSocket/gRPC).

The backend will handle data ingestion from IoT devices (simulated or real) and store it in BadgerDB.

The frontend will periodically fetch updates from the backend and update the charts and tables.

## 2. Build and Deployment Workflow for Go Backend

The Go backend will be built using Go 1.26 with BadgerDB v4.5.1 as the persistent store. The backend will simulate IoT device data generation.

### Project Structure
```
/backend
  ├── go.mod
  ├── go.sum
  ├── main.go
  ├── api/
  │   ├── handlers.go
  │   └── routes.go
  ├── storage/
  │   └── badger_store.go
  ├── models/
  │   └── metric.go
  ├── simulator/
  │   └── device_simulator.go
  └── (no Dockerfile required)
```

### Dependencies Management
- Use Go modules (`go.mod`) to manage dependencies
- Key dependencies:
  - github.com/dgraph-io/badger/v4 v4.5.1
  - google.golang.org/grpc (for gRPC communication)
  - Any other required libraries

### Build Process
1. Initialize module: `go mod init backend`
2. Add dependencies: 
   - `go get github.com/dgraph-io/badger/v4@v4.5.1`
   - `go get google.golang.org/grpc`
3. Build: `go build -o backend ./main.go`
4. Cross-compilation (if needed): `GOOS=linux GOARCH=amd64 go build -o backend ./main.go`

### Deployment Options
1. **Direct Binary**: Copy the built binary to a server and run it (no Docker required)
2. **Cloud Deployment**: Deploy to platforms like AWS, GCP, or Azure using bare metal or VMs

### Running the Backend
- Development: `go run ./main.go`
- Production: `./backend`

## 3. Build and Deployment Workflow for WASM Frontend

The WebAssembly frontend will be built using either standard Go WASM compilation or TinyGo, depending on the requirements and compatibility with the chosen frontend framework.

### Technology Options
1. **Standard Go WASM**: Using `GOOS=js GOARCH=wasm` build flags
2. **TinyGo**: Optimized for WebAssembly with smaller binary sizes
3. **Alternative**: Using AssemblyScript or Rust with WASM if Go proves unsuitable

### Project Structure
```
/frontend
  ├── go.mod (if using Go)
  ├── main.go
  ├── wasm_exec.js (provided by Go toolchain)
  ├── index.html
  ├── styles.css
  ├── assets/
  │   └── (images, icons, etc.)
  └── components/
      ├── chart.js (or Go equivalent)
      └── table.js (or Go equivalent)
```

### Dependencies Management
If using Go for frontend:
- Use Go modules (`go.mod`)
- Key dependencies:
  - github.com/maxence-charriere/go-app/v9 (for Progressive Web App)
  - github.com/gorilla/websocket (if using WebSocket)
  - Any UI/component libraries compatible with Go WASM

### Build Process
#### Option A: Standard Go WASM
1. Initialize module: `go mod init frontend`
2. Add dependencies: `go get github.com/maxence-charriere/go-app/v9`
3. Build: `GOOS=js GOARCH=wasm go build -o frontend.wasm ./main.go`
4. Copy wasm_exec.js: `cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" ./`

#### Option B: TinyGo
1. Install TinyGo
2. Build: `tinygo build -o frontend.wasm -target wasm ./main.go`

### Development Workflow
1. Use a simple HTTP server to serve the WASM files (required due to WASM fetch restrictions)
2. Development server: `go run github.com/rakyll/statik` or `python -m http.server`
3. Hot reload can be achieved with tools like `air` or custom file watchers

### Deployment Options
1. **Static Hosting**: Deploy the WASM file, HTML, CSS, and wasm_exec.js to any static hosting service (Netlify, Vercel, GitHub Pages, S3, etc.)
2. **Embedded in Go Backend**: Serve frontend files directly from the Go backend using `http.FileSystem` or similar
3. **Containerized**: Package with nginx or similar for serving static assets

### Running the Frontend
- Development: Serve via local HTTP server and open in browser
- Production: Deploy static assets to CDN or web server

## 4. Communication Mechanisms

For real-time IoT metrics dashboard, we will use gRPC for all communication between the WASM frontend and Go backend.

### Primary Communication: gRPC
- **Technology**: 
  - Backend: google.golang.org/grpc
  - Frontend: grpc-web (via WASM JS interop) or direct gRPC-WASM if feasible
- **Purpose**: All communication including real-time metric updates, initial data fetch, configuration, and historical data queries
- **Why**: 
  - Strongly typed contracts via Protocol Buffers
  - Efficient binary serialization
  - Built-in support for streaming (perfect for real-time updates)
  - Single communication protocol simplifies implementation
- **Implementation**:
  - Backend: gRPC server with streaming capabilities for real-time metrics
  - Frontend: WASM code that connects to gRPC server and handles streaming responses
  - Message format: Protocol Buffers (protobuf)

### gRPC Services Definition
We'll define the following services in our `.proto` file:

```protobuf
service MetricsService {
  // Stream of real-time metric updates
  rpc SubscribeMetrics(SubscriptionRequest) returns (stream MetricUpdate);
  
  // Fetch historical metrics
  rpc GetHistoricalMetrics(HistoricalRequest) returns (stream MetricPoint);
  
  // Get available devices
  rpc GetDevices(DeviceRequest) returns (DeviceList);
  
  // Update configuration
  rpc UpdateConfig(ConfigUpdate) returns (ConfigResponse);
}
```

### Data Serialization
- Use Protocol Buffers for all gRPC communication
- Define `.proto` files for:
  - MetricUpdate (timestamp, deviceID, metricName, value, etc.)
  - SubscriptionRequest (device filters, metric types)
  - HistoricalRequest (time ranges, filters)
  - DeviceList, ConfigUpdate, etc.

### Why Not WebSocket/REST?
gRPC provides better performance, stronger typing, and built-in streaming capabilities compared to WebSocket/REST hybrid. While gRPC-WASM has some complexity, it's manageable and provides a cleaner architecture for this use case.

## 5. Data Synchronization and Caching Strategy

### Data Flow
1. **Ingestion**: IoT devices send metrics to the Go backend via HTTP POST or MQTT bridge
2. **Persistence**: Backend writes metrics to BadgerDB with appropriate indexing (device ID, timestamp, metric type)
3. **Notification**: Backend publishes updates to WebSocket subscribers
4. **Frontend Update**: WASM frontend receives WebSocket messages and updates UI components

### Caching Strategy
To reduce BadgerDB load and improve response times:

#### In-Memory Cache (Backend)
- Use a lightweight cache like `github.com/patrickmn/go-cache` or sync.Map for:
  - Recent metrics (last N points per device) for quick WebSocket broadcasting
  - Frequently accessed device metadata
  - Aggregated statistics (min/max/avg over recent windows)
- Cache TTL: Configure based on data freshness requirements (e.g., 30 seconds to 5 minutes)
- Cache invalidation: Update on new data arrival; periodic cleanup of expired entries

#### Frontend Caching
- WASM frontend can cache:
  - Recent chart data to reduce redraw frequency
  - Device list and configuration
  - Use browser's localStorage for persistence across sessions (user preferences)

### Synchronization Mechanisms
1. **Write Path**: 
   - IoT data → Backend → BadgerDB (persistent) + In-memory cache → WebSocket broadcast
   
2. **Read Path**:
   - Frontend requests → Check frontend cache → If miss/stale, request backend → Backend checks in-memory cache → If miss/stale, query BadgerDB → Update caches

3. **Consistency**:
   - Strong consistency not required for metrics dashboard; eventual consistency is acceptable
   - Use BadgerDB's ACID transactions for writes
   - Cache updates happen after successful DB write

### BadgerDB Specific Considerations
- Use appropriate indexing: 
  - Primary key: deviceID + timestamp + metricName
  - Separate indexes for time-range queries per device
- Value logging: Enable for larger metric payloads if needed
- Compression: Snappy compression (default) is suitable for time-series data
- GC tuning: Adjust garbage collection based on write volume and retention policy

### Data Retention and Archiving
- Implement retention policy (e.g., keep raw data for 30 days, aggregated data for 1 year)
- Background job to downsample old data and delete raw points
- BadgerDB's TTL feature (via ExpiresAt) can automate expiration

## 6. Dependencies and Tooling

### Go Backend Dependencies
- **Go Toolchain**: Version 1.26 (as specified)
- **BadgerDB**: github.com/dgraph-io/badger/v4 v4.5.1
- **gRPC**: google.golang.org/grpc
- **Protocol Buffers**: google.golang.org/protobuf
- **Caching** (optional): github.com/patrickmn/go-cache
- **Configuration**: github.com/spf13/viper or similar
- **Logging**: github.com/sirupsen/logrus or zap
- **Testing**: 
  - stretchr/testify
  - Go's built-in testing package

### WASM Frontend Dependencies
#### If using Standard Go WASM:
- **Go Toolchain**: Version 1.26 (with WASM support)
- **gRPC**: 
  - For direct gRPC-WASM: Use grpc-go with WASM build tags
  - For grpc-web: Use JavaScript grpc-web client via syscall/js
- **Protocol Buffers**: google.golang.org/protobuf (for generating Go structs from .proto files)
- **UI Framework** (optional):
  - github.com/maxence-charriere/go-app/v9 (for PWA-like experience)
  - github.com/AllenDang/wui (immediate mode GUI)
  - Standard library + custom DOM manipulation via syscall/js
- **JSON**: encoding/json (standard library, for any JSON interop needs)
- **UI Charting**: 
  - Consider using JavaScript charting libraries via WASM JS interop (Chart.js, Plotly, etc.)
  - Or Go-based charting if available and performant

#### If using TinyGo:
- **TinyGo Compiler**: Latest stable version
- **TinyGo WASM target**: Built-in support
- **gRPC**: TinyGo has experimental gRPC support; may need to use grpc-web via JS interop
- **Protocol Buffers**: Use protobuf compiler to generate TinyGo-compatible code
- **UI Libraries**: 
  - github.com/tinygo-org/tinygo/tree/master/examples/wasm
  - Consider using JavaScript interop for complex UI components

### Development and Build Tooling
- **Version Control**: Git
- **IDE**: VS Code or GoLand with Go plugins
- **Package Management**: Go modules (built-in to Go 1.26)
- **Build Automation**:
  - Makefile or Justfile for common tasks
  - Air or CompileDaemon for hot reloading during development
  - Docker for containerization
- **Testing**:
  - Go's built-in testing framework
  - Browser-based testing for frontend (Playwright, Cypress, or simple manual testing)
- **Documentation**:
  - GoDoc for backend API documentation
  - JSDoc or similar for frontend JS interop code

### Deployment Tooling
- **Containerization**: Docker
- **Orchestration** (optional): Kubernetes or Docker Compose for local testing
- **CI/CD**: GitHub Actions, GitLab CI, or similar
- **Monitoring**: 
  - Prometheus + Grafana for metrics
  - ELK stack or similar for logging
- **Static Site Hosting** (for frontend): Netlify, Vercel, GitHub Pages, AWS S3+CloudFront, etc.

## 7. Performance and Compatibility Considerations

### Performance Considerations

#### Backend Performance
- **BadgerDB Optimization**:
  - Use batch writes for high-throughput ingestion
  - Tune memtable size and LSM tree parameters based on write patterns
  - Separate directories for value log if using value logging
  - Monitor and adjust garbage collection frequency
- **WebSocket Scaling**:
  - Use connection pooling and efficient message broadcasting
  - Consider using a message broker (Redis Pub/Sub, NATS) for multiple backend instances
  - Implement connection limits and heartbeat mechanisms
- **CPU and Memory**:
  - Profile CPU usage during peak loads
  - Monitor memory usage of caches and BadgerDB
  - Consider using GOGC environment variable to tune garbage collection

#### Frontend Performance
- **WASM Binary Size**:
  - TinyGo typically produces smaller WASM binaries than standard Go
  - Strip debug information and optimize for size in production builds
  - Consider code splitting if using large UI libraries
- **Rendering Performance**:
  - Use requestAnimationFrame for smooth chart updates
  - Implement data sampling for historical views (show fewer points when zoomed out)
  - Virtualize large tables if displaying many rows
- **Network Efficiency**:
  - Compress WebSocket messages if needed (per-message deflate extension)
  - Use binary encoding (like protobuf) for WebSocket if JSON proves insufficient
  - Cache static assets aggressively with proper HTTP headers

#### Overall System Performance
- **Latency Goals**:
  - WebSocket message delivery: <100ms for 95th percentile
  - Initial dashboard load: <3 seconds
  - Chart update latency: <50ms after receiving new data
- **Throughput Targets**:
  - Handle 10K+ metric points per second ingestion (adjust based on actual requirements)
  - Support 100+ concurrent dashboard users
- **Monitoring and Profiling**:
  - Add Prometheus metrics to backend (request latency, error rates, DB performance)
  - Use Chrome DevTools to profile WASM frontend performance
  - Log slow queries and cache miss rates

### Compatibility Considerations

#### Go Version Compatibility
- **Go 1.26 Specifics**:
  - Ensure all dependencies are compatible with Go 1.26 modules
  - Test WASM compilation with `GOOS=js GOARCH=wasm` (available since Go 1.11)
  - Note any deprecated APIs that changed between Go 1.26 and newer versions

#### BadgerDB v4 Compatibility
- **API Stability**:
  - BadgerDB v4 has a stable API; ensure we're using versioned imports
  - Check for any breaking changes from v3 if migrating
- **Platform Support**:
  - BadgerDB supports Linux, macOS, Windows (our development and deployment platforms)
  - Ensure proper file permissions and storage backend compatibility

#### WASM Browser Compatibility
- **Browser Support**:
  - All modern browsers support WebAssembly (Chrome, Firefox, Safari, Edge)
  - Test across target browser versions
  - Consider fallback for very old browsers if required (unlikely for internal dashboard)
- **WASM Execution Environment**:
  - Memory growth: Monitor and handle WASM memory limits
  - Threading: Standard Go WASM doesn't support threads; use web workers if needed via JS interop
  - SIMD: Not available in standard Go WASM; consider if needed for heavy computations

#### Security Considerations
- **Communication Security**:
  - Use WSS (WebSocket Secure) in production, not plain WS
  - Implement proper CORS policies
  - Validate and sanitize all incoming data from IoT devices
- **Data Security**:
  - Consider encrypting sensitive data in BadgerDB if required
  - Implement authentication and authorization for dashboard access
  - Regular security audits of dependencies
- **Dependency Security**:
  - Use tools like `govulncheck` to scan for vulnerabilities
  - Keep dependencies updated within compatibility constraints

#### Deployment Compatibility
- **Container Compatibility**:
  - Ensure Docker images work on target deployment platforms
  - Test multi-arch builds if needed (amd64, arm64)
- **Resource Requirements**:
  - Define minimum RAM/CPU for backend based on expected load
  - WASM frontend typically requires <50MB RAM in browser
- **Network Requirements**:
  - WebSocket requires persistent connections; ensure proxy/load balancer supports it
  - Consider sticky sessions if using multiple backend instances without message broker

### Testing and Validation
- **Load Testing**:
  - Simulate IoT device influx with tools like vegeta or k6
  - Test WebSocket connection scaling
- **Compatibility Matrix**:
  - Test backend on: Ubuntu LTS, CentOS Stream, Windows Server 2022
  - Test frontend on: Chrome latest, Firefox latest, Safari latest, Edge latest
- **Chaos Engineering**:
  - Test network partitions and recovery
  - Validate cache consistency after restarts
  - BadgerDB recovery testing after power loss simulation

By addressing these performance and compatibility considerations, we can ensure the dashboard application runs smoothly under the specified Go 1.26 and BadgerDB v4 constraints while providing a responsive user experience.