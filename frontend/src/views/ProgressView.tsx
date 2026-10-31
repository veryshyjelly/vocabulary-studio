import React, { useState, useEffect, useCallback } from "react";
import {
  TrendingUp,
  Award,
  Trophy,
  Activity,
  Loader2,
  LogIn,
  ArrowRight,
} from "lucide-react";
import type {
  AccountStatusDTO,
  AccountProgressWord,
  AchievementSummary,
  ActivitiesResponse,
  LeaderboardResponse,
} from "../lib/bridge";
import { getBridge, formatError } from "../lib/bridge";
import { ErrorBanner } from "../components/Shared";

type ProgressTab = "words" | "activities" | "achievements" | "leaderboards";

export const ProgressView: React.FC<{
  account: AccountStatusDTO | null;
  onOpenWord: (word: string) => void;
  onStartPractice: (listId: number) => void;
  onRequestLogin: () => void;
}> = ({ account, onOpenWord, onStartPractice, onRequestLogin }) => {
  const [tab, setTab] = useState<ProgressTab>(
    account?.authenticated ? "words" : "leaderboards"
  );

  // Words state
  const [wordFilter, setWordFilter] = useState<"learning" | "trouble" | "mastered">("learning");
  const [words, setWords] = useState<AccountProgressWord[]>([]);

  // Activities state
  const [activitiesResp, setActivitiesResp] = useState<ActivitiesResponse | null>(null);

  // Achievements state
  const [achievements, setAchievements] = useState<AchievementSummary[]>([]);

  // Leaderboards state
  const [lbMetric, setLbMetric] = useState<"points" | "mastered">("points");
  const [leaderboard, setLeaderboard] = useState<LeaderboardResponse | null>(null);

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadWords = useCallback(
    async (filter: "learning" | "trouble" | "mastered") => {
      if (!account?.authenticated) return;
      setLoading(true);
      setError(null);
      try {
        const list = await getBridge().progress.ListLearningWords(filter, 0, "", "desc");
        setWords(list || []);
      } catch (err) {
        setError(formatError(err));
      } finally {
        setLoading(false);
      }
    },
    [account?.authenticated]
  );

  const loadActivities = useCallback(
    async (nextUri = "") => {
      if (!account?.authenticated) return;
      setLoading(true);
      setError(null);
      try {
        const res = await getBridge().progress.GetActivities("", 20, nextUri);
        setActivitiesResp(res);
      } catch (err) {
        setError(formatError(err));
      } finally {
        setLoading(false);
      }
    },
    [account?.authenticated]
  );

  const loadAchievements = useCallback(async () => {
    if (!account?.authenticated) return;
    setLoading(true);
    setError(null);
    try {
      const list = await getBridge().progress.GetAchievements();
      setAchievements(list || []);
    } catch (err) {
      setError(formatError(err));
    } finally {
      setLoading(false);
    }
  }, [account?.authenticated]);

  const loadLeaderboard = useCallback(async (metric: "points" | "mastered") => {
    setLoading(true);
    setError(null);
    try {
      const res = await getBridge().progress.GetLeaderboard(metric, "", "", 25);
      setLeaderboard(res);
    } catch (err) {
      setError(formatError(err));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (tab === "words") loadWords(wordFilter);
    if (tab === "activities") loadActivities();
    if (tab === "achievements") loadAchievements();
    if (tab === "leaderboards") loadLeaderboard(lbMetric);
  }, [tab, wordFilter, lbMetric, loadWords, loadActivities, loadAchievements, loadLeaderboard]);

  const handlePriorityChange = async (word: string, priority: number) => {
    try {
      await getBridge().progress.SetWordPriority(word, priority);
      loadWords(wordFilter);
    } catch (err) {
      setError(formatError(err));
    }
  };

  return (
    <div className="max-w-5xl mx-auto space-y-6 pb-12">
      <div className="flex flex-wrap items-center justify-between gap-4 pb-4 border-b border-stone-200/80 dark:border-stone-800/80">
        <div>
          <h1 className="text-2xl font-editorial font-semibold text-stone-900 dark:text-stone-100">
            Learning Progress & Mastery
          </h1>
          <p className="text-xs text-stone-500 dark:text-stone-400">
            Review words in progress, trouble words, activity logs, earned milestones, and global leaderboards.
          </p>
        </div>

        <div className="inline-flex p-1 rounded-lg bg-stone-200/70 dark:bg-stone-900 border border-stone-200 dark:border-stone-800">
          {[
            { id: "words" as const, label: "My Words", icon: TrendingUp },
            { id: "activities" as const, label: "Activity Log", icon: Activity },
            { id: "achievements" as const, label: "Achievements", icon: Award },
            { id: "leaderboards" as const, label: "Leaderboards", icon: Trophy },
          ].map((t) => {
            const Icon = t.icon;
            const active = tab === t.id;
            return (
              <button
                key={t.id}
                type="button"
                onClick={() => {
                  setTab(t.id);
                  setError(null);
                }}
                className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
                  active
                    ? "bg-white dark:bg-stone-800 text-stone-900 dark:text-stone-100 shadow-2xs"
                    : "text-stone-600 dark:text-stone-400 hover:text-stone-900 dark:hover:text-stone-200"
                }`}
              >
                <Icon className="w-3.5 h-3.5 text-emerald-700 dark:text-emerald-400" />
                <span>{t.label}</span>
              </button>
            );
          })}
        </div>
      </div>

      {error && <ErrorBanner message={error} onDismiss={() => setError(null)} />}

      {/* Auth Gate for Personal Progress Tabs */}
      {tab !== "leaderboards" && !account?.authenticated ? (
        <div className="p-10 rounded-xl border border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 text-center space-y-4">
          <h2 className="text-lg font-editorial font-semibold text-stone-900 dark:text-stone-100">
            Sign in to view your personal vocabulary mastery
          </h2>
          <p className="text-xs text-stone-500 max-w-md mx-auto">
            Your learning queue, trouble words, activity history, and earned achievements are synced directly from your Vocabulary.com account.
          </p>
          <button
            type="button"
            onClick={onRequestLogin}
            className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-700 text-white text-xs font-medium"
          >
            <LogIn className="w-3.5 h-3.5" />
            <span>Sign In</span>
          </button>
        </div>
      ) : null}

      {/* TAB 1: MY WORDS */}
      {tab === "words" && account?.authenticated && (
        <div className="space-y-4">
          <div className="flex items-center gap-2">
            {[
              { id: "learning" as const, label: "Words I'm Learning" },
              { id: "trouble" as const, label: "Trouble Words" },
              { id: "mastered" as const, label: "Mastered Words" },
            ].map((f) => (
              <button
                key={f.id}
                type="button"
                onClick={() => setWordFilter(f.id)}
                className={`px-3.5 py-1.5 rounded-lg text-xs font-medium border transition-colors ${
                  wordFilter === f.id
                    ? "border-emerald-600 bg-emerald-50 dark:bg-emerald-950/60 text-emerald-800 dark:text-emerald-300"
                    : "border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 text-stone-600 dark:text-stone-400"
                }`}
              >
                {f.label}
              </button>
            ))}
          </div>

          {loading ? (
            <div className="p-12 flex justify-center">
              <Loader2 className="w-6 h-6 animate-spin text-emerald-700" />
            </div>
          ) : words.length > 0 ? (
            <div className="rounded-xl border border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 divide-y divide-stone-100 dark:divide-stone-800">
              {words.map((w) => {
                const pct = Math.round((w.progress || 0) * 100);
                return (
                  <div
                    key={w.word}
                    className="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3"
                  >
                    <div className="space-y-0.5 min-w-0">
                      <button
                        type="button"
                        onClick={() => onOpenWord(w.word)}
                        className="text-lg font-editorial font-semibold text-stone-900 dark:text-stone-100 hover:text-emerald-700 dark:hover:text-emerald-400"
                      >
                        {w.word}
                      </button>
                      <p className="text-xs text-stone-500 dark:text-stone-400 truncate">
                        {w.definition}
                      </p>
                    </div>

                    <div className="flex items-center gap-4 shrink-0">
                      <div className="w-28">
                        <div className="flex justify-between text-[11px] font-mono text-stone-500 mb-1">
                          <span>Mastery</span>
                          <span>{pct}%</span>
                        </div>
                        <div className="h-1.5 w-full rounded-full bg-stone-100 dark:bg-stone-800 overflow-hidden">
                          <div
                            className="h-full bg-emerald-600 rounded-full"
                            style={{ width: `${Math.min(100, Math.max(0, pct))}%` }}
                          />
                        </div>
                      </div>
                      <button
                        type="button"
                        onClick={() => handlePriorityChange(w.word, 1)}
                        className="px-2.5 py-1 text-[11px] rounded border border-stone-200 dark:border-stone-800 hover:border-emerald-500 text-stone-600 dark:text-stone-300"
                      >
                        Boost Priority
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          ) : (
            <div className="p-8 rounded-xl border border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 text-center text-sm text-stone-500">
              No words found in the "{wordFilter}" category yet.
            </div>
          )}
        </div>
      )}

      {/* TAB 2: ACTIVITIES */}
      {tab === "activities" && account?.authenticated && (
        <div className="space-y-4">
          {loading ? (
            <div className="p-12 flex justify-center">
              <Loader2 className="w-6 h-6 animate-spin text-emerald-700" />
            </div>
          ) : activitiesResp?.activities && activitiesResp.activities.length > 0 ? (
            <div className="space-y-3">
              {activitiesResp.activities.map((act) => {
                const listId = act.wordlist_id || act.list_id || 0;
                const pct = Math.round((act.progress || 0) * 100);
                return (
                  <div
                    key={act.id}
                    className="p-4 rounded-xl border border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 flex items-center justify-between gap-4"
                  >
                    <div>
                      <p className="text-sm font-semibold text-stone-900 dark:text-stone-100">
                        {act.name || `Activity #${act.id}`}
                      </p>
                      <p className="text-xs text-stone-500">
                        Type: {act.activity_type || act.type || "practice"} · Progress: {pct}%
                      </p>
                    </div>
                    {listId > 0 && (
                      <button
                        type="button"
                        onClick={() => onStartPractice(listId)}
                        className="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg bg-emerald-700 text-white text-xs font-medium"
                      >
                        <span>Practice</span>
                        <ArrowRight className="w-3.5 h-3.5" />
                      </button>
                    )}
                  </div>
                );
              })}
              {activitiesResp.next_page_uri && (
                <button
                  type="button"
                  onClick={() => loadActivities(activitiesResp.next_page_uri)}
                  className="px-4 py-2 rounded-lg border border-stone-200 dark:border-stone-800 text-xs font-medium"
                >
                  Load Next Page
                </button>
              )}
            </div>
          ) : (
            <div className="p-8 rounded-xl border border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 text-center text-sm text-stone-500">
              No recent activities recorded.
            </div>
          )}
        </div>
      )}

      {/* TAB 3: ACHIEVEMENTS */}
      {tab === "achievements" && account?.authenticated && (
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
          {achievements.map((ach) => (
            <div
              key={ach.id}
              className="p-4 rounded-xl border border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 space-y-2"
            >
              <div className="flex items-center justify-between">
                <span className="text-xs font-mono font-semibold text-emerald-700 dark:text-emerald-400">
                  {ach.id}
                </span>
                <span className="text-xs text-stone-500">Milestone {ach.milestone}</span>
              </div>
              <div className="h-1.5 w-full rounded-full bg-stone-100 dark:bg-stone-800 overflow-hidden">
                <div
                  className="h-full bg-amber-500 rounded-full"
                  style={{ width: `${Math.min(100, Math.round(ach.progress || 0))}%` }}
                />
              </div>
            </div>
          ))}
        </div>
      )}

      {/* TAB 4: LEADERBOARDS */}
      {tab === "leaderboards" && (
        <div className="space-y-4">
          <div className="flex items-center gap-2">
            {[
              { id: "points" as const, label: "Top Points" },
              { id: "mastered" as const, label: "Words Mastered" },
            ].map((m) => (
              <button
                key={m.id}
                type="button"
                onClick={() => setLbMetric(m.id)}
                className={`px-3.5 py-1.5 rounded-lg text-xs font-medium border ${
                  lbMetric === m.id
                    ? "border-emerald-600 bg-emerald-50 dark:bg-emerald-950/60 text-emerald-800 dark:text-emerald-300"
                    : "border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900"
                }`}
              >
                {m.label}
              </button>
            ))}
          </div>

          {leaderboard?.rows && leaderboard.rows.length > 0 && (
            <div className="rounded-xl border border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 divide-y divide-stone-100 dark:divide-stone-800">
              {leaderboard.rows.map((row) => (
                <div
                  key={row.rank}
                  className="px-5 py-3 flex items-center justify-between text-sm"
                >
                  <div className="flex items-center gap-3">
                    <span className="w-7 font-mono text-xs text-stone-400">#{row.rank}</span>
                    <span className="font-medium text-stone-900 dark:text-stone-100">
                      {row.profile?.nickname || "Learner"}
                    </span>
                    {row.profile?.level_name && (
                      <span className="text-xs text-stone-400">({row.profile.level_name})</span>
                    )}
                  </div>
                  <span className="font-mono font-semibold text-emerald-700 dark:text-emerald-400">
                    {row.points.toLocaleString()}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
};
