# Design

## Context

The existing window has no macOS title-bar overrides, so AppKit draws an opaque strip above the web content. The app shell already owns its light/dark colors and a responsive 96px/224px sidebar.

## Goals / Non-Goals

**Goals:** unify the chrome and application colors, preserve native controls, and keep window dragging available.

**Non-Goals:** replace native buttons, alter authentication, change other platforms' window styling, or introduce private APIs.

## Decisions

- Use the installed Wails v3 `MacTitleBarHiddenInset` preset. It retains the native frame and traffic lights while making the title bar transparent, hiding its text/separator, and enabling full-size content. A fully frameless window would unnecessarily require replacement controls.
- Require both `System.IsMac()` and the native WKWebView message handler used by Wails. A server-backed browser preview can report macOS too, so platform detection alone must not add native chrome padding there.
- Reserve a 48px blank strip in the shell, continuing the sidebar and canvas backgrounds. Apply `--wails-draggable: drag` only there; content and controls remain below the region.
- Keep the focused skip link below native controls. Omit `InvisibleTitleBarHeight`, whose native hit-testing can intercept interactive elements independently of CSS exclusions.

## Risks / Trade-offs

- Native toolbar/control geometry varies by macOS version; inspect an actual native build for overlap and fullscreen behavior.
- Fullscreen may auto-hide native chrome. Keep the reserved layout stable and verify fullscreen entry/exit.
- The installed application must be rebuilt/relaunched to pick up native window options. Validate in a separate preview instance so the user's active practice is preserved.

## Validation

Run frontend type checks, build, existing tests, and a native Go build. Inspect native light/dark appearance, traffic-light clearance, dragging, fullscreen entry/exit, and the responsive sidebar. Record actual observed outcomes before checking the corresponding tasks.

### Observed verification

Frontend type checks, production build, and all 40 tests pass, including four new context checks for native macOS, browser/server previews, other platforms, and unrelated WebKit APIs. The native Go build succeeds; the local linker emits deployment-target warnings from its cached native objects.

A separate temporary preview bundle with its own identity visibly displayed themed top regions in both light and dark appearances, without the old gray strip or centered title. Native controls were visible and separated from the brand/content; native fullscreen entry and exit worked. The installed application's active practice was not restarted.

The computer-use drag API returned `noWindowsAvailable`, so dragging was initially left for manual verification. The user subsequently confirmed that manual window dragging works, completing the remaining native verification check. This confirmation is user-supplied evidence, not an automated gesture-test result.
