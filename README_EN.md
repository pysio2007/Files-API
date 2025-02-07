<div align="center">

# Files-API
![Files-API](https://socialify.git.ci/pysio2007/Files-API/image?custom_description=自动同步Github+Repo到Minio&description=1&font=Inter&forks=1&language=1&name=1&owner=1&pattern=Signal&pulls=1&stargazers=1&theme=Auto)

[![GitHub issues](https://img.shields.io/github/issues/pysio2007/Files-API)](https://github.com/pysio2007/Files-API/issues)
[![GitHub license](https://img.shields.io/github/license/pysio2007/Files-API)](https://github.com/pysio2007/Files-API/blob/main/LICENSE)
[![GitHub stars](https://img.shields.io/github/stars/pysio2007/Files-API)](https://github.com/pysio2007/Files-API/stargazers)
[![Go Report Card](https://goreportcard.com/badge/github.com/pysio2007/Files-API)](https://goreportcard.com/report/github.com/pysio2007/Files-API)

🚀 High-performance Git repository synchronization and distribution service with automatic Minio object storage sync support.

[简体中文](./README.md) | English

</div>

## 📚 Table of Contents

- [✨ Features](#-features)
- [🚀 Quick Start](#-quick-start)
- [📝 Configuration](#-configuration)
- [🛠️ API Endpoints](#️-api-endpoints)
- [📈 Performance](#-performance)
- [🔧 Debugging](#-debugging)
- [🤝 Contributing](#-contributing)
- [📄 License](#-license)

## ✨ Features

- 🔄 Multi Git repository auto-sync
- ⏱️ Configurable sync intervals per repository
- 🚀 Customizable concurrent upload threads
- 📝 Incremental updates
- 🔗 Unified file access API
- 🎯 Custom access path mapping
- 🔒 SHA1 checksum verification
- 💫 Async sync without blocking
- 📦 Local cache for faster access

## 💾 Requirements

- Go 1.16+
- Minio Server (or S3 compatible storage)
- Git

## 🎯 Core Functions

1. Git Repository Sync
   - Multi-repo auto sync
   - Incremental updates
   - Custom sync intervals
   - SHA1 checksum verification

2. Minio Object Storage
   - S3 compatible support
   - Presigned URL access
   - Parallel upload optimization
   - Auto bucket creation

3. Caching System
   - Local file cache
   - CDN cache control
   - API response cache
   - Auto cleanup mechanism

4. Log Management
   - Auto log rotation
   - Multi-level logging
   - Log compression
   - Space management

## 🚀 Quick Start

### Requirements

- Go 1.16+
- Minio Server (or S3 compatible storage)
- Git

### Installation

```bash
# Clone repository
git clone https://github.com/pysio2007/Files-API.git
cd Files-API

# Install dependencies
go mod download

# Run service
go run main.go
```

A default `config.yaml` will be generated on first run.

## 📝 Configuration

### Main Configuration

```yaml
server:
    port: 8080          # Server port
    host: "0.0.0.0"     # Listen address
    enableAPI: true     # Enable API service
    apiOnly: false      # API-only mode
    legacyAPI: true     # Enable legacy API support

minio:
    endpoint: "play.min.io"
    accessKey: "your-access-key"
    secretKey: "your-secret-key"
    useSSL: true
    bucket: "documents"
    usePublicURL: true   # Use presigned URLs
    maxWorkers: 16       # Concurrent upload threads

cache:
    enabled: true
    directory: ".cache/files"
    maxSize: 1000        # Cache size limit (MB)
    ttl: "7d"            # Cache TTL
    cacheControl: "30d"  # CDN cache time
```

### Multi-Bucket Configuration

The service supports configuring multiple storage buckets to meet different data storage requirements. Each bucket configuration includes:
- Name: Identifier used for routing.
- Endpoint: Storage server address.
- AccessKey & SecretKey: Credentials for authentication.
- UseSSL: Whether to use SSL.
- BucketName: Actual bucket name.
- BasePath: Base path for files in the bucket (can be empty).
- ReadOnly: Indicates if the bucket is read-only.

Example configuration:
```yaml
buckets:
  - name: "Images"            # Routing identifier; accessed via /Images/
    endpoint: "minioapi.example.com"
    accessKey: "your-access-key"
    secretKey: "your-secret-key"
    useSSL: true
    bucketName: "pysioimages"   # Actual bucket name
    basePath: ""                # Root directory
    readOnly: true
```

### Service Modes

1. Full Mode (Default)
   ```yaml
   enableAPI: true
   apiOnly: false
   ```
   - API endpoints available
   - Direct file access enabled
   - Suitable for most scenarios

2. API-Only Mode
   ```yaml
   enableAPI: true
   apiOnly: true
   ```
   - API endpoints only
   - Direct file access disabled
   - For strict access control

### Legacy API Support

The service supports automatic redirection from legacy API paths to the new format:

1. Enable Support:
```yaml
server:
    legacyAPI: true     # Enable legacy API support
```

2. Path Mapping Rules:
- Legacy: `/files/Pysio-Images/example.png`
- New: `/Pysio-Images/example.png`

3. Redirection Details:
- Uses 301 permanent redirect
- Automatically removes `/files/` prefix
- Preserves original query parameters
- Logs redirections (if enabled)

4. Logging:
```yaml
logs:
    redirectLog: true   # Enable redirection logging
```

### Repository Configuration
```yaml
git:
  cachePath: ".cache/repos"      # Local cache directory
  repositories:
    - url: "https://github.com/user/repo1"   # Repository URL
      branch: "main"                         # Branch name
      localPath: "repos/repo1"              # Local cache path
      minioPath: "static"                   # Storage path prefix
      checkInterval: "1h"                   # Sync check interval (m/h/d/y)

exposedPaths:
    - urlPath: "/assets"        # Access URL path
      minioPath: "static"       # Storage path prefix
```

### Logging Configuration
```yaml
logs:
    accessLog: true     # Record all file requests
    processLog: false   # Record processing details
    redirectLog: false  # Record URL redirections
    presignLog: false   # Record presigned URL generation
    saveToFile: true    # Save logs to file
    maxSize: 100        # Max log directory size (MB)
    directory: "logs"   # Log directory
```

### Cache Configuration
```yaml
cache:
    enabled: true               # Enable file caching
    directory: ".cache/files"   # Cache directory
    maxSize: 1000              # Cache size limit (MB)
    ttl: "7d"                  # Cache TTL
    cacheControl: "30d"        # Static file CDN cache time
    enableAPICache: true       # Enable API cache control
    apiCacheControl: "5m"      # API response cache time
    cacheLog: true             # Log cache operations
    hitLog: true               # Log cache hits
```

### Advanced Cache Configuration

1. API Cache Control
```yaml
cache:
    enableAPICache: true       # Enable API caching
    apiCacheControl: "5m"      # API cache duration
    apiExcludePaths:          # Paths to exclude from caching
        - "/api/files/sync/status"  # Sync status endpoint
```

2. Cache Exclusion Rules
- Supports exact path matching
- Supports path prefix matching
- Takes effect even when enableAPICache is true
- Higher priority than global cache settings

3. Performance Recommendations
- Add dynamic content APIs to exclusion list
- Use longer cache times for static content
- Disable caching for monitoring endpoints

### Cache Mechanism

1. Local Cache
   - Cache file content and metadata on disk
   - Auto cleanup of expired cache files
   - Configurable cache directory size limit
   - Configurable cache TTL

2. Separate Control
   - Different cache times for API and static files
   - Optional API response caching
   - Long cache time for static files (30 days)
   - Short cache time for API responses (5 minutes)

3. CDN Support
   - Control CDN caching via Cache-Control headers
   - Configure cache times by resource type
   - Compatible with various CDN services

4. Cache Monitoring
   - Optional cache operation logging
   - Optional cache hit logging
   - Record cleanup and expiration events
   - Monitor cache space usage

### Time Interval Format

Supported time interval formats:
- `s`: seconds, e.g., `"60s"` for 60 seconds  
- `m`: minutes, e.g., `"10m"` for 10 minutes
- `h`: hours, e.g., `"1h"` for 1 hour
- `d`: days, e.g., `"1d"` for 1 day
- `y`: years, e.g., `"1y"` for 1 year

External URL supports high-frequency checks:

Default is 10 minutes if not configured or invalid.

### External URL Configuration

The service supports automatic synchronization and caching of external URL resources:

```yaml
externalURLs:
    - path: "/external/banner.jpg"      # Access path
      mainURL: "https://example.com/banner.jpg"  # Primary download URL
      backupURLs:                       # List of backup URLs
        - "https://backup1.com/banner.jpg"
        - "https://backup2.com/banner.jpg"
      minioPath: "external/banner.jpg"  # Minio storage path
      cacheControl: "max-age=3600"      # Cache control header (e.g., "no-cache" or "max-age=3600")
      checkInterval: "1h"               # Update check interval (e.g., "1h", "1d")

    - path: "/external/logo.png"
      mainURL: "https://example.com/logo.png"
      backupURLs: 
        - "https://cdn.example.com/logo.png"
      minioPath: "external/logo.png"
      cacheControl: "no-cache"          # Disable caching
      checkInterval: "1d"               # Check once per day
```

Configuration Details:
- path: API access path for the external resource
- mainURL & backupURLs: Primary and backup download URLs; in case the main URL fails, backups are attempted in order.
- minioPath: The target storage path in Minio.
- cacheControl: Sets the HTTP cache header; enable detailed logging during troubleshooting.
- checkInterval: Frequency to check for file updates. Use an appropriate value based on the importance of the resource.

How it works:
1. On the first access, the resource is downloaded and stored in Minio.
2. Later, the system checks for updates based on the checkInterval and automatically retries with backup URLs on failure.
3. Users can monitor logs and error messages to debug issues, ensuring proper Minio write permissions and network connectivity.

Troubleshooting Tips:
- If the resource does not update, verify that the external URLs are reachable and the network is reliable.
- To bypass browser caching during testing, use: 
  curl -H "Cache-Control: no-cache" http://localhost:8080/external/banner.jpg

## Special Launch Parameters

### Skip Initial Sync (--skip)

When started with `--skip`, the program will:
1. Skip initial sync at startup
2. Wait for the configured check interval before first sync
3. Useful for delayed sync scenarios

Example:
```bash
# Normal start (with initial sync)
./Files-API

# Skip initial sync
./Files-API --skip
```

Use cases:
- Avoid duplicate syncs in CI/CD
- When repository content is temporarily unavailable
- Waiting for external services
- Control sync timing

## Command Line Options

### Basic Commands
```bash
# Show help
./Files-API -h
./Files-API --help

# Start service (with initial sync)
./Files-API
```

### Sync Control
```bash
# Skip initial sync
./Files-API --skip

# Single sync and exit
./Files-API --sync

# Sync specific repository
./Files-API --rsync=static
```

### Log Management
```bash
# Compress log files
./Files-API --zip-logs

# Decompress log files
./Files-API --unzip-logs

# Clear logs
./Files-API --clear-logs
./Files-API -cl
```

### Cache Management
```bash
# Clear cache
./Files-API --clear-cache
./Files-API -cc

# Clear all logs and cache
./Files-API --clear-all
```

Command details:

1. Sync Control
   - `--skip`: Skip initial sync
   - `--sync`: Single sync and exit
   - `--rsync`: Sync specific repository

2. Log Management
   - `--zip-logs`: Compress logs to zip
   - `--unzip-logs`: Extract log archives
   - `--clear-logs, -cl`: Clear all logs

3. Cache Management
   - `--clear-cache, -cc`: Clear cache directory
   - `--clear-all`: Clear all logs and cache

## File Access

### URL Format
- `GET /{minioPath}/{filePath}`

### Examples
```bash
# Access static resources
GET /static/images/logo.png
GET /assets/css/main.css

# Access other files
GET /public/files/document.pdf
```

## 🛠️ API Reference

### File List API

Get a list of files and subdirectories in the specified directory.

```http
GET /api/files/{path}?page=1&pageSize=20
```

Parameters:
- `path`: Optional, directory path
- `page`: Optional, page number (default: 1)
- `pageSize`: Optional, items per page (default: 20, max: 100)

Response Format:
```json
{
    "code": 200,
    "message": "success",
    "data": [
        {
            "name": "images",
            "path": "static/images/",
            "isDirectory": true
        },
        {
            "name": "logo.png",
            "path": "static/logo.png",
            "size": 12345,
            "lastModified": "2024-02-05T12:34:56Z",
            "isDirectory": false,
            "url": "https://..."
        }
    ],
    "pagination": {
        "current": 1,
        "pageSize": 20,
        "total": 42
    }
}
```

Response Fields:
1. File Information (FileInfo)
   - `name`: File or directory name
   - `path`: Complete path
   - `size`: File size in bytes
   - `lastModified`: Last modification time
   - `isDirectory`: Whether it's a directory
   - `url`: File access URL (only when usePublicURL=true)

2. Pagination Info
   - `current`: Current page number
   - `pageSize`: Items per page
   - `total`: Total number of items

### File Access API

Direct file content access.

```http
GET /{minioPath}/{filePath}
```

Access Modes:
1. Redirect Mode (usePublicURL=true)
   - Returns 302 redirect to presigned URL
   - Presigned URL valid for 1 hour
   - Reduces server load
   - Recommended for public access

2. Proxy Mode (usePublicURL=false)
   - Returns file content directly
   - Sets appropriate Content-Type
   - Supports large file transfers
   - Suitable for internal networks

Examples:
```bash
# Access file
GET /static/images/logo.png

# Get JSON file info with Accept header
curl -H "Accept: application/json" http://localhost:8080/api/files/static/images/

# Paginated query
curl http://localhost:8080/api/files/static/?page=2&pageSize=50
```

### Sync Status API

Get synchronization status for all repositories.

```http
GET /api/files/sync/status
```

Response Format:
```json
{
    "code": 200,
    "message": "success",
    "data": {
        "repo1": {
            "lastSync": "2024-02-05T12:34:56Z",  // Last sync time
            "nextSync": "2024-02-05T13:34:56Z",  // Next scheduled sync
            "progress": 100,                      // Sync progress (0-100)
            "totalFiles": 50,                     // Total files
            "currentFiles": 50,                   // Processed files
            "status": "idle"                      // Status (idle/syncing/error)
        }
    }
}
```

Status Descriptions:
- `idle`: Waiting for next sync
- `syncing`: Currently synchronizing
- `error`: Sync failed, check error field for details
- `unknown`: Initial state

Monitoring Metrics:
- `lastSync`: Last synchronization time
- `nextSync`: Next scheduled sync time
- `progress`: Current sync progress (0-100)
- `totalFiles`: Total number of files
- `currentFiles`: Processed files count
- `error`: Error message (if any)


Monitoring Examples:
```bash
# Check sync status
curl http://localhost:8080/api/files/sync/status

# Monitor progress with watch
watch -n 1 'curl -s http://localhost:8080/api/files/sync/status | jq'
```

### API Cache Control

The service supports different caching strategies for API responses and static files:

```yaml
cache:
    enableAPICache: true    # Enable API response caching
    apiCacheControl: "5m"   # API cache duration (5 minutes)
    cacheControl: "30d"     # Static files cache duration (30 days)
```

Headers:
- API responses include `Cache-Control: public, max-age=300` (5 minutes)
- Static files include `Cache-Control: public, max-age=2592000` (30 days)

### Error Responses

```json
{
    "code": 404,
    "message": "File not found"
}
```

Common Status Codes:
- `200`: Success
- `400`: Invalid request
- `404`: File not found
- `500`: Server error

## 🔧 Debugging Guide

### Log Configuration

1. Full Debug Configuration (Record Everything)
```yaml
logs:
    accessLog: true     # Record all requests
    processLog: true    # Record file processing
    redirectLog: true   # Record URL redirects
    presignLog: true    # Record presigned URL generation
    saveToFile: true    # Output to both file and console
    maxSize: 100        # Log directory limit (MB)
    directory: "logs"   # Log directory
```

2. Cache Debug Configuration
```yaml
cache:
    cacheLog: true      # Record cache operations
    hitLog: true        # Record cache hits
```

### Troubleshooting

1. File Sync Issues
```bash
# Check sync status
./Files-API --sync

# Sync specific repository
./Files-API --rsync=static

# Monitor sync logs
tail -f logs/Files-API-$(date +%Y-%m-%d).log | grep "sync"
```

2. Cache Issues
```bash
# Check cache status
ls -lh .cache/files/

# Clear cache and retry
./Files-API --clear-cache

# Monitor cache hits
tail -f logs/Files-API-$(date +%Y-%m-%d).log | grep "Cache hit"
```

3. Minio Connection Issues
```bash
# Check Minio connection
curl -I http://{minio-endpoint}/
# Or use tools like s3cmd test

# Verify configuration
cat config.yaml | grep minio -A 8
```

### Performance Analysis

1. Using Go Profiling Tools
```bash
# Enable profiling
GODEBUG=gctrace=1 ./Files-API

# Use pprof
go tool pprof http://localhost:8080/debug/pprof/heap
```

2. Monitor System Resources
```bash
# Check memory usage
ps -o pid,ppid,%mem,rss,cmd -p $(pgrep Files-API)

# Check file descriptors
lsof -p $(pgrep Files-API)
```

### Development Debugging

1. Enable All Logs
```bash
# Edit config file
vim config.yaml
# Set all log options to true

# Test with shorter sync interval
checkInterval: "1m"
```

2. API Testing
```bash
# Test file list API
curl "http://localhost:8080/api/files/static/?page=1&pageSize=10"

# Test file access
curl -I "http://localhost:8080/static/test.txt"
```

3. Performance Testing
```bash
# Test concurrent requests
ab -n 1000 -c 10 http://localhost:8080/api/files/static/

# Test file uploads
for i in {1..10}; do
    ./Files-API --sync
done
```

### Error Codes

| Status Code | Description | Resolution |
|-------------|-------------|------------|
| 400 | Bad Request | Check API parameters |
| 404 | File Not Found | Check path and sync status |
| 500 | Internal Server Error | Check detailed logs |
| 503 | Minio Service Unavailable | Check Minio connection |

## Debugging Guide

### Log Configuration

1. Full Debug Configuration
```yaml
logs:
    accessLog: true     # Record all access
    processLog: true    # Record processing
    redirectLog: true   # Record redirects
    presignLog: true    # Record temp links
    saveToFile: true    # Save to file
    maxSize: 100        # Size limit
    directory: "logs"   # Storage dir
```

2. Minimal Log Configuration
```yaml
logs:
    accessLog: true     # Basic access only
    processLog: false
    redirectLog: false
    presignLog: false
    saveToFile: false   # Console only
```

3. View Logs
```bash
# View today's log
cat logs/Files-API-2025-02-05.log

# Monitor real-time logs
tail -f logs/Files-API-2025-02-05.log
```

## Contributing

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

This project is licensed under the [AGPL-3.0](./LICENSE) License.

<div align="center">

### Like this project? Please give it a ⭐️

</div>
