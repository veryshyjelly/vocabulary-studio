import { useEffect, useRef, useState } from "react";
import { getBridge, SessionStoreMode, type AccountStatusDTO } from "../../lib/bridge";
import { Button } from "../../components/ui";

type PersistenceScope = { active: boolean; version: number; polling: boolean; retrying: boolean };

/** Keeps successful sign-in visible while reporting whether its session was saved. */
export function SessionPersistenceNotice({ account }: { account: AccountStatusDTO | null }) {
  const [state, setState] = useState(account?.persistence_state || "unknown");
  const [retrying, setRetrying] = useState(false);
  const [retryError, setRetryError] = useState(false);
  const scopeRef = useRef<PersistenceScope | null>(null);
  const authenticated = Boolean(account?.authenticated);
  const mode = account?.session_store_mode;
  const initialState = account?.persistence_state;

  useEffect(() => {
    setState(initialState || "unknown");
    setRetryError(false);
    setRetrying(false);
    if (!authenticated || mode === SessionStoreMode.SessionStoreMemory) return;
    const scope: PersistenceScope = { active: true, version: 0, polling: false, retrying: false };
    // Cleanup retires this account/mode scope even if a local bridge call finishes later.
    scopeRef.current = scope;
    const poll = async () => {
      if (scope.polling || scope.retrying) return;
      scope.polling = true;
      const version = scope.version;
      try {
        const status = await getBridge().auth.GetPersistenceStatus();
        // Unknown status is not evidence that an earlier failed save recovered.
        if (scope.active && version === scope.version && status && (status.state === "saved" || status.state === "failed")) {
          setState(status.state);
          if (status.state === "saved") setRetryError(false);
        }
      } catch {
        // A status-read failure does not prove that saving succeeded or failed.
      } finally {
        scope.polling = false;
      }
    };
    void poll();
    const timer = window.setInterval(() => { void poll(); }, 5000);
    return () => {
      scope.active = false;
      scopeRef.current = null;
      window.clearInterval(timer);
    };
  }, [authenticated, account?.uid, mode, initialState]);

  const retry = async () => {
    const scope = scopeRef.current;
    if (!scope?.active || scope.retrying) return;
    // A poll begun before this retry must not overwrite the retry's newer result.
    scope.version++;
    scope.retrying = true;
    setRetrying(true);
    setRetryError(false);
    try {
      const status = await getBridge().auth.RetryPersistence();
      if (scope.active) {
        if (status && (status.state === "saved" || status.state === "failed")) setState(status.state);
        setRetryError(status?.state !== "saved");
      }
    } catch {
      if (scope.active) setRetryError(true);
    } finally {
      scope.retrying = false;
      if (scope.active) setRetrying(false);
    }
  };

  if (!authenticated || mode === SessionStoreMode.SessionStoreMemory || state !== "failed") return null;

  return (
    <div role="status" className="mx-auto mb-5 max-w-[1160px] rounded-card border border-warning-line bg-warning-soft p-4 text-warning-ink">
      <p className="font-nav text-ui font-semibold">You’re signed in, but your session could not be saved.</p>
      <p className="mt-1 text-meta">You can keep studying. You may need to sign in again after restarting.</p>
      {retryError && <p className="mt-2 text-meta">Your session still could not be saved. Please try again.</p>}
      <Button className="mt-3" onClick={() => { void retry(); }} loading={retrying}>Retry saving session</Button>
    </div>
  );
}
