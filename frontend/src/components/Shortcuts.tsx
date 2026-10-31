import React from "react";
import { Keyboard, X } from "lucide-react";
import { Dialog, IconButton, Kbd } from "./ui";
import { modKeyLabel } from "../lib/keys";

export interface ShortcutGroup {
  title: string;
  items: { keys: string[]; action: string }[];
}

export const SHORTCUT_GROUPS: ShortcutGroup[] = [
  {
    title: "Anywhere",
    items: [
      { keys: [modKeyLabel, "K"], action: "Quick word lookup" },
      { keys: [modKeyLabel, "1–7"], action: "Jump to a section" },
      { keys: [modKeyLabel, ","], action: "Open settings" },
      { keys: ["?"], action: "Show keyboard shortcuts" },
    ],
  },
  {
    title: "Practice",
    items: [
      { keys: ["1–4"], action: "Choose an answer (or A–D)" },
      { keys: ["H"], action: "Use the first available lifeline" },
      { keys: ["Enter"], action: "Next question after a correct answer" },
    ],
  },
  {
    title: "Dictionary & Lists",
    items: [
      { keys: ["/"], action: "Focus dictionary search" },
      { keys: ["↑", "↓"], action: "Move through suggestions" },
      { keys: ["Space"], action: "Flip flashcard" },
      { keys: ["←", "→"], action: "Previous / next flashcard" },
    ],
  },
];

export const ShortcutList: React.FC<{ columns?: 1 | 2 | 3 }> = ({ columns = 3 }) => (
  <div
    className={
      columns === 3
        ? "grid grid-cols-1 md:grid-cols-3 gap-5"
        : columns === 2
        ? "grid grid-cols-1 sm:grid-cols-2 gap-5"
        : "space-y-5"
    }
  >
    {SHORTCUT_GROUPS.map((group) => (
      <div key={group.title} className="space-y-2">
        <h3 className="eyebrow text-ink-subtle">{group.title}</h3>
        <ul className="space-y-1">
          {group.items.map((item) => (
            <li key={item.action} className="flex items-center justify-between gap-3 py-1">
              <span className="text-meta text-ink-muted">{item.action}</span>
              <span className="flex items-center gap-1 shrink-0">
                {item.keys.map((k) => (
                  <Kbd key={k}>{k}</Kbd>
                ))}
              </span>
            </li>
          ))}
        </ul>
      </div>
    ))}
  </div>
);

export const ShortcutsDialog: React.FC<{ open: boolean; onClose: () => void }> = ({ open, onClose }) => (
  <Dialog open={open} onClose={onClose} labelledBy="shortcuts-title" className="max-w-3xl">
    <div className="flex items-center justify-between px-6 pt-5 pb-3">
      <h2 id="shortcuts-title" className="flex items-center gap-2 font-nav text-[0.9375rem] font-semibold text-ink">
        <Keyboard className="w-4 h-4 text-accent-ink" aria-hidden />
        Keyboard shortcuts
      </h2>
      <IconButton size="sm" icon={X} label="Close keyboard shortcuts" onClick={onClose} />
    </div>
    <div className="px-6 pb-6">
      <ShortcutList />
    </div>
  </Dialog>
);
