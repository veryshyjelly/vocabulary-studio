# Proposal

## Why

The default macOS gray title bar and duplicated centered application title clash with the app's themed interface. The user requested a more integrated desktop appearance.

## What Changes

- Extend themed content behind a transparent native macOS title bar and remove the centered title/separator.
- Preserve native close, minimize, and fullscreen controls with reserved clearance and a blank draggable region.
- Gate the additional chrome to the native macOS runtime.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `desktop-experience`: integrate macOS window chrome with the selected application theme while retaining native window behavior.

## Impact

Changes the Wails window options, app-shell spacing, and drag-region styling. No SDK, bridge, account, or persistence changes. Existing native title bars and content layout on other platforms remain available.

## Out of Scope

No custom HTML replacements for native window controls, private macOS APIs, translucency effects, or general application redesign.
