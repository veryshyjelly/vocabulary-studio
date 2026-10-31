import React from "react";

import type { LucideIcon } from "lucide-react";


import { cx } from "./utils";

/* -------------------------------------------------------------------------- */
/* Layout                                                                     */
/* -------------------------------------------------------------------------- */

export const Card: React.FC<
  React.HTMLAttributes<HTMLDivElement> & {
    pad?: "none" | "sm" | "md" | "lg";
    as?: "div" | "section" | "article" | "form";
    tone?: "default" | "sunken" | "accent";
  }
> = ({ pad = "md", as: As = "div", tone = "default", className, children, ...rest }) => {
  const pads = { none: "", sm: "p-4", md: "p-5", lg: "p-6 md:p-7" };
  const tones = {
    default: "bg-surface border-line shadow-card",
    sunken: "bg-sunken border-line",
    accent: "bg-accent-soft/60 border-accent-line/60",
  };
  return (
    <As className={cx("rounded-card border", tones[tone], pads[pad], className)} {...(rest as object)}>
      {children}
    </As>
  );
};

export const PageHeader: React.FC<{
  eyebrow?: React.ReactNode;
  title: React.ReactNode;
  description?: React.ReactNode;
  actions?: React.ReactNode;
  children?: React.ReactNode;
}> = ({ eyebrow, title, description, actions, children }) => (
  <header className="space-y-4 pb-5 mb-6 border-b border-line">
    <div className="flex flex-wrap items-end justify-between gap-x-6 gap-y-3">
      <div className="min-w-0 space-y-1.5">
        {eyebrow && <p className="eyebrow text-accent-ink">{eyebrow}</p>}
        <h1 className="font-word text-title font-semibold tracking-[-0.03em] text-ink break-words">
          {title}
        </h1>
        {description && <p className="text-ui leading-relaxed text-ink-muted max-w-[68ch]">{description}</p>}
      </div>
      {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
    </div>
    {children}
  </header>
);

export const SectionTitle: React.FC<{
  icon?: LucideIcon;
  children: React.ReactNode;
  action?: React.ReactNode;
  as?: "h2" | "h3";
  className?: string;
}> = ({ icon: Icon, children, action, as: As = "h2", className }) => (
  <div className={cx("flex items-center justify-between gap-3", className)}>
    <As className="flex items-center gap-2 eyebrow text-ink-subtle">
      {Icon && <Icon className="w-3.5 h-3.5 text-accent-ink" aria-hidden />}
      {children}
    </As>
    {action}
  </div>
);

