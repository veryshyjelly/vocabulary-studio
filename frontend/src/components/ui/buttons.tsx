import React from "react";

import type { LucideIcon } from "lucide-react";
import { Loader2 } from "lucide-react";

import { cx } from "./utils";
import { Kbd } from "./kbd";

/* -------------------------------------------------------------------------- */
/* Buttons                                                                    */
/* -------------------------------------------------------------------------- */

export type ButtonVariant = "primary" | "secondary" | "ghost" | "soft" | "danger" | "danger-outline" | "plain";
export type ButtonSize = "sm" | "md" | "lg" | "xl";

const buttonBase =
  "relative inline-flex items-center justify-center gap-1.5 shrink-0 whitespace-nowrap rounded-control font-nav font-medium select-none " +
  "transition-[background-color,border-color,color,box-shadow,transform,opacity] duration-150 ease-out-soft " +
  "active:translate-y-px disabled:opacity-45 disabled:pointer-events-none";

const buttonVariants: Record<ButtonVariant, string> = {
  primary:
    "bg-accent text-on-accent hover:bg-accent-hover shadow-[inset_0_1px_0_oklch(1_0_0/0.14),0_1px_2px_oklch(0.2_0.02_160/0.2)]",
  secondary:
    "bg-surface text-ink border border-line hover:border-line-strong hover:bg-sunken shadow-card",
  ghost: "text-ink-muted hover:text-ink hover:bg-sunken",
  soft: "bg-accent-soft text-accent-ink hover:bg-accent-soft-hover",
  danger: "bg-danger text-on-danger hover:bg-danger-hover",
  "danger-outline": "text-danger-ink border border-danger-line hover:bg-danger-soft",
  plain: "",
};

const buttonSizes: Record<ButtonSize, string> = {
  sm: "h-7 px-2.5 text-meta",
  md: "h-8.5 px-3.5 text-ui",
  lg: "h-10 px-5 text-ui",
  xl: "h-12 px-6 text-[0.9375rem]",
};

const iconSizes: Record<ButtonSize, string> = {
  sm: "w-3.5 h-3.5",
  md: "w-4 h-4",
  lg: "w-4 h-4",
  xl: "w-4.5 h-4.5",
};

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  icon?: LucideIcon;
  iconRight?: LucideIcon;
  loading?: boolean;
  kbd?: string;
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  {
    variant = "secondary",
    size = "md",
    icon: Icon,
    iconRight: IconRight,
    loading = false,
    kbd,
    className,
    children,
    type = "button",
    disabled,
    ...rest
  },
  ref
) {
  return (
    <button
      ref={ref}
      type={type}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      className={cx(buttonBase, buttonVariants[variant], buttonSizes[size], className)}
      {...rest}
    >
      {loading ? (
        <Loader2 className={cx(iconSizes[size], "animate-spin")} aria-hidden />
      ) : Icon ? (
        <Icon className={iconSizes[size]} aria-hidden />
      ) : null}
      {children}
      {IconRight && <IconRight className={iconSizes[size]} aria-hidden />}
      {kbd && (
        <Kbd aria-hidden className="ml-1" tone={variant === "primary" || variant === "danger" ? "inverse" : "default"}>
          {kbd}
        </Kbd>
      )}
    </button>
  );
});

export const IconButton = React.forwardRef<
  HTMLButtonElement,
  Omit<ButtonProps, "children" | "iconRight" | "kbd"> & { icon: LucideIcon; label: string }
>(function IconButton({ icon: Icon, label, size = "md", variant = "ghost", className, loading, disabled, ...rest }, ref) {
  const dims = size === "sm" ? "h-7 w-7" : size === "lg" ? "h-10 w-10" : size === "xl" ? "h-12 w-12" : "h-8.5 w-8.5";
  return (
    <button
      ref={ref}
      type="button"
      aria-label={label}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      title={label}
      className={cx(buttonBase, buttonVariants[variant], dims, "px-0", className)}
      {...rest}
    >
      {loading ? (
        <Loader2 className={cx(iconSizes[size], "animate-spin")} aria-hidden />
      ) : (
        <Icon className={iconSizes[size]} aria-hidden />
      )}
    </button>
  );
});

