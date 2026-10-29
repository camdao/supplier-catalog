## Architecture

![alt text](/docs/image/image.png)


## Features

### Core Guarantees
- **Idempotent**: Import the same batch multiple times without creating duplicates
- **Version-safe**: Older data never overwrites newer versions, even when delivered out of order
- **Withdrawal-protected**: Withdrawn products remain withdrawn unless explicitly restored by a newer version
- **Resumable**: Interrupted updates resume from the last completed batch, no work is lost

### Operational Safety
- **Fault-tolerant**: Supplier outages and application crashes cannot corrupt the existing catalog
- **Auditable**: Complete change history with version tracking for every product modification


## Build & Run

```bash
# Install dependencies and build
go mod tidy
go build -o catalog .

# Or run directly
go run . <command>
```

## Commands

### Start a catalog update

```bash
./catalog update start
```

Fetches all batches from the supplier and processes them. Creates a new update session and processes all product changes.

### Resume an incomplete update

```bash
./catalog update resume
```

Continues a previously interrupted update. Skips already-completed batches and retries failed ones.

### Check update status

```bash
./catalog update status
```

Shows the current or latest update session: status, batch completion, and any errors.

### List products

```bash
# All active products
./catalog products list

# Include withdrawn products
./catalog products list --withdrawn

# Filter by category
./catalog products list --category Electronics
```

### Show product details

```bash
./catalog products show PROD-001
```

### View change history

```bash
# Changes from latest update
./catalog changes list

# Changes from specific update
./catalog changes list --update-id <update-id>
```

Shows what changed: products added, updated, withdrawn, or skipped due to older versions.

## Tests

Run all tests:
```bash
go test ./...
```

Run with verbose output:
```bash
go test -v ./...
```

Run specific package:
```bash
go test -v ./internal/repo
go test -v ./internal/service
```

The test suite includes:
- **Unit tests** (`internal/service/product_test.go`) - Test business logic in isolation (version comparison, withdrawal protection, idempotency)
- **Integration tests** (`internal/service/update_test.go`, `internal/repo/update_test.go`) - Test full update flows with database and mock suppliers


## Assumptions

### Supplier API Behavior
- The supplier provides a manifest of batch IDs for each update
- Batches can be fetched independently and may arrive out of order
- The same batch ID always returns the same data (idempotent fetches)
- Supplier outages are temporary and batches can be retried

### Version Numbering
- Product versions are monotonically increasing positive integers
- A higher version number always represents newer data
- Version numbers may have gaps (e.g., v1 → v3 is valid)

### Data Model
- Product IDs are unique and stable across updates
- A product can be added as withdrawn (new product that's already discontinued)
- Price is stored as cents (int64) to avoid floating-point precision issues

### Update Process
- Only one update can be in progress at a time
- Batches within an update can be processed independently
- Completed batches are idempotent (safe to process multiple times)
- A partial update can be resumed without losing progress

### Database Choice
- SQLite chosen for simplicity and portability (no separate database server needed)
- Sufficient for the assessment scope (not production-scale requirements)
- Single-writer concurrency model fits the single in-progress update constraint


## Unfinished Work

### Performance Optimizations
- Batch size tuning not yet optimized for large catalogs
- No connection pooling (single SQLite connection)
- Change records could be indexed by update_id for faster queries

### Error Handling
- Limited retry logic for transient supplier errors
- No exponential backoff for failed batch fetches
- Database transaction timeouts not tunable

### Edge Cases
- Network partition during update not fully tested
- Very large batches (>10,000 products) not performance-tested
- Concurrent resume attempts not prevented (assumes single operator)

### Monitoring & Observability
- No metrics collection (update duration, batch processing time)
- No structured logging
- No tracing for debugging update flows

### Deployment
- No containerization (Docker/Kubernetes)
- No health check endpoints
- No graceful shutdown handling
- Configuration is minimal (only database path)

### Features Not Implemented
- API endpoints (currently CLI only)
- Web UI for viewing products/updates
- Rollback mechanism for bad updates
- Update scheduling/automation
- Multi-supplier support


## AI Usage

This project was developed with assistance from **Claude Code (Claude Fable 5)**:

### What AI helped with:
- **Test development**: AI wrote comprehensive unit tests for business logic (idempotency, version ordering, withdrawal protection) and integration tests for update flows
- **Architecture refactoring**: AI assisted in separating business logic from data access layer, moving version comparison and decision logic from repository to service layer
- **Code organization**: AI helped reorganize files by feature (product.go, update.go, change.go) for better maintainability
- **Bug fixes**: AI identified and fixed a logic bug in the resume flow where skipped batches incorrectly marked updates as incomplete

### Development approach:
- All code was reviewed and understood by the developer before being committed
- AI suggestions were evaluated for correctness, security, and alignment with requirements
- The developer made architectural decisions (SQLite, CLI interface, fake supplier design)
- AI acted as a coding assistant, not an autonomous agent
