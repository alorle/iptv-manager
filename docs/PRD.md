# Product Requirements Document: IPTV Manager Redesign

---

## 1. Introduction / Overview

IPTV Manager is a self-hosted web application designed to manage Acestream streams with Electronic Program Guide (EPG) integration. The application enables users to organize streaming sources, match them with EPG metadata, and generate M3U playlists compatible with media players like Jellyfin, Plex, and similar IPTV clients.

This redesign shifts from a channel-centric to a **stream-centric architecture**, where channels emerge from grouping streams by channel name rather than being separate entities. This simplifies data management while maintaining flexibility for multiple stream sources per channel.

**Target Release**: MVP with all core functionality described in this document

---

## 2. Goals / Objectives

### Primary Goals
1. **Reduce friction in stream management**: Enable import of 20-30 new streams in under 5 minutes with minimal manual data entry
2. **Ensure data integrity**: Eliminate duplicate streams through automatic detection and user-controlled resolution
3. **Improve reliability**: Generate consistent M3U playlists that work seamlessly with Jellyfin and similar clients without manual intervention

### Success Metrics
- Time to import 30 streams: < 5 minutes (from paste to saved)
- Duplicate stream rate: 0% (all duplicates caught and handled)
- Playlist generation success rate: 100% (no broken/malformed entries)
- EPG matching accuracy: >80% auto-matched correctly (remaining 20% manually confirmed)

---

## 3. Target Audience / User Personas

**Primary User: Technical Home Media Enthusiast**
- Self-hosts media infrastructure (Jellyfin, Plex, etc.)
- Comfortable with Docker and basic server administration
- Discovers Acestream links from various online sources
- Manages 50-100 active streams across 30-50 channels
- Values clean, intuitive UI despite technical capability
- Expects "it just works" experience once configured

**Key Pain Points**:
- Difficulty identifying which channel new Acestream links belong to
- Uncertainty about whether a stream is already configured
- Time-consuming manual metadata entry
- Managing multiple quality options per channel

---

## 4. User Stories / Use Cases

### US-1: Bulk Stream Import
**As a user**, I want to paste a list of Acestream streams from a website and have the system automatically parse channel names, quality, and tags, **so that** I can quickly add new streams without manual data entry.

**Acceptance Criteria**:
- Support text format (alternating description/ID lines)
- Support JSON format (name/url pairs)
- Parse and extract: channel name, quality (FHD/HD/SD/4K), tags, acestream_id
- Validate 40-character hex acestream_id
- Show preview table with auto-detected fields before saving
- Allow editing all fields except acestream_id in preview

### US-2: Intelligent Channel Matching
**As a user**, I want the system to suggest matching EPG channels when importing streams, **so that** I don't have to manually search and match channel names.

**Acceptance Criteria**:
- Fuzzy match channel names against loaded EPG
- Show top match + 2-3 alternatives when confidence is low
- Allow manual channel name entry when no EPG match exists
- Channel name field restricted to EPG channels (dropdown selection)

### US-3: Duplicate Stream Handling
**As a user**, I want to be notified when importing a stream that already exists, **so that** I can decide whether to skip, replace, or keep both.

**Acceptance Criteria**:
- Detect duplicates by acestream_id (globally unique)
- Show "Already exists in [Channel X]" warning in preview
- Provide action dropdown: Skip / Replace / Keep Both
- Display which channel currently has the duplicate stream

### US-4: Stream Management View
**As a user**, I want to view all streams grouped by channel with EPG metadata, **so that** I can see my complete configuration at a glance.

**Acceptance Criteria**:
- Card-based layout, one card per channel
- Display channel logo, name, and EPG metadata on card
- List all streams within each channel card
- Show stream quality, tags, and notes for each stream
- Filter/search by channel name or tags

### US-5: Stream Reordering
**As a user**, I want to reorder streams within a channel by dragging, **so that** I can control priority in the M3U playlist.

