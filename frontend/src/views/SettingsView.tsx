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
import { ErrorBanner } from "../components/Shared";
import { Button, Field, PageHeader, Select, cx } from "../components/ui";
import { ShortcutList } from "../components/Shortcuts";

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
    <div className="max-w-[880px] mx-auto space-y-8">
      <PageHeader title="Settings & Preferences" description="Make this a comfortable place to learn." />

      {error && <ErrorBanner message={error} onDismiss={() => setError(null)} />}
      <p role="status" aria-live="polite" className="sr-only">{saving ? "Saving preferences…" : savedNotice || ""}</p>

      <section className="space-y-4" aria-labelledby="account-heading">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 id="account-heading" className="font-nav text-ui font-semibold flex items-center gap-2">
            <UserCheck className="w-4 h-4 text-accent-ink" aria-hidden /> Vocabulary.com Account
          </h2>
          {account?.authenticated ? (
            <Button variant="danger-outline" icon={LogOut} disabled={saving} onClick={handleLogout}>Sign Out</Button>
          ) : (
            <Button variant="primary" icon={LogIn} onClick={onRequestLogin}>Sign In</Button>
          )}
        </div>
        {account?.authenticated ? (
          <dl className="grid grid-cols-2 md:grid-cols-4 gap-x-5 gap-y-4 p-5 rounded-card bg-sunken">
            {[
              ["Nickname", account.nickname],
              ["Level", account.level?.name || "Learner"],
              ["Points", account.points.toLocaleString()],
              ["VocabTrainer", account.full_vocab_trainer ? "Full Subscriber" : "Standard"],
            ].map(([label, value]) => (
              <div key={label} className="min-w-0 space-y-1">
                <dt className="eyebrow text-ink-subtle">{label}</dt>
                <dd className="font-nav text-ui font-medium break-words">{value}</dd>
              </div>
            ))}
          </dl>
        ) : (
          <p className="reading-copy text-ink-muted">Your dictionary, examples, and Word of the Day are available without signing in. Connect an account to sync your lists and progress.</p>
        )}
      </section>

      <section className="settings-section space-y-5" aria-labelledby="appearance-heading">
        <h2 id="appearance-heading" className="font-nav text-ui font-semibold">Appearance & Audio</h2>
        <fieldset>
          <legend className="font-nav text-meta text-ink-muted mb-3">Color Theme</legend>
          <div className="grid grid-cols-3 gap-3">
            {[
              { id: "light", label: "Light Paper", icon: Sun },
              { id: "dark", label: "Dark Slate", icon: Moon },
              { id: "system", label: "System", icon: Monitor },
            ].map((theme) => {
              const Icon = theme.icon;
              const active = preferences.theme === theme.id;
              return (
                <button key={theme.id} type="button" aria-pressed={active} disabled={saving}
                  onClick={() => updatePref({ theme: theme.id })}
                  className={cx("p-2.5 rounded-control border text-left space-y-2.5 transition-colors disabled:opacity-50", active ? "border-accent-line bg-accent-soft" : "border-line hover:border-line-strong")}>
                  <span className="theme-swatch" data-appearance={theme.id} aria-hidden>
                    <span />
                    <span className="flex flex-col justify-center gap-1.5 px-3">
                      <span className="font-word text-[1rem] leading-none">Aa</span>
                      <span className="w-3/4 h-px bg-current opacity-25" />
                      <span className="w-1/2 h-px bg-current opacity-25" />
                    </span>
                  </span>
                  <span className="flex flex-wrap items-center gap-1.5 font-nav text-meta">
                    <Icon className="w-3.5 h-3.5 text-accent-ink" aria-hidden /> {theme.label}
                    {active && <Check className="w-3.5 h-3.5 ml-auto text-accent-ink" aria-hidden />}
                  </span>
                </button>
              );
            })}
          </div>
        </fieldset>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
          <fieldset>
            <legend className="font-nav text-meta text-ink-muted mb-2">Preferred Pronunciation Accent</legend>
            <div className="flex flex-wrap gap-2">
              {["US", "UK"].map((accent) => (
                <Button key={accent} disabled={saving} aria-pressed={preferences.pronunciation_accent === accent}
                  variant={preferences.pronunciation_accent === accent ? "soft" : "secondary"}
                  icon={Volume2} onClick={() => updatePref({ pronunciation_accent: accent })}>
                  {accent === "US" ? "American (US)" : "British (UK)"}
                </Button>
              ))}
            </div>
          </fieldset>
          <Field label="Default Startup View" htmlFor="startup-view">
            <Select id="startup-view" disabled={saving} value={preferences.default_view}
              onChange={(e) => updatePref({ default_view: e.target.value })}>
              <option value="home">Home Dashboard</option>
              <option value="practice">Interactive Practice</option>
              <option value="dictionary">Dictionary Explorer</option>
              <option value="lists">My Lists</option>
              <option value="discover">Discover</option>
              <option value="progress">Progress</option>
            </Select>
          </Field>
        </div>
      </section>

      <section className="settings-section space-y-4" aria-labelledby="storage-heading">
        <h2 id="storage-heading" className="font-nav text-ui font-semibold">Session Storage Security</h2>
        <p className="reading-copy text-ink-muted">Choose how this device remembers your sign-in. Passwords and session tokens stay out of browser storage.</p>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          {[
            { id: SessionStoreMode.SessionStoreKeyring, label: "OS Native Keyring", desc: "Protected by your system keychain. Recommended for everyday use.", icon: ShieldCheck },
            { id: SessionStoreMode.SessionStoreFile, label: "Restricted Local File", desc: "Saved locally with access restricted to your user account.", icon: HardDrive },
            { id: SessionStoreMode.SessionStoreMemory, label: "Ephemeral Memory", desc: "Remember this session until you quit the app.", icon: Cpu },
          ].map((item) => {
            const Icon = item.icon;
            const active = preferences.session_store_mode === item.id;
            return (
              <button key={item.id} type="button" disabled={saving} aria-pressed={active}
                onClick={() => updatePref({ session_store_mode: item.id })}
                className={cx("p-4 rounded-control border text-left space-y-2 transition-colors disabled:opacity-50", active ? "border-accent-line bg-accent-soft" : "border-line hover:border-line-strong hover:bg-sunken")}>
                <span className="flex justify-between gap-2 text-accent-ink"><Icon className="w-4 h-4" aria-hidden />{active && <Check className="w-4 h-4" aria-hidden />}</span>
                <span className="block font-nav text-meta font-semibold">{item.label}</span>
                <span className="block text-meta leading-relaxed text-ink-muted">{item.desc}</span>
              </button>
            );
          })}
        </div>
      </section>

      <section className="settings-section space-y-5" aria-labelledby="keyboard-heading">
        <h2 id="keyboard-heading" className="font-nav text-ui font-semibold flex items-center gap-2"><Keyboard className="w-4 h-4 text-accent-ink" aria-hidden /> Keyboard Shortcuts</h2>
        <ShortcutList columns={2} />
      </section>
      <footer className="text-meta text-ink-subtle border-t border-line pt-4">
        Vocabulary Studio v{preferences.app_version || "1.0.0"}
        <span className="mx-2" aria-hidden>·</span>Type set in Monaspace
      </footer>
    </div>
  );
};
