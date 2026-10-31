import React, { useEffect, useState, useCallback } from "react";
import {
  Play,
  BookOpen,
  Sparkles,
  Flame,
  Clock,
  Award,
  AlertTriangle,
  ArrowRight,
  ListChecks,
  History,
  RefreshCw,
  LogIn,
  CalendarDays,
} from "lucide-react";
import type { HomeDashboardDTO, AccountStatusDTO } from "../lib/bridge";
import { getBridge, formatError } from "../lib/bridge";
import { AudioButton, ErrorBanner } from "../components/Shared";
import {
  Badge,
  Button,
  Card,
  Chip,
  IconButton,
  Meter,
  PageHeader,
  SectionTitle,
  Skeleton,
  StatTile,
} from "../components/ui";

const LinkButton: React.FC<{ onClick: () => void; children: React.ReactNode }> = ({ onClick, children }) => (
  <button
    type="button"
    onClick={onClick}
    className="group inline-flex items-center gap-1 font-nav text-meta font-medium text-accent-ink hover:underline underline-offset-4 rounded"
  >
    <span>{children}</span>
    <ArrowRight className="w-3.5 h-3.5 transition-transform duration-150 group-hover:translate-x-0.5" aria-hidden />
  </button>
);

export const HomeView: React.FC<{
  account: AccountStatusDTO | null;
  onStartPractice: (listId?: number) => void;
  onOpenWord: (word: string) => void;
  onOpenList: (listId: number) => void;
  onNavigate: (view: string) => void;
  onRequestLogin: () => void;
}> = ({ account, onStartPractice, onOpenWord, onOpenList, onNavigate, onRequestLogin }) => {
  const [dashboard, setDashboard] = useState<HomeDashboardDTO | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadDashboard = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getBridge().progress.GetHomeDashboard();
      if (data) {
        setDashboard(data);
      }
    } catch (err) {
      setError(formatError(err));
    } finally {
      setLoading(false);
    }
  }, [account?.authenticated]);

  useEffect(() => {
    loadDashboard();
  }, [loadDashboard]);

  const wod = dashboard?.word_of_the_day;
  const effort = dashboard?.effort;
  const isAuthed = Boolean(account?.authenticated || dashboard?.account?.authenticated);
  const nickname = account?.nickname || dashboard?.account?.nickname || "Learner";
  const attentionWords = dashboard?.trouble_words?.length ? dashboard.trouble_words : dashboard?.learning_words || [];

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow={isAuthed ? `Welcome back, ${nickname}` : "Vocabulary Studio"}
        title="What would you like to study today?"
        actions={
          <>
            <IconButton
              icon={RefreshCw}
              label="Refresh dashboard"
              variant="secondary"
              onClick={loadDashboard}
              disabled={loading}
              className={loading ? "[&_svg]:animate-spin" : undefined}
            />
            <Button variant="primary" size="lg" icon={Play} onClick={() => onStartPractice(0)}>
              Continue Learning
            </Button>
          </>
        }
      />

      {error && <ErrorBanner message={error} onRetry={loadDashboard} />}

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-5">
        {/* Study effort or sign-in invitation */}
        <Card pad="lg" className="lg:col-span-7 flex flex-col gap-6">
          {isAuthed ? (
            <>
              <div className="flex items-start justify-between gap-3">
                <div className="space-y-1">
                  <SectionTitle as="h2">7-Day Study Effort</SectionTitle>
                  <p className="font-nav text-[1.0625rem] font-semibold text-ink">
                    {account?.level?.name || dashboard?.account?.level?.name || "Learner"}
                    <span className="mx-2 text-ink-subtle font-normal">·</span>
                    <span className="font-data tabular-nums">
                      {(account?.points ?? dashboard?.account?.points ?? 0).toLocaleString()}
                    </span>{" "}
                    <span className="font-ui text-meta font-normal text-ink-muted">pts</span>
                  </p>
                </div>
                <LinkButton onClick={() => onNavigate("progress")}>Full progress</LinkButton>
              </div>

              {effort ? (
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
                  <StatTile label="Mastered" icon={Award} value={effort.words_mastered.toLocaleString()} />
                  <StatTile label="Answered" icon={Flame} value={effort.questions_answered.toLocaleString()} />
                  <StatTile label="Points" icon={Sparkles} value={effort.points.toLocaleString()} />
                  <StatTile label="Time" icon={Clock} value={`${effort.time_minutes}m`} />
                </div>
              ) : loading ? (
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
                  {[0, 1, 2, 3].map((i) => (
                    <Skeleton key={i} className="h-[4.25rem] rounded-[0.7rem]" />
                  ))}
                </div>
              ) : (
                <p className="text-ui text-ink-muted">Start a practice round to record your weekly learning effort.</p>
              )}

              {dashboard?.in_progress_activities && dashboard.in_progress_activities.length > 0 && (
                <div className="space-y-2.5">
                  <SectionTitle as="h3">Where You Left Off</SectionTitle>
                  <ul className="space-y-2">
                    {dashboard.in_progress_activities.slice(0, 3).map((act) => {
                      const listId = act.wordlist_id || act.list_id || 0;
                      const pct = Math.round((act.progress || 0) * 100);
                      return (
                        <li
                          key={act.id}
                          className="flex items-center gap-4 px-3.5 py-3 rounded-[0.7rem] border border-line bg-surface hover:border-line-strong transition-colors"
                        >
                          <div className="min-w-0 flex-1 space-y-1.5">
                            <div className="flex items-baseline justify-between gap-3">
                              <p className="font-nav text-ui font-medium text-ink truncate">
                                {act.name || `Activity #${act.id}`}
                              </p>
                              <span className="font-data text-micro text-ink-subtle shrink-0 tabular-nums">
                                {pct}% complete
                              </span>
                            </div>
                            <Meter decorative label={`${act.name || "Activity"} progress`} value={pct} />
                          </div>
                          {listId > 0 && (
                            <Button size="sm" variant="soft" icon={Play} onClick={() => onStartPractice(listId)}>
                              Resume
                            </Button>
                          )}
                        </li>
                      );
                    })}
                  </ul>
                </div>
              )}
            </>
          ) : (
            <div className="flex flex-col gap-4 h-full">
              <Badge tone="accent" icon={Sparkles} className="self-start">
                Adaptive Vocabulary Trainer
              </Badge>
              <h2 className="font-nav text-[1.25rem] leading-snug font-semibold text-ink max-w-[30ch]">
                Connect your Vocabulary.com account for personalized training
              </h2>
              <p className="font-read text-read text-ink-muted max-w-[60ch]">
                You can explore the full dictionary, corpus examples, Word of the Day, and public vocabulary lists right
                now. Sign in with your Vocabulary.com account to sync your personal lists, adaptive VocabTrainer
                sessions, trouble words, and mastery progress.
              </p>
              <div className="flex flex-wrap items-center gap-2.5 pt-1 mt-auto">
                <Button variant="primary" icon={LogIn} onClick={onRequestLogin}>
                  Sign In
                </Button>
                <Button icon={BookOpen} onClick={() => onNavigate("dictionary")}>
                  Explore Dictionary
                </Button>
              </div>
            </div>
          )}
        </Card>

        {/* Word of the Day */}
        <Card pad="lg" className="lg:col-span-5 flex flex-col relative overflow-hidden">
          <div
            aria-hidden
            className="pointer-events-none absolute -right-16 -top-16 w-56 h-56 rounded-full bg-accent-soft blur-3xl opacity-70"
          />
          {wod ? (
            <div className="relative flex flex-col gap-4 h-full">
              <div className="flex items-center justify-between gap-3">
                <SectionTitle as="h2" icon={CalendarDays}>
                  Word of the Day · {wod.date || "Today"}
                </SectionTitle>
                <AudioButton audioUrl={wod.audio_url} word={wod.word} size="sm" label="Listen" />
              </div>
              <button
                type="button"
                onClick={() => onOpenWord(wod.word)}
                className="self-start font-word text-[2.5rem] leading-none font-semibold tracking-[-0.03em] text-ink hover:text-accent-ink transition-colors text-left rounded"
              >
                {wod.word}
              </button>
              {wod.short_usage && (
                <p className="font-read text-read text-ink-muted line-clamp-5">{wod.short_usage}</p>
              )}
              <div className="mt-auto pt-2 flex items-center justify-between">
                <LinkButton onClick={() => onOpenWord(wod.word)}>Read full entry & examples</LinkButton>
                <Button size="sm" variant="ghost" onClick={() => onNavigate("discover")}>
                  Archive
                </Button>
              </div>
            </div>
          ) : loading ? (
            <div className="space-y-4" aria-busy>
              <span className="sr-only" role="status">
                Loading Word of the Day...
              </span>
              <Skeleton className="h-3 w-44" />
              <Skeleton className="h-10 w-52" />
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-4/5" />
            </div>
          ) : (
            <div className="flex items-center justify-center h-44 text-meta text-ink-subtle">
              Word of the Day unavailable
            </div>
          )}
        </Card>
      </div>

      {isAuthed && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
          <Card pad="none" className="flex flex-col">
            <div className="flex items-center justify-between gap-3 px-5 pt-4 pb-3">
              <SectionTitle as="h3" icon={AlertTriangle}>
                Words Needing Attention
              </SectionTitle>
              <Button size="sm" variant="ghost" onClick={() => onNavigate("progress")}>
                View all
              </Button>
            </div>
            {attentionWords.length > 0 ? (
              <ul className="px-2 pb-2">
                {attentionWords.slice(0, 5).map((w) => {
                  const pct = Math.round((w.progress || 0) * 100);
                  return (
                    <li key={w.word}>
                      <button
                        type="button"
                        onClick={() => onOpenWord(w.word)}
                        className="w-full grid grid-cols-[1fr_4.5rem] items-center gap-4 px-3 py-2.5 rounded-[0.6rem] text-left hover:bg-sunken transition-colors"
                      >
                        <span className="min-w-0">
                          <span className="block font-word text-[0.9375rem] font-semibold text-ink">{w.word}</span>
                          <span className="block font-read text-[0.9375rem] text-ink-muted truncate">
                            {w.definition}
                          </span>
                        </span>
                        <span className="space-y-1">
                          <span className="block text-right font-data text-micro text-ink-subtle tabular-nums">
                            {pct}%
                          </span>
                          <Meter decorative label={`${w.word} mastery`} value={pct} size="xs" />
                        </span>
                      </button>
                    </li>
                  );
                })}
              </ul>
            ) : (
              <p className="px-5 pb-5 text-meta text-ink-muted">
                No trouble words recorded yet. Practice questions to identify words that need extra review.
              </p>
            )}
          </Card>

          <Card pad="none" className="flex flex-col">
            <div className="flex items-center justify-between gap-3 px-5 pt-4 pb-3">
              <SectionTitle as="h3" icon={ListChecks}>
                Recent Vocabulary Lists
              </SectionTitle>
              <Button size="sm" variant="ghost" onClick={() => onNavigate("lists")}>
                All lists
              </Button>
            </div>
            {dashboard?.recent_lists && dashboard.recent_lists.length > 0 ? (
              <ul className="px-2 pb-2">
                {dashboard.recent_lists.slice(0, 5).map((item) => (
                  <li
                    key={item.wordlistid}
                    className="flex items-center justify-between gap-3 px-3 py-2 rounded-[0.6rem] hover:bg-sunken transition-colors"
                  >
                    <button
                      type="button"
                      onClick={() => onOpenList(item.wordlistid)}
                      className="text-left min-w-0 flex-1 rounded"
                    >
                      <span className="block font-nav text-ui font-medium text-ink truncate">{item.name}</span>
                      <span className="block font-data text-micro text-ink-subtle">{item.wordcount} words</span>
                    </button>
                    <Button size="sm" variant="secondary" onClick={() => onStartPractice(item.wordlistid)}>
                      Practice
                    </Button>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="px-5 pb-5 text-meta text-ink-muted">
                You don't have any recent vocabulary lists yet. Create a list or explore curated collections.
              </p>
            )}
          </Card>
        </div>
      )}

      {dashboard?.search_history && dashboard.search_history.length > 0 && (
        <Card pad="md" className="space-y-3">
          <SectionTitle as="h3" icon={History}>
            Recently Looked Up
          </SectionTitle>
          <div className="flex flex-wrap gap-1.5">
            {dashboard.search_history.slice(0, 14).map((word) => (
              <Chip key={word} word onClick={() => onOpenWord(word)}>
                {word}
              </Chip>
            ))}
          </div>
        </Card>
      )}
    </div>
  );
};
