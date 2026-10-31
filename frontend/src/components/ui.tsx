import React, { createContext, useCallback, useContext, useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import type { LucideIcon } from "lucide-react";
import { AlertCircle, CheckCircle2, ChevronDown, Info, Loader2, X } from "lucide-react";

export function cx(...parts: Array<string | false | null | undefined>): string {
  return parts.filter(Boolean).join(" ");
}

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

/* -------------------------------------------------------------------------- */
/* Form controls                                                              */
/* -------------------------------------------------------------------------- */

const controlBase =
  "w-full rounded-control border border-line bg-surface text-ink placeholder:text-ink-subtle " +
  "transition-[border-color,box-shadow,background-color] duration-150 ease-out-soft " +
  "hover:border-line-strong focus:border-accent focus:ring-3 focus:ring-focus/20 outline-none " +
  "disabled:opacity-55 disabled:cursor-not-allowed";

export const TextInput = React.forwardRef<
  HTMLInputElement,
  React.InputHTMLAttributes<HTMLInputElement> & { icon?: LucideIcon; inputSize?: "sm" | "md" | "lg"; trailing?: React.ReactNode }
>(function TextInput({ icon: Icon, inputSize = "md", trailing, className, ...rest }, ref) {
  const heights = { sm: "h-8 text-meta", md: "h-9 text-ui", lg: "h-11 text-[0.875rem]" };
  return (
    <div className="relative w-full">
      {Icon && (
        <Icon
          className="w-4 h-4 text-ink-subtle absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none"
          aria-hidden
        />
      )}
      <input
        ref={ref}
        className={cx(controlBase, heights[inputSize], Icon ? "pl-9" : "pl-3", trailing ? "pr-28" : "pr-3", className)}
        {...rest}
      />
      {trailing && (
        <div className="absolute right-1.5 top-1/2 -translate-y-1/2 flex items-center gap-1.5">{trailing}</div>
      )}
    </div>
  );
});

export const TextArea = React.forwardRef<HTMLTextAreaElement, React.TextareaHTMLAttributes<HTMLTextAreaElement>>(
  function TextArea({ className, ...rest }, ref) {
    return <textarea ref={ref} className={cx(controlBase, "px-3 py-2.5 text-ui leading-relaxed", className)} {...rest} />;
  }
);

export const Select = React.forwardRef<HTMLSelectElement, React.SelectHTMLAttributes<HTMLSelectElement>>(
  function Select({ className, children, ...rest }, ref) {
    return (
      <div className="relative w-full">
        <select
          ref={ref}
          className={cx(controlBase, "h-9 pl-3 pr-9 text-ui appearance-none", className)}
          {...rest}
        >
          {children}
        </select>
        <ChevronDown
          className="w-4 h-4 text-ink-subtle absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none"
          aria-hidden
        />
      </div>
    );
  }
);

export const Field: React.FC<{
  label: string;
  htmlFor?: string;
  hint?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
}> = ({ label, htmlFor, hint, children, className }) => (
  <div className={cx("space-y-1.5", className)}>
    <label htmlFor={htmlFor} className="block font-nav text-meta font-medium text-ink-muted">
      {label}
    </label>
    {children}
    {hint && <p className="text-micro text-ink-subtle">{hint}</p>}
  </div>
);

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

/* -------------------------------------------------------------------------- */
/* Toasts                                                                     */
/* -------------------------------------------------------------------------- */

type ToastTone = "success" | "info" | "error";
interface ToastItem {
  id: number;
  tone: ToastTone;
  message: string;
}

const ToastContext = createContext<(message: string, tone?: ToastTone) => void>(() => {});

export function useToast() {
  return useContext(ToastContext);
}

export const ToastProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const nextId = useRef(1);

  const dismiss = useCallback((id: number) => {
    setToasts((list) => list.filter((t) => t.id !== id));
  }, []);

  const push = useCallback(
    (message: string, tone: ToastTone = "success") => {
      const id = nextId.current++;
      setToasts((list) => [...list.slice(-2), { id, tone, message }]);
      window.setTimeout(() => dismiss(id), tone === "error" ? 6000 : 3600);
    },
    [dismiss]
  );

  return (
    <ToastContext.Provider value={push}>
      {children}
      <div
        aria-live="polite"
        role="status"
        className="pointer-events-none fixed bottom-5 right-5 z-[60] flex flex-col items-end gap-2"
      >
        {toasts.map((t) => {
          const Icon = t.tone === "error" ? AlertCircle : t.tone === "info" ? Info : CheckCircle2;
          return (
            <div
              key={t.id}
              className="pointer-events-auto animate-toast flex items-center gap-2.5 max-w-sm pl-3.5 pr-2 py-2.5 rounded-[0.7rem] border border-line bg-raised shadow-pop text-ui text-ink"
            >
              <Icon
                className={cx(
                  "w-4 h-4 shrink-0",
                  t.tone === "error" ? "text-danger" : t.tone === "info" ? "text-ink-muted" : "text-success"
                )}
                aria-hidden
              />
              <span className="min-w-0">{t.message}</span>
              <button
                type="button"
                onClick={() => dismiss(t.id)}
                aria-label="Dismiss notification"
                className="ml-1 p-1 rounded-md text-ink-subtle hover:text-ink hover:bg-sunken"
              >
                <X className="w-3.5 h-3.5" aria-hidden />
              </button>
            </div>
          );
        })}
      </div>
    </ToastContext.Provider>
  );
};

