import { ErrorBanner } from "../../components/shared/ErrorBanner";
import React, { useEffect, useState } from "react";
import { Eye, EyeOff, Lock, ShieldCheck, HardDrive, Cpu, X } from "lucide-react";
import type { AccountStatusDTO } from "../../lib/bridge";
import { getBridge, formatError, SessionStoreMode } from "../../lib/bridge";
import { Button, Dialog, IconButton, Kbd, cx } from "../../components/ui";

const STORE_OPTIONS = [
  { id: "keyring", label: "OS Keyring", icon: ShieldCheck, desc: "System keychain" },
  { id: "file", label: "Secure File", icon: HardDrive, desc: "Private local file" },
  { id: "memory", label: "Memory Only", icon: Cpu, desc: "This session only" },
];

/**
 * AuthModal provides an accessible Sign-In dialog that authenticates via the Go backend
 * and never stores credentials or tokens in browser storage.
 */
export const AuthModal: React.FC<{
  isOpen: boolean;
  initialStoreMode?: string;
  onClose: () => void;
  onSuccess: (status: AccountStatusDTO) => void;
}> = ({ isOpen, initialStoreMode = SessionStoreMode.SessionStoreKeyring, onClose, onSuccess }) => {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [storeMode, setStoreMode] = useState<string>(initialStoreMode || "keyring");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) setStoreMode(initialStoreMode || "keyring");
  }, [isOpen, initialStoreMode]);

  const handleClose = () => {
    setPassword("");
    setError(null);
    onClose();
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username.trim() || !password) {
      setError("Please enter both your email/username and password.");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const status = await getBridge().auth.Login(username.trim(), password, storeMode);
      if (status && status.authenticated) {
        setPassword("");
        onSuccess(status);
        onClose();
      } else {
        setError("Sign-in succeeded, but the session could not be verified.");
      }
    } catch (err) {
      setError(formatError(err));
    } finally {
      setLoading(false);
    }
  };

  const inputClass =
    "w-full h-10 px-3 rounded-control border border-line bg-surface text-ui text-ink placeholder:text-ink-subtle " +
    "hover:border-line-strong focus:border-accent focus:ring-3 focus:ring-focus/20 outline-none transition-[border-color,box-shadow]";

  return (
    <Dialog open={isOpen} onClose={handleClose} labelledBy="auth-modal-title">
      <div className="flex items-start justify-between gap-3 px-6 pt-5 pb-4">
        <div className="flex items-start gap-3">
          <div className="w-9 h-9 rounded-[0.6rem] bg-accent-soft border border-accent-line/50 flex items-center justify-center shrink-0">
            <Lock className="w-4 h-4 text-accent-ink" aria-hidden />
          </div>
          <div>
            <h2 id="auth-modal-title" className="font-nav text-[0.9375rem] font-semibold text-ink">
              Sign in to Vocabulary.com
            </h2>
            <p className="text-meta text-ink-muted mt-0.5">Sync your lists, trainer progress and mastery.</p>
          </div>
        </div>
        <IconButton size="sm" icon={X} label="Close sign in dialog" onClick={handleClose} />
      </div>

      <form onSubmit={handleSubmit} className="px-6 pb-6 space-y-4">
        {error && <ErrorBanner message={error} onDismiss={() => setError(null)} />}

        <div className="space-y-1.5">
          <label htmlFor="vocab-username" className="block font-nav text-meta font-medium text-ink-muted">
            Email or Username
          </label>
          <input
            id="vocab-username"
            type="text"
            autoFocus
            autoComplete="username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="you@example.com"
            className={inputClass}
          />
        </div>

        <div className="space-y-1.5">
          <label htmlFor="vocab-password" className="block font-nav text-meta font-medium text-ink-muted">
            Password
          </label>
          <div className="relative">
            <input
              id="vocab-password"
              type={showPassword ? "text" : "password"}
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              className={cx(inputClass, "pr-10")}
            />
            <button
              type="button"
              onClick={() => setShowPassword((v) => !v)}
              aria-label={showPassword ? "Hide password" : "Show password"}
              className="absolute right-1.5 top-1/2 -translate-y-1/2 h-7 w-7 inline-flex items-center justify-center rounded-md text-ink-subtle hover:text-ink hover:bg-sunken"
            >
              {showPassword ? <EyeOff className="w-4 h-4" aria-hidden /> : <Eye className="w-4 h-4" aria-hidden />}
            </button>
          </div>
        </div>

        <fieldset className="space-y-1.5">
          <legend className="font-nav text-meta font-medium text-ink-muted mb-1.5">Session Storage Security</legend>
          <div className="grid grid-cols-3 gap-2">
            {STORE_OPTIONS.map((opt) => {
              const Icon = opt.icon;
              const selected = storeMode === opt.id;
              return (
                <button
                  key={opt.id}
                  type="button"
                  aria-pressed={selected}
                  onClick={() => setStoreMode(opt.id)}
                  className={cx(
                    "flex flex-col items-start gap-0.5 p-2.5 rounded-control border text-left transition-[background-color,border-color] duration-150",
                    selected
                      ? "border-accent-line bg-accent-soft text-accent-ink"
                      : "border-line hover:border-line-strong hover:bg-sunken text-ink"
                  )}
                >
                  <Icon className="w-3.5 h-3.5 mb-1 text-accent-ink" aria-hidden />
                  <span className="font-nav text-meta font-medium leading-tight">{opt.label}</span>
                  <span className="text-micro text-ink-subtle">{opt.desc}</span>
                </button>
              );
            })}
          </div>
        </fieldset>

        <p className="text-meta text-ink-subtle leading-relaxed">
          Your sign-in is protected on this device. Passwords and session tokens are never saved in browser storage.
        </p>

        <div className="flex items-center justify-between gap-2 pt-1">
          <span className="hidden sm:inline-flex items-center gap-1.5 text-micro text-ink-subtle">
            <Kbd>Esc</Kbd> to close
          </span>
          <div className="flex items-center gap-2 ml-auto">
            <Button variant="ghost" onClick={handleClose}>
              Cancel
            </Button>
            <Button type="submit" variant="primary" loading={loading}>
              {loading ? "Signing in..." : "Sign In"}
            </Button>
          </div>
        </div>
      </form>
    </Dialog>
  );
};
