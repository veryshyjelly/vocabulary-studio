import React, { createContext, useCallback, useContext, useRef, useState } from "react";


import { AlertCircle, CheckCircle2, Info, X } from "lucide-react";

import { cx } from "./utils";

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

