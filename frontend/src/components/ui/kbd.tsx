import React from "react";




import { cx } from "./utils";

/* -------------------------------------------------------------------------- */
/* Keyboard hint                                                              */
/* -------------------------------------------------------------------------- */

export type KbdTone = "default" | "inverse" | "success" | "danger" | "interactive";

const kbdSizes = {
  xs: "h-4.5 min-w-4.5 px-0.5 text-[0.625rem]",
  sm: "h-5 min-w-5 px-1 text-micro",
  md: "h-6 min-w-6 px-1.5 text-meta",
};

const kbdTones: Record<KbdTone, string> = {
  default: "bg-surface border-line text-ink-muted",
  inverse: "bg-transparent border-current/35 text-current opacity-85",
  success: "bg-surface border-success-line text-success-ink",
  danger: "bg-surface border-danger-line text-danger-ink",
  interactive:
    "bg-surface border-line text-ink-muted group-hover:border-accent-line group-hover:text-accent-ink transition-colors",
};

export const Kbd: React.FC<React.HTMLAttributes<HTMLElement> & { size?: keyof typeof kbdSizes; tone?: KbdTone }> = ({
  children,
  className,
  size = "sm",
  tone = "default",
  ...rest
}) => (
  <kbd
    className={cx(
      "inline-flex items-center justify-center rounded-[5px] border border-b-2 font-data font-medium leading-none",
      kbdSizes[size],
      kbdTones[tone],
      className
    )}
    {...rest}
  >
    {children}
  </kbd>
);

