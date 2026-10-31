import React, { useState } from "react";
import {
  Sun,
  Moon,
  Monitor,
  Volume2,
  ShieldCheck,
  HardDrive,
  Cpu,
  Keyboard,
  UserCheck,
  LogOut,
  LogIn,
  Check,
} from "lucide-react";
import type { AppPreferences, AccountStatusDTO } from "../lib/bridge";
import { getBridge, formatError, SessionStoreMode } from "../lib/bridge";
import { Kbd, ErrorBanner } from "../components/Shared";

export const SettingsView: React.FC<{
  preferences: AppPreferences;
  account: AccountStatusDTO | null;
  onPreferencesUpdated: (prefs: AppPreferences) => void;
  onAccountChanged: (status: AccountStatusDTO) => void;
  onRequestLogin: () => void;
}> = ({
  preferences,
  account,
  onPreferencesUpdated,
  onAccountChanged,
  onRequestLogin,
}) => {
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [savedNotice, setSavedNotice] = useState<string | null>(null);

  const updatePref = async (patch: Partial<AppPreferences>) => {
    setSaving(true);
    setError(null);
    setSavedNotice(null);
    try {
      const merged: AppPreferences = {
        ...preferences,
        ...patch,
      };
      const updated = await getBridge().settings.UpdatePreferences(merged);
      if (updated) {
        onPreferencesUpdated(updated);
        setSavedNotice("Preferences saved.");
      }
    } catch (err) {
      setError(formatError(err));
    } finally {
      setSaving(false);
    }
  };

  const handleLogout = async () => {
    setSaving(true);
    setError(null);
    try {
      const st = await getBridge().auth.Logout();
      if (st) {
        onAccountChanged(st);
      }
    } catch (err) {
      setError(formatError(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="max-w-3xl mx-auto space-y-6 pb-12">
      <div className="pb-4 border-b border-stone-200/80 dark:border-stone-800/80">
        <h1 className="text-2xl font-editorial font-semibold text-stone-900 dark:text-stone-100">
          Settings & Preferences
        </h1>
        <p className="text-xs text-stone-500 dark:text-stone-400">
          Configure appearance, pronunciation accent, session storage security, and keyboard shortcuts.
        </p>
      </div>

      {error && <ErrorBanner message={error} onDismiss={() => setError(null)} />}
      {savedNotice && (
        <div className="flex items-center gap-2 px-4 py-2.5 rounded-lg bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-200 dark:border-emerald-900 text-xs text-emerald-900 dark:text-emerald-200">
          <Check className="w-4 h-4 text-emerald-600" />
          <span>{savedNotice}</span>
        </div>
      )}

      {/* Account Section */}
      <div className="rounded-xl border border-stone-200/90 dark:border-stone-800 bg-white dark:bg-stone-900 p-6 space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <UserCheck className="w-4 h-4 text-emerald-700 dark:text-emerald-400" />
            <h2 className="text-sm font-semibold text-stone-900 dark:text-stone-100">
              Vocabulary.com Account
            </h2>
          </div>
          {account?.authenticated ? (
            <button
              type="button"
              disabled={saving}
              onClick={handleLogout}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-rose-200 dark:border-rose-900/60 text-rose-700 dark:text-rose-300 text-xs font-medium hover:bg-rose-50 dark:hover:bg-rose-950/40"
            >
              <LogOut className="w-3.5 h-3.5" />
              <span>Sign Out</span>
            </button>
          ) : (
            <button
              type="button"
              onClick={onRequestLogin}
              className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg bg-emerald-700 text-white text-xs font-medium"
            >
              <LogIn className="w-3.5 h-3.5" />
              <span>Sign In</span>
            </button>
          )}
        </div>

        {account?.authenticated ? (
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
            <div className="p-3 rounded-lg bg-stone-50 dark:bg-stone-950">
              <span className="text-stone-400 block">Nickname</span>
              <strong className="text-stone-900 dark:text-stone-100">{account.nickname}</strong>
            </div>
            <div className="p-3 rounded-lg bg-stone-50 dark:bg-stone-950">
              <span className="text-stone-400 block">Level</span>
              <strong className="text-stone-900 dark:text-stone-100">
                {account.level?.name || "Learner"}
              </strong>
            </div>
            <div className="p-3 rounded-lg bg-stone-50 dark:bg-stone-950">
              <span className="text-stone-400 block">Points</span>
              <strong className="text-stone-900 dark:text-stone-100">
                {account.points.toLocaleString()}
              </strong>
            </div>
            <div className="p-3 rounded-lg bg-stone-50 dark:bg-stone-950">
              <span className="text-stone-400 block">VocabTrainer</span>
              <strong className="text-stone-900 dark:text-stone-100">
                {account.full_vocab_trainer ? "Full Subscriber" : "Standard"}
              </strong>
            </div>
          </div>
        ) : (
          <p className="text-xs text-stone-500">
            Not currently signed in. Dictionary lookup, corpus examples, and Word of the Day work without an account.
          </p>
        )}
      </div>

      {/* Appearance & Study Preferences */}
      <div className="rounded-xl border border-stone-200/90 dark:border-stone-800 bg-white dark:bg-stone-900 p-6 space-y-5">
        <h2 className="text-sm font-semibold text-stone-900 dark:text-stone-100">
          Appearance & Audio
        </h2>

        <div className="space-y-2">
          <label className="block text-xs font-medium text-stone-500">Color Theme</label>
          <div className="grid grid-cols-3 gap-2.5">
            {[
              { id: "light", label: "Light Paper", icon: Sun },
              { id: "dark", label: "Dark Slate", icon: Moon },
              { id: "system", label: "System", icon: Monitor },
            ].map((t) => {
              const Icon = t.icon;
              const active = preferences.theme === t.id;
              return (
                <button
                  key={t.id}
                  type="button"
                  onClick={() => updatePref({ theme: t.id })}
                  className={`flex items-center gap-2 px-3.5 py-2.5 rounded-lg border text-xs font-medium transition-colors ${
                    active
                      ? "border-emerald-600 bg-emerald-50/70 dark:bg-emerald-950/50 text-emerald-900 dark:text-emerald-200"
                      : "border-stone-200 dark:border-stone-800 text-stone-700 dark:text-stone-300"
                  }`}
                >
                  <Icon className="w-4 h-4 text-emerald-700 dark:text-emerald-400" />
                  <span>{t.label}</span>
                </button>
              );
            })}
          </div>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-2">
          <div>
            <label className="block text-xs font-medium text-stone-500 mb-1.5">
              Preferred Pronunciation Accent
            </label>
            <div className="flex items-center gap-2">
              {["US", "UK"].map((acc) => (
                <button
                  key={acc}
                  type="button"
                  onClick={() => updatePref({ pronunciation_accent: acc })}
                  className={`inline-flex items-center gap-1.5 px-3.5 py-2 rounded-lg border text-xs font-medium ${
                    preferences.pronunciation_accent === acc
                      ? "border-emerald-600 bg-emerald-50 dark:bg-emerald-950/50 text-emerald-800 dark:text-emerald-300"
                      : "border-stone-200 dark:border-stone-800"
                  }`}
                >
                  <Volume2 className="w-3.5 h-3.5" />
                  <span>{acc === "US" ? "American (US)" : "British (UK)"}</span>
                </button>
              ))}
            </div>
          </div>

          <div>
            <label className="block text-xs font-medium text-stone-500 mb-1.5">
              Default Startup View
            </label>
            <select
              value={preferences.default_view}
              onChange={(e) => updatePref({ default_view: e.target.value })}
              className="w-full px-3 py-2 text-xs rounded-lg border border-stone-200 dark:border-stone-800 bg-stone-50 dark:bg-stone-950"
            >
              <option value="home">Home Dashboard</option>
              <option value="practice">Interactive Practice</option>
              <option value="dictionary">Dictionary Explorer</option>
              <option value="lists">My Lists</option>
              <option value="discover">Discover</option>
              <option value="progress">Progress</option>
            </select>
          </div>
        </div>
      </div>

      {/* Session Persistence Backend */}
      <div className="rounded-xl border border-stone-200/90 dark:border-stone-800 bg-white dark:bg-stone-900 p-6 space-y-4">
        <h2 className="text-sm font-semibold text-stone-900 dark:text-stone-100">
          Session Storage Security
        </h2>
        <p className="text-xs text-stone-500 leading-relaxed">
          Choose where the Go backend persists your authenticated session cookies and Bearer JWT. No secrets are ever stored in WebView LocalStorage or IndexedDB.
        </p>

        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          {[
            {
              id: SessionStoreMode.SessionStoreKeyring,
              label: "OS Native Keyring",
              desc: "macOS Keychain / Windows Credential Manager / Secret Service",
              icon: ShieldCheck,
            },
            {
              id: SessionStoreMode.SessionStoreFile,
              label: "Restricted Local File",
              desc: "Atomic 0600 JSON file in user config directory",
              icon: HardDrive,
            },
            {
              id: SessionStoreMode.SessionStoreMemory,
              label: "Ephemeral Memory",
              desc: "Cleared immediately when application closes",
              icon: Cpu,
            },
          ].map((item) => {
            const Icon = item.icon;
            const active = preferences.session_store_mode === item.id;
            return (
              <button
                key={item.id}
                type="button"
                disabled={saving}
                onClick={() => updatePref({ session_store_mode: item.id })}
                className={`p-3.5 rounded-xl border text-left transition-colors ${
                  active
                    ? "border-emerald-600 bg-emerald-50/70 dark:bg-emerald-950/40"
                    : "border-stone-200 dark:border-stone-800 hover:bg-stone-50 dark:hover:bg-stone-800/40"
                }`}
              >
                <Icon className="w-4 h-4 text-emerald-700 dark:text-emerald-400 mb-1.5" />
                <p className="text-xs font-semibold text-stone-900 dark:text-stone-100">
                  {item.label}
                </p>
                <p className="text-[11px] text-stone-500 dark:text-stone-400 mt-0.5">
                  {item.desc}
                </p>
              </button>
            );
          })}
        </div>
      </div>

      {/* Keyboard Shortcuts Reference */}
      <div className="rounded-xl border border-stone-200/90 dark:border-stone-800 bg-white dark:bg-stone-900 p-6 space-y-4">
        <div className="flex items-center gap-2">
          <Keyboard className="w-4 h-4 text-emerald-700 dark:text-emerald-400" />
          <h2 className="text-sm font-semibold text-stone-900 dark:text-stone-100">
            Keyboard Shortcuts
          </h2>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-2.5 text-xs">
          {[
            { keys: "1 – 4", action: "Select answer choice in VocabTrainer" },
            { keys: "Enter", action: "Advance to next question after completion" },
            { keys: "H", action: "Use lifeline hint on current question" },
            { keys: "/", action: "Focus Dictionary search bar" },
            { keys: "Space", action: "Flip flashcard in Lists view" },
            { keys: "← / →", action: "Previous / Next flashcard" },
          ].map((sc) => (
            <div
              key={sc.keys}
              className="flex items-center justify-between p-2.5 rounded-lg bg-stone-50 dark:bg-stone-950 border border-stone-200/60 dark:border-stone-800/70"
            >
              <span className="text-stone-600 dark:text-stone-300">{sc.action}</span>
              <Kbd>{sc.keys}</Kbd>
            </div>
          ))}
        </div>
        <p className="text-[11px] text-stone-400 pt-2">
          Vocabulary Studio v{preferences.app_version || "1.0.0"} · Powered by vocabulary_go SDK & Wails v3
        </p>
      </div>
    </div>
  );
};
