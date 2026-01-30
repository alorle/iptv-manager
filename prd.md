# PRD: M3U Playlist Proxy Endpoints with Caching

## Overview
Add two M3U playlist proxy endpoints to the existing Go server that fetch playlists from IPFS sources, rewrite acestream:// URLs to Acestream player URLs, and provide persistent caching with automatic expiration and refresh. The endpoints will serve cached versions when IPFS sources are unreachable, ensuring reliability.

## Goals
- Provide transparent proxy access to IPFS-hosted M3U playlists
- Rewrite acestream:// URLs to player-compatible format using configurable base URL
- Implement persistent file-based caching with configurable TTL
- Automatically refresh expired cache entries on request
- Serve stale cache when upstream IPFS sources are unavailable
- Maintain all original playlist content except stream URLs

## Quality Gates

These commands must pass for every user story:
- `go test ./...` - Run all tests
- `go vet ./...` - Static analysis
- `go build` - Ensure code compiles

For stories involving HTTP endpoints or caching:
- Include integration tests against test server or mocked IPFS responses

## User Stories

### US-001: Add configuration for cache and player URL
**Description:** As a developer, I want to configure the cache directory, TTL, and Acestream player base URL via environment variables so that the deployment is flexible.

**Acceptance Criteria:**
- [ ] Add `CACHE_DIR` environment variable with validation
- [ ] Add `CACHE_TTL` environment variable (duration format like "1h", "30m")
- [ ] Add `ACESTREAM_PLAYER_BASE_URL` environment variable
- [ ] Update `.env.example` with new variables
- [ ] Add configuration validation on server startup
- [ ] Return clear error if required env vars are missing

### US-002: Implement persistent cache storage layer
**Description:** As a system, I want to store fetched playlists to disk with metadata so that they can be served when IPFS is unavailable.

**Acceptance Criteria:**
- [ ] Create cache storage interface with Get/Set/IsExpired methods
- [ ] Implement file-based cache storage in configured CACHE_DIR
- [ ] Store both playlist content and fetch timestamp
- [ ] Cache key should be derived from source URL
- [ ] Handle file system errors gracefully
- [ ] Ensure cache directory is created if it doesn't exist

### US-003: Implement M3U fetching with cache fallback
**Description:** As a system, I want to fetch M3U content from IPFS with cache fallback so that the service remains available during upstream failures.

**Acceptance Criteria:**
- [ ] Create HTTP client with reasonable timeout (e.g., 10s)
- [ ] Fetch content from IPFS URL
- [ ] On success, update cache with fresh content
- [ ] On failure, check for cached version and serve if available
- [ ] Return 502 Bad Gateway only if no cache exists
- [ ] Log fetch attempts and cache hits/misses

### US-004: Implement acestream:// URL rewriting
**Description:** As a system, I want to rewrite acestream:// URLs to player-compatible format so that clients can play the streams.