**Acceptance Criteria**:
- Drag-and-drop interface within channel card
- Reordering limited to same channel (no cross-channel dragging)
- Automatically update `order` field based on visual position
- Lower order value = higher priority in M3U playlist
- Changes saved immediately or with explicit save action

### US-6: Stream Editing and Deletion
**As a user**, I want to edit stream notes/tags and delete streams, **so that** I can maintain accurate metadata and remove broken sources.

**Acceptance Criteria**:
- Inline editing for notes and tags fields
- Checkbox selection for batch deletion
- Confirmation dialog for delete actions
- Support deleting multiple streams at once

### US-7: M3U Playlist Generation
**As a user**, I want to access a single M3U playlist endpoint with all configured streams, **so that** I can import it into Jellyfin without manual configuration.

**Acceptance Criteria**:
- Single endpoint: `/playlist.m3u`
- One M3U entry per stream (flattened from channel→streams structure)
- Numbered format: "Channel (#1)", "Channel (#2)" for multiple streams
- Include quality and tags in entry title
- Include TVG metadata (tvg-id, tvg-logo, group-title)
- No authentication required

### US-8: Partial Import with Validation
**As a user**, I want to import only valid streams when some entries have errors, **so that** I don't lose all work due to a few bad entries.

**Acceptance Criteria**:
- Highlight invalid entries in preview (invalid acestream_id, parsing errors)
- Allow confirming import of only valid entries
- Show count of valid vs. invalid entries
- Provide error details for invalid entries

### US-9: System Status Visibility
**As a user**, I want to see connection status for external services (EPG, Acestream engine), **so that** I understand when functionality may be degraded.

**Acceptance Criteria**:
- Status indicators in UI for: EPG service, Acestream engine
- Visual indicators: Connected (green), Disconnected (red), Unknown (gray)
- Allow manual retry for failed connections
- Graceful degradation: allow manual channel entry when EPG unavailable

---

## 5. Functional Requirements

### FR-1: Data Model
- **Stream entity**: `{id: uuid, acestream_id: string, channel_name: string, quality: string, tags: []string, notes: string, order: int}`
- **Uniqueness constraint**: One stream per `acestream_id` globally
- **Channel concept**: Virtual grouping by `channel_name` field (no separate channel entity)
- **Storage**: JSON file (`streams.json`) containing flat array of streams
- **Future migration path**: SQLite database (post-MVP)

### FR-2: Bulk Import - Parsing
- **Text format parser**:
  - Pattern: Line 1 = description, Line 2 = acestream_id (alternating)
  - Example: `DAZN LA LIGA 1 FHD --> NEW ERA VI` followed by `0e50439e68aa2435b38f0563bb2f2e98f32ff4b1`
- **JSON format parser**:
  - Expected structure: Array of `{name: string, url: string}` objects
  - Extract acestream_id from URL
- **Quality detection**: Pattern-based extraction (FHD, HD, SD, 4K, 1080p, 720p, etc.)
- **Tags extraction**: Space-separated tokens from remaining text after channel/quality removal
- **Validation**: 40-character hexadecimal acestream_id

### FR-3: EPG Integration
- **Format**: XMLTV with `<channel>` and `<display-name>` tags
- **Fetch strategy**: Daily refresh or on-demand (lazy load + cache)
- **Matching algorithm**: Fuzzy string matching on channel names
- **Confidence scoring**: Return top match + alternatives when confidence < threshold
- **Fallback**: Allow manual channel name entry when no match found

### FR-4: API Endpoints

