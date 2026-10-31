import React from "react";

import type { LucideIcon } from "lucide-react";
import { Loader2 } from "lucide-react";

import { cx } from "./utils";

/* -------------------------------------------------------------------------- */
/* Feedback states                                                            */
/* -------------------------------------------------------------------------- */

export const Spinner: React.FC<{ className?: string; label?: string }> = ({ className, label = "Loading" }) => (
  <Loader2 className={cx("w-4 h-4 animate-spin text-accent-ink", className)} aria-label={label} role="status" />
);

export const Skeleton: React.FC<{ className?: string }> = ({ className }) => (
  <div className={cx("rounded-md bg-sunken animate-shimmer", className)} aria-hidden />
);

export const LoadingBlock: React.FC<{ label: string; className?: string }> = ({ label, className }) => (
  <div
    role="status"
    aria-live="polite"
    className={cx("flex flex-col items-center justify-center gap-3 py-14 text-ink-subtle", className)}
  >
    <Loader2 className="w-5 h-5 animate-spin text-accent-ink" aria-hidden />
    <span className="text-meta">{label}</span>
  </div>
);

export const EmptyState: React.FC<{
  icon?: LucideIcon;
  title: string;
  description?: React.ReactNode;
  action?: React.ReactNode;
  className?: string;
}> = ({ icon: Icon, title, description, action, className }) => (
  <div
    className={cx(
      "flex flex-col items-center text-center gap-4 px-6 py-12 rounded-card bg-sunken/60",
      className
    )}
  >
    {Icon && (
      <div className="w-10 h-10 rounded-full bg-sunken border border-line flex items-center justify-center">
        <Icon className="w-4.5 h-4.5 text-ink-subtle" aria-hidden />
      </div>
    )}
    <div className="space-y-1 max-w-md">
      <h3 className="font-nav text-ui font-semibold text-ink">{title}</h3>
      {description && <p className="font-read text-read text-ink-muted leading-relaxed">{description}</p>}
    </div>
    {action}
  </div>
);

export const StatTile: React.FC<{
  label: string;
  value: React.ReactNode;
  icon?: LucideIcon;
  hint?: string;
}> = ({ label, value, icon: Icon, hint }) => (
  <div className="border-l-2 border-line pl-3 py-1 space-y-2 min-w-0" title={hint}>
    <div className="flex items-center gap-1.5 eyebrow text-ink-subtle">
      {Icon && <Icon className="w-3.5 h-3.5" aria-hidden />}
      <span>{label}</span>
    </div>
    <p className="font-data text-[1.375rem] leading-none font-semibold tracking-[-0.02em] text-ink tabular-nums">
      {value}
    </p>
  </div>
);

