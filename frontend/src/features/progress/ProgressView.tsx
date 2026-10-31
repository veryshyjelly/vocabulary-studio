import React, { useState, useEffect, useCallback } from "react";
import { TrendingUp, Award, Trophy, Activity, LogIn, ArrowRight } from "lucide-react";
import type { AccountStatusDTO, AccountProgressWord, AchievementSummary, ActivitiesResponse, LeaderboardResponse } from "../../lib/bridge";
import { getBridge, formatError } from "../../lib/bridge";
import { Button, EmptyState, LoadingBlock, Meter, PageHeader, Tabs } from "../../components/ui";
import { ErrorBanner } from "../../components/shared";

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
      <PageHeader title="Learning Progress & Mastery" description="Revisit what you know. Give the difficult words a little more attention.">
        <Tabs<ProgressTab>
          label="Progress views"
          value={tab}
          onChange={(id) => { setTab(id); setError(null); }}
          items={[
            { id: "words", label: "My Words", icon: TrendingUp },
            { id: "activities", label: "Activity Log", icon: Activity },
            { id: "achievements", label: "Achievements", icon: Award },
            { id: "leaderboards", label: "Leaderboards", icon: Trophy },
          ]}
        />
      </PageHeader>

      {error && <ErrorBanner message={error} onDismiss={() => setError(null)} />}

      {/* Auth Gate for Personal Progress Tabs */}
      {tab !== "leaderboards" && !account?.authenticated ? (
        <div className="p-10 rounded-xl border border-line bg-surface text-center space-y-4">
          <h2 className="text-lg font-word font-semibold text-ink">
            Sign in to view your personal vocabulary mastery
          </h2>
          <p className="text-xs text-ink-subtle max-w-md mx-auto">
            Your learning queue, trouble words, activity history, and earned achievements are synced directly from your Vocabulary.com account.
          </p>
          <button
            type="button"
            onClick={onRequestLogin}
            className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-accent text-on-accent text-xs font-medium"
          >
            <LogIn className="w-3.5 h-3.5" />
            <span>Sign In</span>
          </button>
        </div>
      ) : null}

      {/* TAB 1: MY WORDS */}
      {tab === "words" && account?.authenticated && (
        <div className="space-y-4">
          <div className="flex flex-wrap items-center gap-2">
            {[
              { id: "learning" as const, label: "Words I'm Learning" },
              { id: "trouble" as const, label: "Trouble Words" },
              { id: "mastered" as const, label: "Mastered Words" },
            ].map((f) => (
              <button
                key={f.id}
                type="button"
                aria-pressed={wordFilter === f.id}
                onClick={() => setWordFilter(f.id)}
                className={`px-3.5 py-1.5 rounded-lg text-xs font-medium border transition-colors ${
                  wordFilter === f.id
                    ? "border-accent-line bg-accent-soft text-accent-ink"
                    : "border-line bg-surface text-ink-muted"
                }`}
              >
                {f.label}
              </button>
            ))}
          </div>

          {loading ? (
            <LoadingBlock label="Loading your progress…" />
          ) : words.length > 0 ? (
            <div className="rounded-xl border border-line bg-surface divide-y divide-line">
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
                        className="font-word text-[1.125rem] font-semibold text-ink hover:text-accent-ink text-left [overflow-wrap:anywhere]"
                      >
                        {w.word}
                      </button>
                      <p className="font-read text-read text-ink-muted line-clamp-2">
                        {w.definition}
                      </p>
                    </div>

                    <div className="flex items-center gap-4 shrink-0">
                      <div className="w-28">
                        <div className="flex justify-between text-[11px] font-data text-ink-subtle mb-1">
                          <span>Mastery</span>
                          <span>{pct}%</span>
                        </div>
                        <Meter label={`Mastery of ${w.word}`} value={pct} />
                      </div>
                      <button
                        type="button"
                        onClick={() => handlePriorityChange(w.word, 1)}
                        className="px-2.5 py-1 text-[11px] rounded border border-line hover:border-accent-line text-ink-muted"
                      >
                        Boost Priority
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          ) : (
            <EmptyState icon={TrendingUp} title={`No ${wordFilter} words yet`}
              description="Your words will appear here as you study. A short practice round is a good place to begin."
              action={<Button variant="primary" onClick={() => onStartPractice(0)}>Start Practice</Button>} />
          )}
        </div>
      )}

      {/* TAB 2: ACTIVITIES */}
      {tab === "activities" && account?.authenticated && (
        <div className="space-y-4">
          {loading ? (
            <LoadingBlock label="Loading your progress…" />
          ) : activitiesResp?.activities && activitiesResp.activities.length > 0 ? (
            <div className="space-y-3">
              {activitiesResp.activities.map((act) => {
                const listId = act.wordlist_id || act.list_id || 0;
                const pct = Math.round((act.progress || 0) * 100);
                return (
                  <div
                    key={act.id}
                    className="p-4 rounded-xl border border-line bg-surface flex items-center justify-between gap-4"
                  >
                    <div>
                      <p className="text-sm font-semibold text-ink">
                        {act.name || `Activity #${act.id}`}
                      </p>
                      <p className="text-xs text-ink-subtle">
                        Type: {act.activity_type || act.type || "practice"} · Progress: {pct}%
                      </p>
                    </div>
                    {listId > 0 && (
                      <button
                        type="button"
                        onClick={() => onStartPractice(listId)}
                        className="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg bg-accent text-on-accent text-xs font-medium"
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
                  className="px-4 py-2 rounded-lg border border-line text-xs font-medium"
                >
                  Load Next Page
                </button>
              )}
            </div>
          ) : (
            <EmptyState icon={Activity} title="Your next session starts the story" description="Completed and in-progress learning activities will appear here."
              action={<Button onClick={() => onStartPractice(0)}>Start Practice</Button>} />
          )}
        </div>
      )}

      {/* TAB 3: ACHIEVEMENTS */}
      {tab === "achievements" && account?.authenticated && (
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
          {loading && <LoadingBlock label="Loading achievements…" className="col-span-full" />}
          {!loading && achievements.length === 0 && <EmptyState className="col-span-full" icon={Award} title="Milestones take a little practice" description="Your earned achievements will appear here as your vocabulary grows." />}
          {!loading && achievements.map((ach) => (
            <div
              key={ach.id}
              className="p-4 rounded-xl border border-line bg-surface space-y-2"
            >
              <div className="flex items-center justify-between">
                <span className="text-xs font-data font-semibold text-accent-ink">
                  {ach.id}
                </span>
                <span className="text-xs text-ink-subtle">Milestone {ach.milestone}</span>
              </div>
              <Meter label={`Achievement ${ach.id} progress`} tone="warning" value={ach.progress || 0} />
            </div>
          ))}
        </div>
      )}

      {/* TAB 4: LEADERBOARDS */}
      {tab === "leaderboards" && (
        <div className="space-y-4">
          <div className="flex flex-wrap items-center gap-2">
            {[
              { id: "points" as const, label: "Top Points" },
              { id: "mastered" as const, label: "Words Mastered" },
            ].map((m) => (
              <button
                key={m.id}
                type="button"
                aria-pressed={lbMetric === m.id}
                onClick={() => setLbMetric(m.id)}
                className={`px-3.5 py-1.5 rounded-lg text-xs font-medium border ${
                  lbMetric === m.id
                    ? "border-accent-line bg-accent-soft text-accent-ink"
                    : "border-line bg-surface"
                }`}
              >
                {m.label}
              </button>
            ))}
          </div>

          {loading ? <LoadingBlock label="Loading leaderboard…" /> : leaderboard?.rows?.length ? (
            <div className="rounded-card border border-line bg-surface overflow-x-auto">
              <table className="w-full text-left text-ui">
                <caption className="sr-only">{lbMetric === "points" ? "Top Points" : "Words Mastered"} leaderboard</caption>
                <thead className="bg-sunken border-b border-line font-nav text-meta text-ink-muted">
                  <tr><th scope="col" className="px-5 py-3 font-medium">Rank</th><th scope="col" className="px-3 py-3 font-medium">Learner</th><th scope="col" className="px-5 py-3 text-right font-medium">{lbMetric === "points" ? "Points" : "Words mastered"}</th></tr>
                </thead>
                <tbody className="divide-y divide-line">
                  {leaderboard.rows.map((row) => (
                    <tr key={row.rank}>
                      <td className="px-5 py-3 font-data tabular-nums text-ink-subtle">#{row.rank}</td>
                      <td className="px-3 py-3 font-nav"><span className="font-medium">{row.profile?.nickname || "Learner"}</span>{row.profile?.level_name && <span className="block text-meta text-ink-muted mt-0.5">{row.profile.level_name}</span>}</td>
                      <td className="px-5 py-3 text-right font-data font-medium tabular-nums text-accent-ink">{row.points.toLocaleString()}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : !error && <EmptyState icon={Trophy} title="No leaderboard entries yet" description="Try a different ranking to explore the community's progress." />}

        </div>
      )}
    </div>
  );
};
