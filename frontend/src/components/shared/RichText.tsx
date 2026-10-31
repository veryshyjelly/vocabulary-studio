import React from "react";

import type { RichSpan } from "../../lib/bridge";

import { cx } from "../ui";

/**
 * RichText safely renders structured RichSpan[] without ever using dangerouslySetInnerHTML.
 * `emphasis="word"` renders bold spans (the target vocabulary word) in the word typeface.
 */
export const RichText: React.FC<{
  spans?: RichSpan[] | null;
  fallback?: string;
  className?: string;
  emphasis?: "default" | "word" | "mark";
}> = ({ spans, fallback, className, emphasis = "default" }) => {
  if (!spans || spans.length === 0) {
    return <span className={className}>{fallback || ""}</span>;
  }
  const boldClass =
    emphasis === "word"
      ? "font-word font-semibold text-ink"
      : emphasis === "mark"
      ? "word-mark"
      : "font-semibold text-ink";
  return (
    <span className={className}>
      {spans.map((sp, i) => {
        if (sp.bold) {
          return (
            <strong key={i} className={cx(boldClass, sp.italic && "italic")}>
              {sp.text}
            </strong>
          );
        }
        if (sp.italic) {
          return (
            <em key={i} className="italic">
              {sp.text}
            </em>
          );
        }
        return <React.Fragment key={i}>{sp.text}</React.Fragment>;
      })}
    </span>
  );
};

