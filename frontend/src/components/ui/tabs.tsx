import React, { useRef } from "react";

import type { LucideIcon } from "lucide-react";


import { cx } from "./utils";

/* -------------------------------------------------------------------------- */
/* Tabs (segmented control)                                                   */
/* -------------------------------------------------------------------------- */

export interface TabItem<T extends string> {
  id: T;
  label: string;
  icon?: LucideIcon;
  badge?: React.ReactNode;
}

export function Tabs<T extends string>({
  items,
  value,
  onChange,
  label,
  size = "md",
  className,
}: {
  items: TabItem<T>[];
  value: T;
  onChange: (id: T) => void;
  label: string;
  size?: "sm" | "md";
  className?: string;
}) {
  const refs = useRef<Array<HTMLButtonElement | null>>([]);

  const onKeyDown = (e: React.KeyboardEvent, idx: number) => {
    let next = -1;
    if (e.key === "ArrowRight") next = (idx + 1) % items.length;
    else if (e.key === "ArrowLeft") next = (idx - 1 + items.length) % items.length;
    else if (e.key === "Home") next = 0;
    else if (e.key === "End") next = items.length - 1;
    if (next >= 0) {
      e.preventDefault();
      e.stopPropagation();
      refs.current[next]?.focus();
      onChange(items[next].id);
    }
  };

  return (
    <div
      role="tablist"
      aria-label={label}
      className={cx("inline-flex max-w-full overflow-x-auto p-0.75 rounded-[0.625rem] bg-sunken border border-line", className)}
    >
      {items.map((item, idx) => {
        const Icon = item.icon;
        const active = item.id === value;
        return (
          <button
            key={item.id}
            ref={(el) => {
              refs.current[idx] = el;
            }}
            type="button"
            role="tab"
            aria-selected={active}
            tabIndex={active ? 0 : -1}
            onClick={() => onChange(item.id)}
            onKeyDown={(e) => onKeyDown(e, idx)}
            className={cx(
              "inline-flex items-center gap-1.5 whitespace-nowrap rounded-[0.4rem] font-nav font-medium",
              "transition-[background-color,color,box-shadow] duration-150 ease-out-soft",
              size === "sm" ? "h-6.5 px-2.5 text-meta" : "h-7.5 px-3 text-ui",
              active
                ? "bg-raised text-ink shadow-raised"
                : "text-ink-muted hover:text-ink"
            )}
          >
            {Icon && (
              <Icon
                className={cx("w-3.5 h-3.5", active ? "text-accent-ink" : "text-ink-subtle")}
                aria-hidden
              />
            )}
            <span>{item.label}</span>
            {item.badge}
          </button>
        );
      })}
    </div>
  );
}

