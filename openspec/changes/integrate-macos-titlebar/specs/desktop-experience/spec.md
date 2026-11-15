# Spec Delta

## ADDED Requirements

### Requirement: Integrated macOS window chrome
The macOS desktop SHALL integrate the title-bar background with the application theme, omit the duplicated centered title, preserve native window controls, and provide a draggable region without covering interactive content.

#### Scenario: Launch a themed macOS window
- WHEN the native macOS application opens in a supported light or dark appearance
- THEN its top region follows the application colors without an opaque gray title strip or centered duplicate title
- AND the native controls remain visible and unobstructed

#### Scenario: Move or enter fullscreen
- WHEN the user drags the blank top region or operates the native fullscreen control
- THEN the native window supports the requested operation without replacing its controls with HTML buttons

#### Scenario: Other runtime environments
- WHEN the interface runs outside the native macOS runtime
- THEN the app shell does not reserve space for macOS window controls
