import React from "react";
import { AlertCircle, X } from "lucide-react";


import { Button, IconButton } from "../ui";

/**
 * ErrorBanner presents network or API errors with an optional Retry action.
 */
export const ErrorBanner: React.FC<{
  message: string;
  onRetry?: () => void;
  onDismiss?: () => void;
}> = ({ message, onRetry, onDismiss }) => (
  <div
    role="alert"
    className="animate-enter flex items-start justify-between gap-3 px-4 py-3 rounded-[0.7rem] border border-danger-line/70 bg-danger-soft text-danger-ink"
  >
    <div className="flex items-start gap-2.5 min-w-0">
      <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" aria-hidden />
      <span className="text-ui">{message}</span>
    </div>
    <div className="flex items-center gap-1 shrink-0 -my-0.5">
      {onRetry && (
        <Button size="sm" variant="danger-outline" onClick={onRetry}>
          Retry
        </Button>
      )}
      {onDismiss && (
        <IconButton
          size="sm"
          icon={X}
          label="Dismiss error"
          onClick={onDismiss}
          variant="plain"
          className="text-danger-ink hover:bg-danger-line/30"
        />
      )}
    </div>
  </div>
);

