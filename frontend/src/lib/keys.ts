export function isTypingTarget(target: EventTarget | null): boolean {
  const el = target as HTMLElement | null;
  if (!el || !el.tagName) return false;
  const tag = el.tagName.toLowerCase();
  return tag === "input" || tag === "textarea" || tag === "select" || el.isContentEditable;
}

export function isModalOpen(): boolean {
  return typeof document !== "undefined" && document.querySelector('[role="dialog"][aria-modal="true"]') !== null;
}

/** True when a single-key shortcut is eligible outside text entry and modal interaction. */
export function acceptsPlainShortcut(e: KeyboardEvent): boolean {
  // Shift remains allowed for punctuation shortcuts such as "?".
  if (e.defaultPrevented || e.isComposing || e.repeat || e.metaKey || e.ctrlKey || e.altKey) return false;
  if (isModalOpen()) return false;
  return !isTypingTarget(e.target);
}

export const isMac = typeof navigator !== "undefined" && /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent);

export const modKeyLabel = isMac ? "⌘" : "Ctrl";
