import React, { useState, useEffect, useCallback, useRef } from "react";
import {
  Home,
  Sparkles,
  BookOpen,
  ListChecks,
  Compass,
  TrendingUp,
  Settings,
  LogIn,
  Search,
  Sun,
  Moon,
  Keyboard,
} from "lucide-react";
import type { AccountStatusDTO, AppPreferences } from "./lib/bridge";
import { getBridge, SessionStoreMode } from "./lib/bridge";
import { AuthModal } from "./components/Shared";
import { ShortcutsDialog } from "./components/Shortcuts";
import { Kbd, ToastProvider, cx } from "./components/ui";
import { acceptsPlainShortcut, isModalOpen, modKeyLabel } from "./lib/keys";
import { HomeView } from "./views/HomeView";
import { PracticeView } from "./views/PracticeView";
import { DictionaryView } from "./views/DictionaryView";
import { ListsView } from "./views/ListsView";
import { DiscoverView } from "./views/DiscoverView";
import { ProgressView } from "./views/ProgressView";
import { SettingsView } from "./views/SettingsView";

export type NavSection =
  | "home"
  | "practice"
  | "dictionary"
  | "lists"
  | "discover"
  | "progress"
  | "settings";

const DEFAULT_PREFS: AppPreferences = {
  theme: "system",
  pronunciation_accent: "US",
  default_view: "home",
  session_store_mode: SessionStoreMode.SessionStoreKeyring,
  search_history: [],
  app_version: "1.0.0",
};

const NAV_ITEMS: { id: NavSection; label: string; icon: React.ElementType }[] = [
  { id: "home", label: "Home", icon: Home },
  { id: "practice", label: "Practice", icon: Sparkles },
  { id: "dictionary", label: "Dictionary", icon: BookOpen },
  { id: "lists", label: "My Lists", icon: ListChecks },
  { id: "discover", label: "Discover", icon: Compass },
  { id: "progress", label: "Progress", icon: TrendingUp },
  { id: "settings", label: "Settings", icon: Settings },
];

function systemPrefersDark(): boolean {
  return typeof window !== "undefined" && !!window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches;
}

