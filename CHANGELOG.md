# Changelog

## [v0.4.1] - 2026-03-24

### Added
- **User-Based Routing (Selective Outbound per User)**:
  - Admins can now specify target outbound servers for specific users directly from the V2board panel.
  - Secure implementation using UUID matching on the backend to avoid handling real user Emails.
  - Automatic Email-to-UUID translation during rule synchronization from the panel.
  - Support for dynamic loading of custom Xray outbound configurations.
  - Optimized log display: Routing rule remarks/names are now shown directly in V2bX logs to identify used rules easily.

### Changed
- Improved API client for panel data retention (preserves custom routing rules).
- Updated internal data structures for user-specific routing mappings.
