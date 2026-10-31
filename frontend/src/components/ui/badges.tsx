import React from "react";

import type { LucideIcon } from "lucide-react";


import { cx } from "./utils";

/* -------------------------------------------------------------------------- */
/* Chips & badges                                                             */
/* -------------------------------------------------------------------------- */

export type Tone = "neutral" | "accent" | "success" | "warning" | "danger";

const badgeTones: Record<Tone, string> = {
  neutral: "bg-sunken text-ink-muted border-line",
  accent: "bg-accent-soft text-accent-ink border-accent-line/50",
  success: "bg-success-soft text-success-ink border-success-line/60",
  warning: "bg-warning-soft text-warning-ink border-warning-line/60",
  danger: "bg-danger-soft text-danger-ink border-danger-line/60",
};

export const Badge: React.FC<{
  tone?: Tone;
  icon?: LucideIcon;
  className?: string;
  children: React.ReactNode;
  title?: string;
  font?: "nav" | "data";
}> = ({ tone = "neutral", icon: Icon, className, children, title, font = "nav" }) => (
  <span
    title={title}
    className={cx(
      "inline-flex items-center gap-1 h-5.5 px-2 rounded-chip border text-micro font-medium whitespace-nowrap",
      font === "data" ? "font-data tabular-nums" : "font-nav",
      badgeTones[tone],
      className
    )}
  >
    {Icon && <Icon className="w-3 h-3" aria-hidden />}
    {children}
  </span>
);

export const Chip: React.FC<
  React.ButtonHTMLAttributes<HTMLButtonElement> & { active?: boolean; size?: "sm" | "md"; word?: boolean }
> = ({ active, size = "md", word = false, className, children, type = "button", ...rest }) => (
  <button
    type={type}
    aria-pressed={active}
    className={cx(
      "inline-flex items-center gap-1.5 rounded-full border whitespace-nowrap",
      word ? "font-word" : "font-nav",
      "transition-[background-color,border-color,color] duration-150 ease-out-soft",
      size === "sm" ? "h-6 px-2.5 text-micro" : "h-7 px-3 text-meta",
      active
        ? "bg-accent-soft border-accent-line text-accent-ink font-medium"
        : "bg-surface border-line text-ink-muted hover:border-line-strong hover:text-ink",
      className
    )}
    {...rest}
  >
    {children}
  </button>
);