| Endpoint | Method | Purpose | Request | Response |
|----------|--------|---------|---------|----------|
| `/api/streams/preview` | POST | Parse input, return preview (no save) | Text or JSON stream data | Array of parsed streams with match suggestions |
| `/api/streams/import` | POST | Save confirmed streams | Array of confirmed stream objects | Success/failure status |
| `/api/streams` | GET | Retrieve streams grouped by channel | Query params for filters | Streams grouped by channel + EPG metadata |
| `/api/streams/{id}` | PATCH | Update stream fields | Stream ID + fields to update | Updated stream object |
| `/api/streams/{id}` | DELETE | Delete single stream | Stream ID | Success status |
| `/api/streams/batch-delete` | POST | Delete multiple streams | Array of stream IDs | Success status |
| `/api/streams/{id}/reorder` | POST | Change order within channel | Stream ID + new position | Updated stream order |
| `/playlist.m3u` | GET | Generate M3U playlist | None | M3U formatted text |
| `/api/status` | GET | System status check | None | EPG status, Acestream status |

### FR-5: M3U Playlist Format
- **Entry naming**:
  - If only one stream: `Channel Name [Quality] [Tag1] [Tag2]`
  - If several streams: `Channel Name (#N) [Quality] [Tag1] [Tag2]`
- **TVG attributes**:
  - `tvg-id`: EPG channel ID
  - `tvg-logo`: Channel logo URL
  - `group-title`: Channel category/group
- **URL format**: `{ACESTREAM_URL}/ace/getstream?id={acestream_id}&.mp4`
- **Ordering**: Sorted by channel name, then by `order` field within channel

### FR-6: Error Handling
- **EPG fetch failure**: Allow manual channel name entry, show warning banner
- **Acestream engine unreachable**: Show status indicator, allow configuration changes
- **Invalid import data**: Highlight errors, allow partial import
- **Duplicate detection**: Non-blocking warning with user action required
- **File save failures**: Show error message, preserve user input for retry

---

## 6. Non-Functional Requirements

### NFR-1: Performance
- **Scale**: Support 50-100 streams comfortably (target: up to 200)
- **Import speed**: Process and preview 30 streams in < 2 seconds
- **UI responsiveness**: Card rendering and drag-drop operations < 100ms
- **Playlist generation**: < 500ms for full playlist with 100 streams

### NFR-2: Usability
- **UI polish**: Modern, clean interface with intuitive navigation
- **Consumer-grade UX**: Hide technical complexity behind simple interactions
- **Responsive design**: Usable on desktop/laptop (mobile not required for MVP)
- **Visual feedback**: Loading states, success/error messages, status indicators
- **Accessibility**: Keyboard navigation for drag-drop, ARIA labels for screen readers (nice-to-have)

### NFR-3: Reliability
- **Data integrity**: Atomic saves (all-or-nothing), validate before writing
- **Idempotent operations**: Safe to retry failed requests
- **Error recovery**: Graceful degradation when external services unavailable
- **No data loss**: Preserve user input during errors until explicitly discarded

### NFR-4: Maintainability
- **Code structure**: Clean Architecture (domain, use case, repository, API layers)
- **Type safety**: TypeScript frontend, strongly-typed Go backend
- **API contract**: OpenAPI spec as single source of truth
- **Code generation**: Auto-generate API client and server code from spec

### NFR-5: Deployment
- **Packaging**: Single Docker container with backend + frontend
- **Configuration**: Environment variables for URLs and paths
- **Port**: Single HTTP port (default: 8080)
- **Persistence**: Volume mount for `streams.json`
- **No authentication**: Assumed self-hosted, single user, trusted network

### NFR-6: Security
- **Input validation**: Sanitize all user input (prevent injection attacks)
- **CORS**: Restrict to same-origin (no external access needed)
- **File access**: Restrict to designated streams file location
- **Dependencies**: Regular security updates for libraries

---

## 7. Design Considerations

### UI Components

**Main Layout**:
- Header: App title, system status indicators, import button
- Main area: Scrollable card grid
- Footer: Playlist URL, copy button

**Import Modal**:
- Large textarea for paste input
- Format auto-detection (text vs. JSON)
- Preview table with editable fields
- Column headers: Channel (dropdown), Quality, Tags, Notes, Status/Actions
- Highlight rows: Valid (green), Invalid (red), Duplicate (yellow)
- Action buttons: Cancel, Import Valid Only, Import All

