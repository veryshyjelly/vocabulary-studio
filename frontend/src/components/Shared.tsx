import React, { useEffect, useRef, useState } from "react";
import {
  Volume2,
  Loader2,
  AlertCircle,
  Check,
  Copy,
  Eye,
  EyeOff,
  Lock,
  ShieldCheck,
  HardDrive,
  Cpu,
  X,
  VolumeX,
} from "lucide-react";
import type { RichSpan, AccountStatusDTO } from "../lib/bridge";
import { getBridge, formatError, SessionStoreMode } from "../lib/bridge";
import { Button, Dialog, IconButton, Kbd, cx } from "./ui";

export { Kbd };

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

/**
 * AudioButton plays a pronunciation MP3 URL or resolves the audio URL for a word on demand.
 */
export const AudioButton: React.FC<{
  audioUrl?: string;
  word?: string;
  label?: string;
  ariaLabel?: string;
  size?: "sm" | "md";
  variant?: "secondary" | "ghost" | "soft";
  className?: string;
}> = ({ audioUrl, word, label, ariaLabel, size = "md", variant = "secondary", className = "" }) => {
  const [playing, setPlaying] = useState(false);
  const [failed, setFailed] = useState(false);
  const audioRef = useRef<HTMLAudioElement | null>(null);

  useEffect(
    () => () => {
      audioRef.current?.pause();
    },
    []
  );

  const handlePlay = async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (playing) return;
    setFailed(false);
    setPlaying(true);

    try {
      let urlToPlay = audioUrl;
      if (!urlToPlay && word) {
        urlToPlay = await getBridge().dictionary.ResolveAudioURL(word);
      }
      if (!urlToPlay) {
        setFailed(true);
        setPlaying(false);
        return;
      }
      audioRef.current?.pause();
      const audio = new Audio(urlToPlay);
      audioRef.current = audio;
      audio.onended = () => setPlaying(false);
      audio.onerror = () => {
        setFailed(true);
        setPlaying(false);
      };
      await audio.play();
    } catch {
      setFailed(true);
      setPlaying(false);
    }
  };

  const Icon = playing ? Loader2 : failed ? VolumeX : Volume2;
  const accessible = `Play pronunciation${word ? ` for ${word}` : ""}`;

  return (
    <button
      type="button"
      onClick={handlePlay}
      title={failed ? "Audio unavailable" : accessible}
      aria-label={`${ariaLabel || (label ? `${label}${word ? ` (${word})` : ""}` : accessible)}${failed ? "; audio unavailable, try again" : ""}`}
      aria-busy={playing || undefined}
      className={cx(
        "inline-flex items-center justify-center gap-1.5 shrink-0 rounded-control font-nav font-medium",
        "transition-[background-color,border-color,color,transform] duration-150 ease-out-soft active:translate-y-px",
        size === "sm" ? (label ? "h-7 px-2.5 text-meta" : "h-7 w-7") : label ? "h-8.5 px-3 text-ui" : "h-8.5 w-8.5",
        failed
          ? "border border-warning-line bg-warning-soft text-warning-ink"
          : playing
          ? "border border-accent-line bg-accent-soft text-accent-ink"
          : variant === "ghost"
          ? "text-ink-muted hover:text-accent-ink hover:bg-sunken"
          : variant === "soft"
          ? "bg-accent-soft text-accent-ink hover:bg-accent-soft-hover"
          : "border border-line bg-surface text-ink hover:border-line-strong hover:bg-sunken",
        className
      )}
    >
      <Icon className={cx("w-3.5 h-3.5", playing && "animate-spin", !failed && !playing && "text-accent-ink")} aria-hidden />
      {label && <span>{label}</span>}
    </button>
  );
};

/**
 * CopyButton copies text to the clipboard with a temporary checkmark confirmation.
 */
export const CopyButton: React.FC<{ text: string; label?: string; size?: "sm" | "md" }> = ({
  text,
  label,
  size = "md",
}) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 1800);
    } catch {
      // Clipboard unavailable
    }
  };

  return (
    <Button size={size} variant="ghost" icon={copied ? Check : Copy} onClick={handleCopy} aria-live="polite">
      {copied ? "Copied" : label || "Copy"}
    </Button>
  );
};

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
