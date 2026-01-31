# PRD: Web UI for Channel Overrides Management

## Overview
A web-based admin interface for managing IPTV channel overrides with EPG validation. The UI allows editing all M3U attributes for channels (identified by AcestreamID), validates TVG-ID values against the EPG source (https://raw.githubusercontent.com/davidmuma/EPG_dobleM/master/guiatv.xml), and supports bulk editing. The React frontend is embedded in the Go binary to maintain single Docker image deployment.

## Goals
- Provide a user-friendly interface for managing channel overrides without editing YAML files manually
- Validate TVG-ID values against the EPG XML to ensure correct EPG mappings
- Suggest valid TVG-IDs when validation fails, while allowing force-save for edge cases
- Support bulk editing to efficiently update multiple channels at once
- Maintain single Docker image deployment by embedding the React build in the Go binary

## Quality Gates

These commands must pass for every user story:
- `go test ./...` - Run all Go tests
- `go vet ./...` - Static analysis

For frontend stories, also include:
- `npm run build` in the web-ui directory - Ensure frontend builds successfully

## User Stories

### US-001: EPG XML fetching and caching service
**Description:** As a system administrator, I want the EPG XML to be fetched and cached on startup so that TVG-ID validation is fast and doesn't require network calls for each validation.

**Acceptance Criteria:**
- [ ] Create `epg` package with EPGCache struct
- [ ] Fetch EPG XML from configurable URL (default: https://raw.githubusercontent.com/davidmuma/EPG_dobleM/master/guiatv.xml) on startup
- [ ] Parse EPG XML and extract all valid channel IDs into a map for O(1) lookup
- [ ] Store channel display names alongside IDs for suggestion feature
- [ ] Expose method to check if a TVG-ID is valid
- [ ] Expose method to search/suggest TVG-IDs by partial match
- [ ] Log EPG cache initialization success/failure with channel count

### US-002: API endpoint for listing channels with override status
**Description:** As an admin, I want to see all available channels with their current override status so that I can identify which channels need configuration.

**Acceptance Criteria:**
- [ ] Create `GET /api/channels` endpoint that returns all channels from processed M3U
- [ ] Include current values (original + any applied overrides) for each channel
- [ ] Include `hasOverride` boolean flag for each channel
- [ ] Include AcestreamID as the unique identifier
- [ ] Support optional query parameter for filtering by name/group
- [ ] Return JSON response with proper error handling

### US-003: API endpoint for validating TVG-ID
**Description:** As an admin, I want to validate a TVG-ID before saving so that I can ensure correct EPG mapping.

**Acceptance Criteria:**
- [ ] Create `POST /api/validate/tvg-id` endpoint accepting `{"tvg_id": "string"}`
- [ ] Return `{"valid": true}` if TVG-ID exists in EPG cache
- [ ] Return `{"valid": false, "suggestions": [...]}` with up to 10 closest matches if invalid
- [ ] Suggestions should be sorted by relevance (exact prefix match first, then fuzzy)
- [ ] Handle empty/null TVG-ID as valid (means "no EPG")

### US-004: API endpoints for CRUD operations on overrides
**Description:** As an admin, I want API endpoints to create, read, update, and delete channel overrides so that the web UI can manage overrides.

**Acceptance Criteria:**
- [ ] `GET /api/overrides/:acestreamId` - Get override for specific channel
- [ ] `PUT /api/overrides/:acestreamId` - Create or update override (supports all M3U attributes)
- [ ] `DELETE /api/overrides/:acestreamId` - Delete override for channel
- [ ] `GET /api/overrides` - List all overrides
- [ ] Support `force` query parameter on PUT to skip TVG-ID validation
- [ ] Return 400 with validation errors if TVG-ID is invalid and force=false
- [ ] All endpoints use existing `overrides.Manager` for persistence

### US-005: API endpoint for bulk override updates
**Description:** As an admin, I want to update a single field across multiple channels at once so that I can efficiently manage group changes.

**Acceptance Criteria:**
- [ ] Create `PATCH /api/overrides/bulk` endpoint
- [ ] Accept JSON body: `{"acestream_ids": [...], "field": "group_title", "value": "Sports"}`
- [ ] Support all override fields: enabled, tvg_id, tvg_name, tvg_logo, group_title
- [ ] Validate TVG-ID if that's the field being updated (unless force=true)
- [ ] Return summary: `{"updated": 5, "failed": 1, "errors": [...]}`
- [ ] Atomic operation: all updates succeed or none (with option for partial)

### US-006: React project setup with Vite
**Description:** As a developer, I want a minimal React project setup so that I can build the web UI with modern tooling.

**Acceptance Criteria:**
- [ ] Create `web-ui/` directory with Vite + React + TypeScript setup
- [ ] Configure Vite to output build to `web-ui/dist/`
- [ ] Add basic folder structure: components/, hooks/, api/, types/
- [ ] Configure ESLint and Prettier for code quality
- [ ] Add npm scripts: dev, build, test, lint
- [ ] Create .gitignore for node_modules and dist

### US-007: Embed React build in Go binary
**Description:** As a system administrator, I want the web UI embedded in the Go binary so that deployment remains a single Docker image.

**Acceptance Criteria:**
- [ ] Use Go 1.16+ `embed` directive to embed `web-ui/dist/`
- [ ] Create HTTP handler that serves embedded static files
- [ ] Mount UI at `/ui/` path (or configurable)
- [ ] Serve `index.html` for all non-API routes under `/ui/` (SPA routing)
- [ ] Set correct Content-Type headers for JS, CSS, and other assets
- [ ] Update Dockerfile to build frontend before Go binary

### US-008: Channel list view with filtering
**Description:** As an admin, I want to see all channels in a table with search/filter capabilities so that I can quickly find channels to configure.

**Acceptance Criteria:**
- [ ] Create ChannelList component displaying channels in a table
- [ ] Columns: checkbox (for bulk select), Name, Group, TVG-ID, Status (has override indicator)
- [ ] Search input that filters by channel name (client-side)
- [ ] Dropdown filter for group-title
- [ ] Visual indicator for channels with existing overrides
- [ ] Click row to open edit panel
- [ ] Responsive layout for different screen sizes

### US-009: Channel override edit form
**Description:** As an admin, I want a form to edit all M3U attributes for a channel so that I can configure overrides.

**Acceptance Criteria:**
- [ ] Create EditOverrideForm component as a side panel or modal
- [ ] Display current (original) values as placeholders
- [ ] Input fields for: enabled (toggle), tvg_id, tvg_name, tvg_logo, group_title
- [ ] Support custom attributes as key-value pairs (add/remove)
- [ ] TVG-ID field shows validation status in real-time (debounced)
- [ ] Show suggestions dropdown when TVG-ID is invalid
- [ ] Save button with loading state
- [ ] "Force save" checkbox when validation fails
- [ ] Cancel button to close without saving
- [ ] Delete override button (with confirmation)

### US-010: TVG-ID autocomplete with EPG validation
**Description:** As an admin, I want TVG-ID input to show validation status and suggestions so that I can easily find the correct EPG mapping.

**Acceptance Criteria:**
- [ ] TVG-ID input validates on blur and while typing (debounced 300ms)
- [ ] Show green checkmark icon when valid
- [ ] Show red X icon with warning message when invalid
- [ ] Show dropdown with suggestions when invalid (clickable to select)
- [ ] Allow typing to filter suggestions
- [ ] Show "Force save anyway" option when invalid
- [ ] Empty value is valid (means no EPG mapping)

### US-011: Bulk edit functionality
**Description:** As an admin, I want to select multiple channels and update a single field for all of them so that I can efficiently manage group changes.

**Acceptance Criteria:**
- [ ] Checkbox in table header selects/deselects all visible channels
- [ ] "Bulk Edit" button appears when channels are selected
- [ ] Bulk edit modal shows: field selector dropdown, value input
- [ ] Field selector includes: enabled, tvg_id, tvg_name, tvg_logo, group_title
- [ ] Preview shows number of channels to be updated
- [ ] Submit calls bulk API endpoint
- [ ] Show success/error summary after completion
- [ ] Clear selection after successful bulk update

### US-012: Error handling and user feedback
**Description:** As an admin, I want clear feedback on all operations so that I understand what succeeded or failed.

**Acceptance Criteria:**
- [ ] Toast notifications for success/error on save operations
- [ ] Loading spinners during API calls
- [ ] Graceful error display when API is unreachable
- [ ] Form validation errors displayed inline
- [ ] Confirmation dialog before destructive actions (delete)
- [ ] Retry option on failed operations

## Functional Requirements

- FR-1: The system must fetch and cache EPG XML on startup
- FR-2: The system must validate TVG-ID values against cached EPG data
- FR-3: The system must suggest valid TVG-IDs when an invalid value is entered
- FR-4: The system must allow force-saving invalid TVG-IDs with explicit user confirmation
- FR-5: The system must persist overrides to YAML file (existing format)
- FR-6: The web UI must be served from the same Go binary (embedded)
- FR-7: All API endpoints must return JSON with appropriate HTTP status codes
- FR-8: The UI must support bulk editing of a single field across multiple channels
- FR-9: The system must support all M3U attributes including custom ones

## Non-Goals (Out of Scope)

- User authentication/authorization (single admin user assumed)
- EPG XML editing or custom EPG sources management
- Real-time EPG cache refresh (manual restart required for EPG updates)
- Channel reordering or sorting preferences persistence
- Import/export of overrides as separate file
- Mobile-optimized interface (desktop-first, basic responsiveness only)
- Undo/redo functionality for override changes
- Multi-language support (English only)

## Technical Considerations

- **Existing overrides package**: Reuse `overrides.Manager` for all CRUD operations
- **EPG XML size**: The EPG file may be large; parse efficiently and store only channel IDs
- **Embedding strategy**: Use `//go:embed` with build tag to optionally exclude UI in dev
- **Vite proxy**: Configure Vite dev server to proxy `/api` to Go backend for development
- **Bundle size**: Keep React bundle minimal - consider Preact if size becomes an issue
- **Docker build**: Multi-stage Dockerfile - build frontend, then Go binary with embedded assets

## Success Metrics

- All channels from M3U are visible in the UI
- TVG-ID validation correctly identifies valid/invalid IDs
- Overrides persist correctly to YAML file
- Bulk edit successfully updates multiple channels
- Single Docker image deployment works as before
- Go tests pass for all new API endpoints
- Frontend builds without errors

## Open Questions

- Should we add EPG cache refresh endpoint for manual refresh without restart?
- What fuzzy matching algorithm should be used for TVG-ID suggestions? (Levenshtein, prefix match, etc.)
- Should custom M3U attributes be fully dynamic or limited to a predefined set?