**Channel Card**:
- Header: Channel logo, name, EPG status badge
- Body: List of streams with quality/tags badges
- Per-stream actions: Edit (inline), Delete checkbox
- Footer: Stream count, drag handle for reordering

**Status Indicators**:
- Small colored dots or icons
- Tooltip on hover with details
- Placement: Top-right corner of header

### Design System
- **Colors**: Dark mode primary (existing app uses dark theme)
- **Typography**: System fonts, clear hierarchy
- **Spacing**: Consistent padding/margins (8px grid)
- **Icons**: Material Design Icons or similar
- **Components**: Reuse existing UI library where possible

---

## 8. Success Metrics

### Quantitative Metrics
1. **Import efficiency**: Average time to import 30 streams < 5 minutes
2. **Duplicate prevention**: 100% of duplicate acestream_ids detected
3. **EPG matching accuracy**: >80% of streams auto-matched correctly
4. **Playlist reliability**: 0 malformed M3U entries
5. **User error rate**: <5% of imports require retry due to errors

### Qualitative Metrics
1. **User satisfaction**: "Feels fast and intuitive to use"
2. **Discovery clarity**: "I can immediately see if I already have a stream"
3. **Confidence**: "I trust the EPG matching suggestions"
4. **Reliability**: "My Jellyfin playlist just works without tweaking"

### Validation Methods
- **Manual testing**: Complete user workflows with realistic data
- **Performance benchmarks**: Measure import/render times with 100+ streams
- **Dogfooding**: Daily use by primary user for 2-4 weeks
- **Regression testing**: Ensure existing M3U clients still work

---

## 9. Technical Architecture Summary

### Backend (Go)
- **Framework**: Standard library HTTP server + Chi router
- **API generation**: oapi-codegen from OpenAPI spec
- **Data access**: In-memory repository with JSON file persistence
- **EPG parsing**: XMLTV library for parsing, custom fuzzy matching
- **Acestream integration**: URL construction (no direct engine communication)

### Frontend (React + TypeScript)
- **Build tool**: Vite
- **State management**: TanStack Query for server state
- **API client**: openapi-fetch with generated types
- **UI library**: Existing component library (continue current pattern)
- **Drag-and-drop**: react-beautiful-dnd or @dnd-kit

### Deployment
- **Container**: Multi-stage Dockerfile (build frontend → embed in Go binary)
- **Volumes**: Mount for `streams.json` persistence
- **Ports**: Single HTTP port exposed
- **Environment**: `HTTP_PORT`, `STREAMS_FILE`, `EPG_URL`, `ACESTREAM_URL`

### Data Flow
1. User pastes stream data → Frontend sends to `/api/streams/preview`
2. Backend parses, fetches EPG, performs fuzzy matching → Returns suggestions
3. User edits preview table → Frontend sends to `/api/streams/import`
4. Backend validates, checks duplicates, writes to `streams.json` → Returns success
5. Frontend refetches stream list, updates UI

---

## 10. Open Questions / Future Considerations

### Post-MVP Features (Deferred)
1. **Stream health monitoring**: Automatic connection testing, bitrate detection, buffering metrics
2. **Thumbnail grid view**: Auto-play previews for visual channel identification
3. **SQLite migration**: Replace JSON file storage for better concurrency and querying
4. **Multi-user support**: Authentication, per-user stream configurations
5. **Stream scheduling**: Time-based filtering (e.g., hide sports channels outside game times)
6. **EPG guide UI**: Built-in TV guide interface (currently only used for matching)
7. **Backup/restore**: Automatic snapshots, export/import configurations
8. **API rate limiting**: Protection against abuse if exposed to internet

### Technical Debt to Address Later
- **UUID persistence**: Currently regenerated on restart (acceptable for MVP with in-memory data)
- **Concurrent writes**: JSON file locking not implemented (single user mitigates risk)
- **EPG refresh strategy**: Manual trigger only (no background scheduled refresh)
- **Error telemetry**: No logging/monitoring beyond console output