export const App: React.FC = () => {
  const [activeNav, setActiveNav] = useState<NavSection>("home");
  const [account, setAccount] = useState<AccountStatusDTO | null>(null);
  const [preferences, setPreferences] = useState<AppPreferences>(DEFAULT_PREFS);
  const [authModalOpen, setAuthModalOpen] = useState(false);
  const [shortcutsOpen, setShortcutsOpen] = useState(false);
  const [isDark, setIsDark] = useState<boolean>(() =>
    typeof document !== "undefined" && document.documentElement.classList.contains("dark")
  );

  // Cross-view navigation parameters
  const [targetWord, setTargetWord] = useState<string>("callow");
  const [targetListId, setTargetListId] = useState<number>(0);
  const [practiceMode, setPracticeMode] = useState<"trainer" | "bee" | "quiz">("trainer");
  const [quickSearch, setQuickSearch] = useState("");
  const [searchFocusToken, setSearchFocusToken] = useState(0);
  const quickSearchRef = useRef<HTMLInputElement | null>(null);
  const mainRef = useRef<HTMLElement | null>(null);

  // Apply light/dark/system theme class on <html>, following OS changes in "system" mode.
  useEffect(() => {
    const root = document.documentElement;
    const apply = () => {
      const dark = preferences.theme === "dark" || (preferences.theme !== "light" && systemPrefersDark());
      root.classList.toggle("dark", dark);
      setIsDark(dark);
    };
    apply();
    if (preferences.theme !== "system" || !window.matchMedia) return;
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    mq.addEventListener?.("change", apply);
    return () => mq.removeEventListener?.("change", apply);
  }, [preferences.theme]);

  // Initial bootstrap: load preferences and restore persisted session
  useEffect(() => {
    let mounted = true;
    async function init() {
      try {
        const prefs = await getBridge().settings.GetPreferences();
        if (mounted && prefs) {
          setPreferences(prefs);
          if (
            prefs.default_view &&
            ["home", "practice", "dictionary", "lists", "discover", "progress"].includes(prefs.default_view)
          ) {
            setActiveNav(prefs.default_view as NavSection);
          }
        }
      } catch {
        // Use default preferences
      }

      try {
        const status = await getBridge().auth.RestoreSession();
        if (mounted && status) {
          setAccount(status);
        }
      } catch {
        // Unauthenticated fallback
      }
    }
    init();
    return () => {
      mounted = false;
    };
  }, []);

  const navigate = useCallback((section: NavSection) => {
    setActiveNav(section);
    mainRef.current?.scrollTo?.({ top: 0 });
  }, []);

  const handleOpenWord = useCallback(
    (word: string) => {
      setTargetWord(word);
      navigate("dictionary");
    },
    [navigate]
  );

  const handleOpenList = useCallback(
    (listId: number) => {
      setTargetListId(listId);
      navigate("lists");
    },
    [navigate]
  );

  const handleStartPractice = useCallback(
    (listId = 0) => {
      setTargetListId(listId);
      setPracticeMode("trainer");
      navigate("practice");
    },
    [navigate]
  );

  const handleStartBee = useCallback(
    (listId: number) => {
      setTargetListId(listId);
      setPracticeMode("bee");
      navigate("practice");
    },
    [navigate]
  );

  const focusLookup = useCallback(() => {
    const input = quickSearchRef.current;
    if (input && input.offsetParent !== null) {
      input.focus();
      input.select();
      return;
    }
    navigate("dictionary");
    setSearchFocusToken((t) => t + 1);
  }, [navigate]);

  // Global shortcuts: Mod+K lookup, Mod+1..7 sections, Mod+, settings, ? help.
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.defaultPrevented || e.isComposing || e.repeat) return;
      const mod = e.metaKey || e.ctrlKey;
      if (mod && !e.altKey && !e.shiftKey && !isModalOpen()) {
        if (e.key.toLowerCase() === "k") {
          e.preventDefault();
          focusLookup();
          return;
        }
        if (e.key === ",") {
          e.preventDefault();
          navigate("settings");
          return;
        }
        const idx = Number(e.key);
        if (Number.isInteger(idx) && idx >= 1 && idx <= NAV_ITEMS.length) {
          e.preventDefault();
          navigate(NAV_ITEMS[idx - 1].id);
          return;
        }
      }
      if (e.key === "?" && acceptsPlainShortcut(e)) {
        e.preventDefault();
        setShortcutsOpen(true);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [focusLookup, navigate]);

  const handleQuickSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!quickSearch.trim()) return;
    handleOpenWord(quickSearch.trim());
    setQuickSearch("");
    quickSearchRef.current?.blur();
  };

  const toggleQuickTheme = async () => {
    const nextTheme = isDark ? "light" : "dark";
    setPreferences((p) => ({ ...p, theme: nextTheme }));
    try {
      const updated = await getBridge().settings.UpdatePreferences({
        ...preferences,
        theme: nextTheme,
      });
      if (updated) setPreferences(updated);
    } catch {
      // ignore
    }
  };

  const nickname = account?.nickname || "Learner";

  return (
    <ToastProvider>
      <div className="h-screen flex overflow-hidden bg-canvas text-ink">
        <a
          href="#main"
          className="sr-only focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-[70] focus:px-3 focus:py-2 focus:rounded-control focus:bg-raised focus:shadow-pop focus:text-ui"
        >
          Skip to content
        </a>

        {/* Navigation sidebar: icon rail below the "wide" breakpoint */}
        <aside className="w-24 wide:w-56 shrink-0 border-r border-line bg-sunken/70 flex flex-col select-none transition-[width] duration-200 ease-out-soft">
          <div className="flex flex-col gap-4 wide:gap-5 px-2 wide:px-3 pt-4 wide:pt-5 pb-3 min-h-0 flex-1 overflow-y-auto">
            {/* Brand */}
            <div className="flex shrink-0 items-center justify-center wide:justify-between gap-2 wide:px-1.5">
              <button
                type="button"
                onClick={() => navigate("home")}
                className="flex items-center gap-2.5 text-left rounded-control"
                aria-label="Vocabulary Studio home"
              >
                <span className="w-8 h-8 rounded-[0.55rem] bg-accent text-on-accent flex items-center justify-center font-word font-bold text-[0.95rem] shadow-[inset_0_1px_0_oklch(1_0_0/0.2)]">
                  V
                </span>
                <span className="hidden wide:block leading-none">
                  <span className="block font-word font-semibold text-[0.9375rem] text-ink">Vocabulary</span>
                  <span className="block eyebrow text-accent-ink mt-1">Studio</span>
                </span>
              </button>
              <button
                type="button"
                onClick={toggleQuickTheme}
                title={isDark ? "Switch to light theme" : "Switch to dark theme"}
                aria-label="Toggle theme"
                className="hidden wide:inline-flex h-7 w-7 items-center justify-center rounded-md text-ink-subtle hover:text-ink hover:bg-surface transition-colors"
              >
                {isDark ? <Sun className="w-4 h-4" aria-hidden /> : <Moon className="w-4 h-4" aria-hidden />}
              </button>
            </div>

            {/* Quick lookup (full sidebar) / search button (rail) */}
            <form onSubmit={handleQuickSearchSubmit} className="relative shrink-0 hidden wide:block" role="search">
              <Search
                className="w-3.5 h-3.5 text-ink-subtle absolute left-2.5 top-1/2 -translate-y-1/2 pointer-events-none"
                aria-hidden
              />
              <input
                ref={quickSearchRef}
                type="text"
                value={quickSearch}
                onChange={(e) => setQuickSearch(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Escape") {
                    setQuickSearch("");
                    e.currentTarget.blur();
                  }
                }}
                placeholder="Look up a word"
                aria-label="Quick word lookup"
                className="w-full h-8 pl-8 pr-12 rounded-control border border-line bg-surface text-meta text-ink placeholder:text-ink-subtle hover:border-line-strong focus:border-accent focus:ring-3 focus:ring-focus/20 outline-none transition-[border-color,box-shadow]"
              />
              <span className="absolute right-1.5 top-1/2 -translate-y-1/2 flex items-center gap-0.5 pointer-events-none" aria-hidden>
                <Kbd size="xs">{modKeyLabel}</Kbd>
                <Kbd size="xs">K</Kbd>
              </span>
            </form>
            <button
              type="button"
              onClick={focusLookup}
              aria-label="Look up a word"
              title={`Look up a word (${modKeyLabel}K)`}
              className="wide:hidden shrink-0 mx-auto h-9 w-9 inline-flex items-center justify-center rounded-control border border-line bg-surface text-ink-muted hover:text-ink hover:border-line-strong transition-colors"
            >
              <Search className="w-4 h-4" aria-hidden />
            </button>

            {/* Sections */}
            <nav aria-label="Main navigation" className="shrink-0 flex flex-col gap-0.5">
              {NAV_ITEMS.map((item, idx) => {
                const Icon = item.icon;
                const active = activeNav === item.id;
                return (
                  <button
                    key={item.id}
                    type="button"
                    aria-label={item.label}
                    aria-current={active ? "page" : undefined}
                    title={`${item.label} (${modKeyLabel}${idx + 1})`}
                    onClick={() => navigate(item.id)}
                    className={cx(
                      "group relative shrink-0 flex flex-col wide:flex-row items-center justify-center wide:justify-start gap-1 wide:gap-3 h-12 wide:h-10 wide:px-2.5 rounded-control font-nav text-ui",
                      "transition-[background-color,color,box-shadow] duration-150 ease-out-soft",
                      active
                        ? "bg-accent-soft text-accent-ink font-semibold"
                        : "text-ink-muted hover:text-ink hover:bg-surface/80"
                    )}
                  >
                    <span
                      aria-hidden
                      className={cx(
                        "absolute left-0 top-2 bottom-2 w-[3px] rounded-r-full bg-accent transition-opacity duration-150",
                        active ? "opacity-100" : "opacity-0"
                      )}
                    />
                    <Icon
                      className={cx("w-4 h-4 shrink-0", active ? "text-accent-ink" : "text-ink-subtle group-hover:text-ink-muted")}
                      aria-hidden
                    />
                    <span className="text-[0.625rem] wide:text-ui wide:flex-1 wide:text-left">{item.label}</span>
                    <span
                      aria-hidden
                      className="hidden wide:inline font-data text-micro text-ink-subtle opacity-0 group-hover:opacity-100 transition-opacity"
                    >
                      {modKeyLabel}
                      {idx + 1}
                    </span>
                  </button>
                );
              })}
            </nav>
          </div>

          {/* Footer: shortcuts, theme (rail) and account */}
          <div className="px-2.5 wide:px-3.5 pb-3.5 pt-3 space-y-2 border-t border-line">
            <div className="flex flex-col wide:flex-row items-center gap-1">
              <button
                type="button"
                onClick={() => setShortcutsOpen(true)}
                aria-label="Keyboard shortcuts"
                title="Keyboard shortcuts (?)"
                className="h-8 w-9 wide:w-auto wide:flex-1 inline-flex items-center justify-center wide:justify-start gap-2 wide:px-2.5 rounded-control text-meta text-ink-subtle hover:text-ink hover:bg-surface transition-colors"
              >
                <Keyboard className="w-4 h-4" aria-hidden />
                <span className="hidden wide:inline font-nav">Shortcuts</span>
                <Kbd className="hidden wide:inline-flex ml-auto">?</Kbd>
              </button>
              <button
                type="button"
                onClick={toggleQuickTheme}
                aria-label={isDark ? "Switch to light theme" : "Switch to dark theme"}
                title={isDark ? "Switch to light theme" : "Switch to dark theme"}
                className="wide:hidden h-8 w-9 inline-flex items-center justify-center rounded-control text-ink-subtle hover:text-ink hover:bg-surface transition-colors"
              >
                {isDark ? <Sun className="w-4 h-4" aria-hidden /> : <Moon className="w-4 h-4" aria-hidden />}
              </button>
            </div>

            {account?.authenticated ? (
              <button
                type="button"
                onClick={() => navigate("settings")}
                title={`${nickname} · account settings`}
                className="w-full flex items-center justify-center wide:justify-start gap-2.5 p-1.5 wide:p-2 rounded-control border border-transparent wide:border-line wide:bg-surface hover:border-line-strong transition-colors text-left"
              >
                <span className="w-7 h-7 rounded-full bg-accent-soft border border-accent-line/60 text-accent-ink flex items-center justify-center shrink-0 font-word font-semibold text-meta uppercase" aria-hidden>
                  {nickname.slice(0, 1)}
                </span>
                <span className="hidden wide:block min-w-0">
                  <span className="block font-nav text-meta font-semibold text-ink truncate">
                    {account.nickname || "Learner"}
                  </span>
                  <span className="block font-data text-micro text-ink-subtle truncate tabular-nums">
                    {account.level?.name || "Novice"} · {account.points.toLocaleString()} pts
                  </span>
                </span>
              </button>
            ) : (
              <button
                type="button"
                onClick={() => setAuthModalOpen(true)}
                aria-label="Sign In"
                title="Sign in to Vocabulary.com"
                className="w-full h-9 inline-flex items-center justify-center gap-2 rounded-control bg-accent text-on-accent hover:bg-accent-hover font-nav text-ui font-medium transition-colors"
              >
                <LogIn className="w-4 h-4" aria-hidden />
                <span className="hidden wide:inline">Sign In</span>
              </button>
            )}
          </div>
        </aside>

        {/* Main content */}
        <main id="main" ref={mainRef} tabIndex={-1} className="flex-1 min-w-0 overflow-y-auto focus:outline-none">
          <div key={activeNav} className="page-content animate-enter mx-auto w-full max-w-[1160px]">
            {activeNav === "home" && (
              <HomeView
                account={account}
                onStartPractice={handleStartPractice}
                onOpenWord={handleOpenWord}
                onOpenList={handleOpenList}
                onNavigate={(v) => navigate(v as NavSection)}
                onRequestLogin={() => setAuthModalOpen(true)}
              />
            )}

            {activeNav === "practice" && (
              <PracticeView
                account={account}
                initialListId={targetListId}
                initialMode={practiceMode}
                onOpenWord={handleOpenWord}
                onRequestLogin={() => setAuthModalOpen(true)}
              />
            )}

            {activeNav === "dictionary" && (
              <DictionaryView
                account={account}
                initialWord={targetWord}
                preferredAccent={preferences.pronunciation_accent}
                focusSearchToken={searchFocusToken}
                onOpenList={handleOpenList}
                onRequestLogin={() => setAuthModalOpen(true)}
              />
            )}

            {activeNav === "lists" && (
              <ListsView
                account={account}
                initialListId={targetListId}
                onStartPractice={handleStartPractice}
                onStartBee={handleStartBee}
                onOpenWord={handleOpenWord}
                onRequestLogin={() => setAuthModalOpen(true)}
              />
            )}

            {activeNav === "discover" && (
              <DiscoverView
                onOpenWord={handleOpenWord}
                onOpenList={handleOpenList}
                onStartPractice={handleStartPractice}
              />
            )}

            {activeNav === "progress" && (
              <ProgressView
                account={account}
                onOpenWord={handleOpenWord}
                onStartPractice={handleStartPractice}
                onRequestLogin={() => setAuthModalOpen(true)}
              />
            )}

            {activeNav === "settings" && (
              <SettingsView
                preferences={preferences}
                account={account}
                onPreferencesUpdated={setPreferences}
                onAccountChanged={setAccount}
                onRequestLogin={() => setAuthModalOpen(true)}
              />
            )}
          </div>
        </main>

        <AuthModal
          isOpen={authModalOpen}
          initialStoreMode={preferences.session_store_mode}
          onClose={() => setAuthModalOpen(false)}
          onSuccess={(status) => setAccount(status)}
        />
        <ShortcutsDialog open={shortcutsOpen} onClose={() => setShortcutsOpen(false)} />
      </div>
    </ToastProvider>
  );
};
