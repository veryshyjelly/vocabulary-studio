import React from "react";




import { cx } from "./utils";

/* -------------------------------------------------------------------------- */
/* Meter                                                                      */
/* -------------------------------------------------------------------------- */

export const Meter: React.FC<{
  value: number;
  label: string;
  tone?: "accent" | "warning" | "success";
  className?: string;
  size?: "xs" | "sm" | "md";
  /** Purely visual (the value is already stated in adjacent text). */
  decorative?: boolean;
}> = ({ value, label, tone = "accent", className, size = "sm", decorative = false }) => {
  const pct = Math.max(0, Math.min(100, Math.round(value)));
  const fill = tone === "warning" ? "bg-warning" : tone === "success" ? "bg-success" : "bg-accent";
  const a11y = decorative
    ? { "aria-hidden": true as const }
    : { role: "progressbar", "aria-label": label, "aria-valuemin": 0, "aria-valuemax": 100, "aria-valuenow": pct };
  return (
    <div
      {...a11y}
      className={cx(
        "w-full rounded-full bg-sunken overflow-hidden",
        size === "xs" ? "h-1" : size === "sm" ? "h-1.5" : "h-2",
        className
      )}
    >
      <div
        className={cx("h-full rounded-full transition-[width] duration-500 ease-out-soft", fill)}
        style={{ width: `${pct}%` }}
      />
    </div>
  );
};

