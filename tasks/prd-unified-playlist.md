# PRD: Unified Playlist Endpoint

## Overview
Create a new endpoint `/playlist.m3u` that returns a unified playlist combining all streams from both the NewEra and Elcano playlists. The endpoint will deduplicate streams by acestream ID, sort them alphabetically by display name, remove logo metadata from all entries, and add a `network-caching=1000` parameter to all acexy URLs.

## Goals
- Provide a single endpoint that aggregates all available streams from multiple sources
- Eliminate duplicate streams based on acestream ID
- Present streams in a consistent, alphabetically sorted order
- Remove logo metadata to reduce playlist size and bandwidth
- Optimize playback with network-caching parameter for acexy URLs

## Quality Gates

These commands must pass for every user story:
- `go build ./...` - Build verification
- `go test ./...` - All tests passing

## User Stories

### US-001: Create unified playlist endpoint handler
**Description:** As a user, I want to access `/playlist.m3u` so that I get a single playlist containing all available streams.

**Acceptance Criteria:**
- [ ] New endpoint registered at `GET /playlist.m3u`
- [ ] Returns HTTP 405 Method Not Allowed for non-GET requests
- [ ] Returns `Content-Type: audio/x-mpegurl` header
- [ ] Returns HTTP 200 OK on success
- [ ] Returns HTTP 502 Bad Gateway if both sources fail and no cache exists

### US-002: Fetch and merge playlists from both sources
**Description:** As a user, I want the unified playlist to contain streams from both NewEra and Elcano sources so that I have access to all available content.

**Acceptance Criteria:**
- [ ] Fetches content from Elcano source URL using existing FetchWithCache
- [ ] Fetches content from NewEra source URL using existing FetchWithCache
- [ ] Merges entries from both playlists into a single list
- [ ] Handles partial failures gracefully (serves available content if one source fails but cache exists for both)

### US-003: Deduplicate streams by acestream ID
**Description:** As a user, I want duplicate streams removed so that the playlist doesn't contain redundant entries.

**Acceptance Criteria:**
- [ ] Streams with identical acestream IDs are deduplicated
- [ ] First occurrence is kept when duplicates exist
- [ ] Non-acestream URLs are preserved (no deduplication applied)
- [ ] Deduplication logic is implemented in a testable function

### US-004: Sort streams alphabetically by display name
**Description:** As a user, I want streams sorted alphabetically by name so that I can easily find channels.

**Acceptance Criteria:**
- [ ] Streams are sorted case-insensitively by display name
- [ ] Display name is extracted from text after comma in EXTINF line
- [ ] `#EXTM3U` header remains at the top of the playlist
- [ ] Sorting logic is implemented in a testable function

### US-005: Remove logo metadata from all streams
**Description:** As a user, I want logo metadata removed so that the playlist is smaller and loads faster.

**Acceptance Criteria:**
- [ ] `tvg-logo="..."` attribute is removed from all EXTINF lines
- [ ] Other metadata attributes (tvg-id, tvg-name, group-title) are preserved
- [ ] Logo removal handles various formats (with/without quotes, different URL patterns)
- [ ] Logo removal logic is implemented in a testable function

### US-006: Add network-caching parameter to acexy URLs
**Description:** As a user, I want acexy URLs to include network-caching parameter so that playback is optimized.

**Acceptance Criteria:**
- [ ] Rewritten acexy URLs include `&network-caching=1000` parameter
- [ ] Parameter is appended after the existing `?id={streamID}` parameter
- [ ] Non-acestream URLs are not modified
- [ ] URL format: `{baseURL}?id={streamID}&network-caching=1000`

### US-007: Add integration tests for unified playlist endpoint
**Description:** As a developer, I want integration tests so that the endpoint behavior is verified.

**Acceptance Criteria:**
- [ ] Test for successful fetch and merge from both sources
- [ ] Test for deduplication behavior
- [ ] Test for alphabetical sorting
- [ ] Test for logo removal
- [ ] Test for network-caching parameter presence
- [ ] Test for cache fallback behavior
- [ ] Tests follow existing patterns in `main_test.go`

## Functional Requirements
- FR-1: The endpoint must be accessible at `GET /playlist.m3u`
- FR-2: The endpoint must fetch playlists from both Elcano and NewEra source URLs
- FR-3: The endpoint must use the existing caching mechanism (FetchWithCache)
- FR-4: Streams with duplicate acestream IDs must be deduplicated, keeping the first occurrence
- FR-5: All streams must be sorted alphabetically (case-insensitive) by display name
- FR-6: The `tvg-logo` attribute must be removed from all EXTINF metadata lines
- FR-7: All rewritten acexy URLs must include `&network-caching=1000` parameter
- FR-8: The `#EXTM3U` header must be present at the start of the response
- FR-9: The response must have `Content-Type: audio/x-mpegurl`

## Non-Goals
- Custom sorting options (e.g., by group, by source)
- Filtering by group or category
- Configurable network-caching value (hardcoded to 1000)
- Logo replacement (only removal)
- Merging more than two sources (only Elcano and NewEra)

## Technical Considerations
- Extend or create new functions in `rewriter/rewriter.go` for logo removal and parameter addition
- Create a new `merger` package or add merge/sort/dedupe logic to a new file
- Parsing EXTINF lines requires extracting display name (text after last comma)
- Consider memory efficiency when processing large playlists
- Existing test patterns use mock HTTP servers for integration tests

## Success Metrics
- All integration tests pass
- Endpoint returns combined, deduplicated, sorted playlist
- No logo attributes present in response
- All acexy URLs contain network-caching parameter
- Response time comparable to existing single-source endpoints

## Open Questions
- Should the endpoint have its own dedicated cache key, or cache each source separately?
- If one source is completely unavailable (no cache), should we return partial results from the other source?