/* -------------------------------------------------------------------------- */
/* Dialog                                                                     */
/* -------------------------------------------------------------------------- */

const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

export const Dialog: React.FC<{
  open: boolean;
  onClose: () => void;
  labelledBy: string;
  children: React.ReactNode;
  className?: string;
}> = ({ open, onClose, labelledBy, children, className }) => {
  const panelRef = useRef<HTMLDivElement | null>(null);
  const openerRef = useRef<HTMLElement | null>(null);
  const wasOpen = useRef(false);
  const closeRef = useRef(onClose);
  closeRef.current = onClose;
  // Captured during render: child autoFocus moves focus before any effect runs.
  if (open && !wasOpen.current && typeof document !== "undefined") {
    openerRef.current = document.activeElement as HTMLElement | null;
  }
  wasOpen.current = open;

  useEffect(() => {
    if (!open) return;
    const panel = panelRef.current;
    const previouslyFocused = openerRef.current;
    const appRoot = document.getElementById("root");
    const wasInert = appRoot?.inert ?? false;
    if (appRoot) appRoot.inert = true;
    if (panel && !panel.contains(document.activeElement)) {
      (panel.querySelector<HTMLElement>("[data-autofocus]") || panel.querySelector<HTMLElement>(FOCUSABLE) || panel)?.focus();
    }

    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        e.stopPropagation();
        closeRef.current();
        return;
      }
      if (e.key !== "Tab" || !panel) return;
      const nodes = Array.from(panel.querySelectorAll<HTMLElement>(FOCUSABLE));
      if (nodes.length === 0) {
        e.preventDefault();
        panel.focus();
        return;
      }
      const first = nodes[0];
      const last = nodes[nodes.length - 1];
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault();
        last.focus();
      } else if (!panel.contains(document.activeElement) || (!e.shiftKey && document.activeElement === last)) {
        e.preventDefault();
        first.focus();
      }
    };
    document.addEventListener("keydown", onKeyDown, true);
    return () => {
      document.removeEventListener("keydown", onKeyDown, true);
      if (appRoot) appRoot.inert = wasInert;
      if (previouslyFocused?.isConnected) previouslyFocused.focus();
    };
  }, [open]);

  if (!open) return null;

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-overlay backdrop-blur-[2px] animate-fade" onClick={onClose} aria-hidden />
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={labelledBy}
        tabIndex={-1}
        className={cx(
          "relative w-full max-w-md max-h-[calc(100dvh-2rem)] overflow-y-auto rounded-[1rem] border border-line bg-raised shadow-pop animate-dialog",
          className
        )}
      >
        {children}
      </div>
    </div>,
    document.body
  );
};
