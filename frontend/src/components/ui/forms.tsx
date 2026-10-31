import React from "react";

import type { LucideIcon } from "lucide-react";
import { ChevronDown } from "lucide-react";

import { cx } from "./utils";

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