### Unresolved Design Decisions
- **Network caching parameter**: Should `networkCaching` be per-stream configurable or global default?
- **Tag taxonomy**: Free-form vs. predefined tag list?
- **Quality normalization**: Should "1080p" be normalized to "FHD" for consistency?
- **Stream URL parameters**: Any additional Acestream parameters needed beyond `id`?

---

## 11. Constraints & Assumptions

### Constraints
- **No authentication**: Self-hosted, single user, trusted network assumed
- **JSON storage**: Must remain human-editable for troubleshooting
- **Backward compatibility**: Existing `streams.json` format must be migrated automatically
- **No stream testing**: Acestream engine connectivity not validated (deferred feature)

### Assumptions
- User has Acestream engine running locally or on network
- EPG XMLTV feed is accessible and regularly updated
- Jellyfin/IPTV clients support standard M3U format with TVG tags
- Docker environment available for deployment
- User comfortable with basic command-line operations for initial setup

---

## Appendix: Example Data Formats

### Example streams.json (Current Format - To Be Migrated)
```json
{
  "channels": [
    {
      "title": "DAZN LA LIGA 1",
      "guideId": "DaznLaLiga1.es",
      "logo": "http://example.com/logo.png",
      "groupTitle": "Sports",
      "streams": [
        {
          "acestream_id": "0e50439e68aa2435b38f0563bb2f2e98f32ff4b1",
          "quality": "FHD",
          "tags": ["NEW ERA VI"],
          "networkCaching": 10000
        }
      ]
    }
  ]
}
```

### Example streams.json (New Format - Stream-Centric)
```json
{
  "streams": [
    {
      "id": "uuid-here",
      "acestream_id": "0e50439e68aa2435b38f0563bb2f2e98f32ff4b1",
      "channel_name": "DAZN LA LIGA 1",
      "quality": "FHD",
      "tags": ["NEW ERA VI"],
      "notes": "Backup stream",
      "order": 1
    }
  ]
}
```

### Example Import Input (Text Format)
```
DAZN LA LIGA 1 FHD --> NEW ERA VI
0e50439e68aa2435b38f0563bb2f2e98f32ff4b1
DAZN LA LIGA 1 SD --> ELCANO
4e6d9cf7d177366045d33cd8311d8b1d7f4bed1f
```

### Example Import Input (JSON Format)
```json
[
  {
    "name": "DAZN LA LIGA 1 FHD --> NEW ERA VI",
    "url": "acestream://0e50439e68aa2435b38f0563bb2f2e98f32ff4b1"
  },
  {
    "name": "DAZN LA LIGA 1 SD --> ELCANO",
    "url": "acestream://4e6d9cf7d177366045d33cd8311d8b1d7f4bed1f"
  }
]
```

### Example M3U Output
```m3u
#EXTM3U

#EXTINF:-1 tvg-id="DaznLaLiga1.es" tvg-logo="http://example.com/logo.png" group-title="Sports",DAZN LA LIGA 1 [FHD] [NEW ERA VI]
http://127.0.0.1:6878/ace/getstream?id=0e50439e68aa2435b38f0563bb2f2e98f32ff4b1&.mp4

#EXTINF:-1 tvg-id="DaznLaLiga1.es" tvg-logo="http://example.com/logo.png" group-title="Sports",DAZN LA LIGA 1 (#2) [SD] [ELCANO]
http://127.0.0.1:6878/ace/getstream?id=4e6d9cf7d177366045d33cd8311d8b1d7f4bed1f&.mp4
```

---

**End of PRD**

---

This PRD is comprehensive and ready for implementation. All requirements have been elicited through our collaborative discussion and organized into a clear, actionable structure. Next steps would typically include design mockups, technical spike for fuzzy matching algorithm, and breaking down into implementation tasks.