**Acceptance Criteria:**
- [ ] Parse M3U content line by line
- [ ] Identify lines starting with "acestream://"
- [ ] Extract stream ID from acestream:// URL
- [ ] Rewrite to `${ACESTREAM_PLAYER_BASE_URL}/ace/getstream?id={id}`
- [ ] Preserve all other lines unchanged (including #EXTINF metadata)
- [ ] Handle edge cases (malformed URLs, empty lines)

### US-005: Implement cache expiration and refresh logic
**Description:** As a system, I want to check cache expiration on each request and refresh if needed so that users get reasonably fresh content.

**Acceptance Criteria:**
- [ ] On request, check if cached content exists and is fresh (within TTL)
- [ ] If cache is fresh, serve immediately
- [ ] If cache is expired, attempt to fetch new content
- [ ] If fetch succeeds, update cache and serve new content
- [ ] If fetch fails, serve stale cache with warning log
- [ ] TTL calculation based on file modification time + CACHE_TTL

### US-006: Add /playlists/elcano.m3u endpoint
**Description:** As a user, I want to access the Elcano playlist at /playlists/elcano.m3u so that I can use it in my media player.

**Acceptance Criteria:**
- [ ] Register GET handler at `/playlists/elcano.m3u`
- [ ] Source URL: `https://ipfs.io/ipns/k51qzi5uqu5di462t7j4vu4akwfhvtjhy88qbupktvoacqfqe9uforjvhyi4wr/hashes_acestream.m3u`
- [ ] Use fetch + cache + rewrite pipeline
- [ ] Return Content-Type: `audio/x-mpegurl` or `application/vnd.apple.mpegurl`
- [ ] Return 200 OK with playlist content
- [ ] Return 502 Bad Gateway if no cache and fetch fails

### US-007: Add /playlists/newera.m3u endpoint
**Description:** As a user, I want to access the NewEra playlist at /playlists/newera.m3u so that I can use it in my media player.

**Acceptance Criteria:**
- [ ] Register GET handler at `/playlists/newera.m3u`
- [ ] Source URL: `https://ipfs.io/ipns/k2k4r8oqlcjxsritt5mczkcn4mmvcmymbqw7113fz2flkrerfwfps004/data/listas/lista_fuera_iptv.m3u`
- [ ] Use fetch + cache + rewrite pipeline
- [ ] Return Content-Type: `audio/x-mpegurl` or `application/vnd.apple.mpegurl`
- [ ] Return 200 OK with playlist content
- [ ] Return 502 Bad Gateway if no cache and fetch fails

### US-008: Add integration tests for endpoints
**Description:** As a developer, I want integration tests that verify the complete flow so that I can ensure the feature works end-to-end.

**Acceptance Criteria:**
- [ ] Test case: Fresh fetch from mock IPFS server
- [ ] Test case: Cache hit (serve from cache without fetch)
- [ ] Test case: Expired cache refresh
- [ ] Test case: IPFS failure with stale cache fallback
- [ ] Test case: URL rewriting produces correct output
- [ ] Verify Content-Type headers are correct
- [ ] Verify HTTP status codes for all scenarios

## Functional Requirements
- FR-1: The server must fetch M3U playlists from specified IPFS URLs
- FR-2: All acestream:// URLs must be rewritten to `${ACESTREAM_PLAYER_BASE_URL}/ace/getstream?id={id}` format
- FR-3: All non-stream-URL content (metadata, comments, etc.) must be preserved exactly
- FR-4: Fetched playlists must be cached to disk in the configured CACHE_DIR
- FR-5: Cached content must include timestamp for TTL calculation
- FR-6: On each request, expired cache must trigger refresh attempt
- FR-7: If IPFS source is unreachable and cache exists, stale cache must be served
- FR-8: If IPFS source is unreachable and no cache exists, return HTTP 502
- FR-9: Cache TTL must be configurable via CACHE_TTL environment variable
- FR-10: Both endpoints must return proper Content-Type for M3U playlists

## Non-Goals (Out of Scope)
- Validating acestream ID format (40-char hex) - rewrite all acestream:// URLs blindly
- Background refresh jobs - refresh happens on-demand only
- Cache cleanup/expiration - old cache files remain on disk
- Manual cache refresh endpoint - refresh is automatic
- Aggregating multiple sources into single playlist
- Modifying playlist metadata or stream order
- Streaming content validation
- Authentication or rate limiting

## Technical Considerations
- Use existing Go HTTP server/router patterns from the codebase
- Consider using `time.Duration` parsing for CACHE_TTL (e.g., "1h30m")
- Cache file naming should avoid collisions (hash source URL)
- Consider atomic file writes for cache updates (write to temp, then rename)
- HTTP client should have reasonable timeout (10-15 seconds)
- Large playlists may require streaming/buffered processing
- IPFS URLs may be slow - cache significantly improves performance

## Success Metrics
- Both endpoints return valid M3U playlists
- All acestream:// URLs are correctly rewritten
- Cache hit rate reduces IPFS request frequency
- Service remains available when IPFS sources are down (using cache)
- Integration tests pass consistently
- No memory leaks from cache operations

## Open Questions
- Should we add metrics/logging for cache hit rates?
- Should cache files have size limits?
- Do we need graceful cache warming on server startup?
