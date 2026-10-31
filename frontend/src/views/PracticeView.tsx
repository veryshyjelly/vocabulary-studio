import React, { useState, useEffect, useCallback, useRef } from "react";
import {
  Play,
  CheckCircle2,
  XCircle,
  Sparkles,
  Lightbulb,
  ArrowRight,
  RotateCcw,
  BookOpen,
  Loader2,
  Flame,
  SpellCheck,
  ListChecks,
  HelpCircle,
  Flag,
  LogIn,
  Volume2,
} from "lucide-react";
import type {
  TrainerSessionStateDTO,
  HintResultDTO,
  BeeSessionDTO,
  QuizSessionDTO,
  QuizQuestionResult,
  AccountStatusDTO,
} from "../lib/bridge";
import { getBridge, formatError } from "../lib/bridge";
import { acceptsPlainShortcut, isModalOpen } from "../lib/keys";
import { RichText, AudioButton, ErrorBanner } from "../components/Shared";
import { Badge, Button, Card, EmptyState, Field, Kbd, Meter, Skeleton, Tabs, TextInput, cx } from "../components/ui";

type PracticeMode = "trainer" | "bee" | "quiz";

const ROUND_LENGTH = 10;

function hintLabel(hintType: string): string {
  return hintType === "F" ? "50/50" : hintType === "D" ? "Definition" : "Word in the Wild";
}

export const PracticeView: React.FC<{
  account: AccountStatusDTO | null;
  initialListId?: number;
  initialMode?: PracticeMode;
  onOpenWord: (word: string) => void;
  onRequestLogin: () => void;
}> = ({ account, initialListId = 0, initialMode = "trainer", onOpenWord, onRequestLogin }) => {
  const [mode, setMode] = useState<PracticeMode>(initialMode);
  const [listIdInput, setListIdInput] = useState<string>(initialListId > 0 ? String(initialListId) : "");

  // Trainer / List Practice State
  const [sessionState, setSessionState] = useState<TrainerSessionStateDTO | null>(null);
  const [selectedChoices, setSelectedChoices] = useState<Record<number, "correct" | "wrong">>({});
  const [spellingInput, setSpellingInput] = useState("");
  const [activeHints, setActiveHints] = useState<HintResultDTO[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [pendingChoice, setPendingChoice] = useState<number | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Spelling Bee State
  const [beeState, setBeeState] = useState<BeeSessionDTO | null>(null);
  const [beeGuessInput, setBeeGuessInput] = useState("");

  // Quiz State
  const [quizIdInput, setQuizIdInput] = useState("");
  const [quizState, setQuizState] = useState<QuizSessionDTO | null>(null);
  const [quizSelectedNonce, setQuizSelectedNonce] = useState<string>("");
  const [quizResults, setQuizResults] = useState<QuizQuestionResult[] | null>(null);

  const spellingRef = useRef<HTMLInputElement | null>(null);
  const beeGuessRef = useRef<HTMLInputElement | null>(null);

  const startTrainer = useCallback(
    async (overrideListId?: number) => {
      setLoading(true);
      setError(null);
      setSelectedChoices({});
      setSpellingInput("");
      setActiveHints([]);
      const targetList =
        overrideListId !== undefined ? overrideListId : listIdInput.trim() ? Number(listIdInput.trim()) : 0;
      try {
        const st = await getBridge().trainer.StartSession(targetList || 0, "");
        setSessionState(st);
      } catch (err) {
        setError(formatError(err));
      } finally {
        setLoading(false);
      }
    },
    [listIdInput]
  );

  // Auto-start when navigated with an initialListId or on first mount of trainer mode
  useEffect(() => {
    if (initialListId > 0) {
      setListIdInput(String(initialListId));
    }
    if (initialMode) {
      setMode(initialMode);
    }
  }, [initialListId, initialMode]);

  useEffect(() => {
    if (mode === "trainer" && !sessionState && !loading && !error) {
      startTrainer(initialListId);
    }
  }, [mode, sessionState, loading, error, initialListId, startTrainer]);

  const handleChoiceSelect = useCallback(
    async (index: number) => {
      if (!sessionState?.question || submitting) return;
      if (sessionState.session_state === "question_completed") return;
      if (selectedChoices[index]) return;

      const choice = sessionState.question.choices?.[index];
      if (!choice || choice.eliminated) return;

      setSubmitting(true);
      setPendingChoice(index);
      setError(null);
      try {
        const updated = await getBridge().trainer.SubmitChoice(index);
        if (updated) {
          setSessionState(updated);
          const isCorrect = Boolean(updated.last_feedback?.correct);
          setSelectedChoices((prev) => ({
            ...prev,
            [index]: isCorrect ? "correct" : "wrong",
          }));
        }
      } catch (err) {
        setError(formatError(err));
      } finally {
        setPendingChoice(null);
        setSubmitting(false);
      }
    },
    [sessionState, submitting, selectedChoices]
  );

  const handleSpellingSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!sessionState?.question || submitting || !spellingInput.trim()) return;
    if (sessionState.session_state === "question_completed") return;

    setSubmitting(true);
    setError(null);
    try {
      const updated = await getBridge().trainer.SubmitSpelling(spellingInput);
      if (updated) {
        setSessionState(updated);
      }
    } catch (err) {
      setError(formatError(err));
    } finally {
      setSubmitting(false);
    }
  };

  const handleUseHint = useCallback(
    async (hintType: string) => {
      if (!sessionState?.question || submitting || sessionState.tries > 0) return;
      setSubmitting(true);
      setError(null);
      try {
        const hintRes = await getBridge().trainer.UseHint(hintType);
        if (hintRes) {
          setActiveHints((prev) => [...prev, hintRes]);
          const refreshed = await getBridge().trainer.GetSessionState();
          if (refreshed) {
            setSessionState(refreshed);
          }
        }
      } catch (err) {
        setError(formatError(err));
      } finally {
        setSubmitting(false);
      }
    },
    [sessionState, submitting]
  );

  const handleNextQuestion = useCallback(async () => {
    if (submitting || loading) return;
    setLoading(true);
    setError(null);
    setSelectedChoices({});
    setSpellingInput("");
    setActiveHints([]);
    try {
      const nextSt = await getBridge().trainer.NextQuestion();
      setSessionState(nextSt);
    } catch (err) {
      setError(formatError(err));
    } finally {
      setLoading(false);
    }
  }, [submitting, loading]);

  // Global keyboard shortcuts for Trainer mode (ignored while typing, with modifiers, or under a dialog)
  useEffect(() => {
    if (mode !== "trainer" || !sessionState?.question) return;

    const onKeyDown = (e: KeyboardEvent) => {
      if (e.isComposing || e.metaKey || e.ctrlKey || e.altKey || isModalOpen()) return;
      const isCompleted = sessionState.session_state === "question_completed";
      const canAdvance = acceptsPlainShortcut(e) ||
        (e.target instanceof HTMLInputElement && e.target.disabled && !e.defaultPrevented && !e.repeat);

      if (e.key === "Enter" && isCompleted && canAdvance && !submitting && !loading) {
        e.preventDefault();
        handleNextQuestion();
        return;
      }

      if (!acceptsPlainShortcut(e) || isCompleted || submitting || loading) return;

      if (!sessionState.question?.is_spelling) {
        const keyMap: Record<string, number> = { "1": 0, a: 0, "2": 1, b: 1, "3": 2, c: 2, "4": 3, d: 3 };
        const idx = keyMap[e.key.toLowerCase()];
        if (idx !== undefined) {
          e.preventDefault();
          handleChoiceSelect(idx);
          return;
        }
      }

      if ((e.key === "h" || e.key === "H") && sessionState.tries === 0) {
        const firstAvail = sessionState.question?.available_hints?.find(
          (ht) => !activeHints.some((ah) => ah.hint_type === ht)
        );
        if (firstAvail) {
          e.preventDefault();
          handleUseHint(firstAvail);
        }
      }
    };

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [mode, sessionState, submitting, loading, activeHints, handleChoiceSelect, handleNextQuestion, handleUseHint]);

  // Auto-focus spelling input when a spelling question loads
  useEffect(() => {
    if (sessionState?.question?.is_spelling && spellingRef.current) {
      spellingRef.current.focus();
    }
  }, [sessionState?.question]);

  // Spelling Bee Handlers
  const handleStartBee = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    const id = Number(listIdInput.trim());
    if (!id || id <= 0) {
      setError("Enter a valid Vocabulary List ID to start a Spelling Bee.");
      return;
    }
    setLoading(true);
    setError(null);
    setBeeGuessInput("");
    try {
      const st = await getBridge().trainer.StartBee(id, "");
      setBeeState(st);
    } catch (err) {
      setError(formatError(err));
    } finally {
      setLoading(false);
    }
  };

  const handleGuessBee = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!beeGuessInput.trim() || submitting) return;
    setSubmitting(true);
    setError(null);
    try {
      const st = await getBridge().trainer.GuessBee(beeGuessInput);
      setBeeState(st);
      if (st?.correct) {
        setBeeGuessInput("");
      }
    } catch (err) {
      setError(formatError(err));
    } finally {
      setSubmitting(false);
    }
  };

  const handleSurrenderBee = async () => {
    if (submitting) return;
    setSubmitting(true);
    setError(null);
    try {
      const st = await getBridge().trainer.SurrenderBee();
      setBeeState(st);
    } catch (err) {
      setError(formatError(err));
    } finally {
      setSubmitting(false);
    }
  };

  const handleNextBeeWord = async () => {
    try {
      const st = await getBridge().trainer.NextBeeWord();
      setBeeState(st);
      setBeeGuessInput("");
      beeGuessRef.current?.focus();
    } catch (err) {
      setError(formatError(err));
    }
  };

  // Quiz Handlers
  const handleStartQuiz = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!quizIdInput.trim()) {
      setError("Enter a Quiz ID or Activity ID to start.");
      return;
    }
    setLoading(true);
    setError(null);
    setQuizResults(null);
    setQuizSelectedNonce("");
    try {
      const st = await getBridge().trainer.StartQuiz(quizIdInput.trim(), "", "");
      setQuizState(st);
    } catch (err) {
      setError(formatError(err));
    } finally {
      setLoading(false);
    }
  };

  const handleSubmitQuiz = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!quizSelectedNonce || submitting) return;
    setSubmitting(true);
    setError(null);
    try {
      const st = await getBridge().trainer.SubmitQuizAnswer(quizSelectedNonce);
      setQuizState(st);
      setQuizSelectedNonce("");
      if (st?.is_complete && st.activity_id) {
        const res = await getBridge().trainer.GetQuizResults(st.activity_id);
        setQuizResults(res);
      }
    } catch (err) {
      setError(formatError(err));
    } finally {
      setSubmitting(false);
    }
  };

  const q = sessionState?.question;
  const feedback = sessionState?.last_feedback;
  const isCompleted = sessionState?.session_state === "question_completed";
  const isTrialOver = sessionState?.session_state === "trial_over";
  const answeredInRound = sessionState?.round?.played_count || 0;
  const questionNumber = answeredInRound + (isCompleted ? 0 : 1);

  return (
    <div className="max-w-[860px] mx-auto space-y-5">
      <h1 className="font-word text-title font-semibold tracking-[-0.03em]">Practice</h1>
      {/* Mode switcher & session source */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <Tabs
          label="Practice mode"
          value={mode}
          onChange={(id) => {
            setMode(id);
            setError(null);
          }}
          items={[
            { id: "trainer", label: "VocabTrainer", icon: Sparkles },
            { id: "bee", label: "Spelling Bee", icon: SpellCheck },
            { id: "quiz", label: "Quiz", icon: ListChecks },
          ]}
        />

        {mode === "trainer" && (
          <form
            className="flex items-center gap-2"
            onSubmit={(e) => {
              e.preventDefault();
              startTrainer();
            }}
          >
            <label htmlFor="practice-list-id" className="eyebrow text-ink-subtle hidden sm:block">
              Source
            </label>
            <div className="w-40">
              <TextInput
                id="practice-list-id"
                inputSize="sm"
                inputMode="numeric"
                value={listIdInput}
                onChange={(e) => setListIdInput(e.target.value)}
                placeholder="Adaptive (all)"
                aria-label="List ID for practice"
                className="font-data"
              />
            </div>
            <Button size="sm" type="submit" icon={RotateCcw} disabled={loading || submitting}>
              {listIdInput.trim() ? "Practice List" : "Start Trainer"}
            </Button>
          </form>
        )}
      </div>

      {error && (
        <ErrorBanner
          message={error}
          onRetry={mode === "trainer" ? () => startTrainer() : undefined}
          onDismiss={() => setError(null)}
        />
      )}

      {/* ==================== MODE 1: VOCABTRAINER & LIST PRACTICE ==================== */}
      {mode === "trainer" && (
        <div className="space-y-4">
          {/* Session status strip */}
          {sessionState?.active && (
            <div className="flex flex-wrap items-center justify-between gap-x-5 gap-y-2 px-1 py-3 border-b border-line">
              <div className="flex flex-wrap items-center gap-x-4 gap-y-2 min-w-0">
                {sessionState.round && (
                  <div className="flex items-center gap-3">
                    <span className="font-nav text-meta text-ink-muted whitespace-nowrap">
                      Round <strong className="font-data text-ink">{sessionState.round.number}</strong>
                      <span className="mx-1.5 text-ink-subtle">·</span>
                      Question <strong className="font-data text-ink">{questionNumber}</strong>
                      <span className="text-ink-subtle">/{ROUND_LENGTH}</span>
                    </span>
                    <RoundMeter answered={answeredInRound} current={isCompleted ? -1 : questionNumber - 1} />
                  </div>
                )}
                {sessionState.round && sessionState.round.streak > 0 && (
                  <Badge tone="warning" icon={Flame} title="Correct answers in a row">
                    {sessionState.round.streak} streak
                  </Badge>
                )}
                {sessionState.game?.name && (
                  <span className="font-nav text-meta text-accent-ink truncate max-w-[18rem]">
                    List: {sessionState.game.name} ({Math.round((sessionState.game.progress || 0) * 100)}%)
                  </span>
                )}
              </div>

              <div className="flex items-center gap-2.5">
                {submitting && (
                  <span
                    data-testid="pacing-indicator"
                    role="status"
                    className="inline-flex items-center gap-1.5 font-nav text-meta text-accent-ink"
                  >
                    <Loader2 className="w-3.5 h-3.5 animate-spin" aria-hidden />
                    <span>Submitting answer…</span>
                  </span>
                )}
                {q?.mode && <Badge className="uppercase tracking-wider">{q.mode}</Badge>}
                {q && q.point_value > 0 && (
                  <span className="font-data text-meta font-semibold text-ink tabular-nums" title="Points for a first-try answer">
                    +{q.point_value} pts
                  </span>
                )}
              </div>
            </div>
          )}

          {/* Trial over */}
          {isTrialOver && (
            <EmptyState
              icon={LogIn}
              title="Sign in to continue your training session"
              description="Vocabulary.com requires an authenticated subscriber account to continue playing beyond the free preview round."
              action={
                <Button variant="primary" icon={LogIn} onClick={onRequestLogin}>
                  Sign In to Continue
                </Button>
              }
            />
          )}

          {/* Loading skeleton */}
          {loading && !q && (
            <Card pad="lg" className="space-y-5" aria-busy>
              <span className="sr-only" role="status">
                Preparing question slide...
              </span>
              <Skeleton className="h-3 w-24" />
              <Skeleton className="h-6 w-2/3" />
              <div className="space-y-2.5 pt-1">
                {[0, 1, 2, 3].map((i) => (
                  <Skeleton key={i} className="h-13 w-full rounded-[0.75rem]" />
                ))}
              </div>
            </Card>
          )}

          {/* Active question */}
          {q && !isTrialOver && (
            <Card
              pad="none"
              key={`${q.turn}-${q.type}`}
              className="study-sheet animate-enter overflow-hidden"
              aria-labelledby="practice-question-heading"
              aria-busy={submitting || loading}
            >
              <div className="study-content space-y-5">
                <div className="flex items-start justify-between gap-4">
                  <div className="space-y-2.5 min-w-0">
                    {q.type_label && <p className="eyebrow text-accent-ink">{q.type_label}</p>}
                  </div>
                  {(q.audio_url || q.target_word) && (
                    <AudioButton audioUrl={q.audio_url} word={q.target_word} label="Audio" size="sm" />
                  )}
                </div>

                {/* Passage / sentence context */}
                {q.sentence_text && (
                  <figure className="relative pl-5 border-l-[3px] border-accent-line">
                    <blockquote className="font-read text-read-lg text-ink">
                      <RichText spans={q.sentence_spans} fallback={q.sentence_text} emphasis="mark" />
                    </blockquote>
                    {q.source_text && (
                      <figcaption className="mt-2 font-note text-meta text-ink-subtle">— {q.source_text}</figcaption>
                    )}
                  </figure>
                )}

                {/* Single-image prompt (type G) */}
                {q.prompt_image_url && (
                  <div className="flex justify-center">
                    <img
                      src={q.prompt_image_url}
                      alt={q.target_word || "Vocabulary prompt"}
                      className="max-h-64 rounded-[0.75rem] border border-line object-contain bg-sunken"
                    />
                  </div>
                )}

                <h2
                  id="practice-question-heading"
                  className="question-prompt text-ink"
                >
                  <RichText
                    spans={q.instructions_spans}
                    emphasis="word"
                    fallback={
                      q.instructions_text ||
                      (q.is_spelling
                        ? "Spell the word you hear in the audio prompt:"
                        : `Select the best match for ${q.target_word || "this word"}:`)
                    }
                  />
                </h2>

                {/* Lifeline results */}
                {activeHints.length > 0 && (
                  <div className="space-y-2">
                    {activeHints.map((h, i) => (
                      <div
                        key={i}
                        className="animate-enter flex gap-3 px-4 py-3 rounded-[0.7rem] border border-warning-line/70 bg-warning-soft"
                      >
                        <Lightbulb className="w-4 h-4 mt-0.5 shrink-0 text-warning-ink" aria-hidden />
                        <div className="space-y-1 min-w-0">
                          <span className="block font-note text-meta text-warning-ink">Hint · {hintLabel(h.hint_type)}</span>
                          {h.definition_text && (
                            <p className="font-read text-[1rem] text-ink">
                              <RichText spans={h.definition_spans} fallback={h.definition_text} />
                            </p>
                          )}
                          {h.example_sentence && (
                            <p className="font-read text-[1rem] italic text-ink">
                              "{h.example_sentence}" {h.example_source ? `— ${h.example_source}` : ""}
                            </p>
                          )}
                          {h.eliminated_nonces && h.eliminated_nonces.length > 0 && (
                            <p className="text-ui text-ink-muted">Two incorrect choices have been eliminated.</p>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                )}

                {/* Answer area: spelling vs 4-image grid vs multiple choice */}
                {q.is_spelling ? (
                  <form onSubmit={handleSpellingSubmit} className="flex flex-col sm:flex-row items-stretch gap-2.5">
                    <input
                      ref={spellingRef}
                      type="text"
                      value={spellingInput}
                      onChange={(e) => setSpellingInput(e.target.value)}
                      disabled={isCompleted || submitting}
                      placeholder="Type the word…"
                      aria-label="Word spelling input"
                      autoComplete="off"
                      autoCapitalize="off"
                      spellCheck={false}
                      className={cx(
                        "flex-1 min-w-0 h-14 px-4 rounded-control border bg-sunken font-ui text-[1.375rem] tracking-[0.02em] text-ink placeholder:text-ink-subtle placeholder:tracking-normal placeholder:text-[1rem]",
                        "outline-none transition-[border-color,box-shadow] focus:border-accent focus:ring-3 focus:ring-focus/20",
                        feedback && !feedback.correct && !isCompleted ? "border-danger-line animate-shake" : "border-line"
                      )}
                    />
                    <Button
                      type="submit"
                      variant="primary"
                      size="xl"
                      loading={submitting}
                      disabled={isCompleted || !spellingInput.trim()}
                    >
                      Spell It
                    </Button>
                  </form>
                ) : q.is_image_grid ? (
                  <div className="grid grid-cols-2 gap-3 max-w-[640px] mx-auto" role="group" aria-label="Image answer choices">
                    {(q.choices || []).map((ch, idx) => {
                      const status = selectedChoices[idx];
                      const disabled = isCompleted || submitting || Boolean(ch.eliminated) || Boolean(status);
                      return (
                        <button
                          key={ch.nonce || idx}
                          type="button"
                          disabled={disabled}
                          aria-pressed={pendingChoice === idx || Boolean(status)}
                          onClick={() => handleChoiceSelect(idx)}
                          aria-label={`Choice ${idx + 1}${ch.eliminated ? " (eliminated)" : ""}${status ? ` (${status})` : ""}`}
                          className={cx(
                            "group relative aspect-[16/10] max-h-44 rounded-control border-2 overflow-hidden bg-sunken",
                            "transition-[border-color,opacity,transform,box-shadow] duration-200 ease-out-soft",
                            status === "correct"
                              ? "border-success ring-4 ring-success/20 animate-pop"
                              : status === "wrong"
                              ? "border-danger opacity-60 animate-shake"
                              : ch.eliminated
                              ? "border-line opacity-25 grayscale cursor-not-allowed"
                              : isCompleted
                              ? "border-line opacity-60"
                              : "border-line hover:border-accent hover:-translate-y-0.5 hover:shadow-raised"
                          )}
                        >
                          {ch.image_url ? (
                            <img src={ch.image_url} alt="" className="w-full h-full object-contain" />
                          ) : (
                            <div className="flex items-center justify-center h-full text-ui text-ink-muted">
                              {ch.text || `Image ${idx + 1}`}
                            </div>
                          )}
                          <Kbd size="md" className="absolute top-2.5 left-2.5 shadow-card">
                            {idx + 1}
                          </Kbd>
                          {status === "correct" && (
                            <span className="absolute top-2.5 right-2.5 bg-success text-on-success rounded-full p-1 shadow-raised">
                              <CheckCircle2 className="w-4 h-4" aria-hidden />
                            </span>
                          )}
                          {status === "wrong" && (
                            <span className="absolute top-2.5 right-2.5 bg-danger text-on-danger rounded-full p-1 shadow-raised">
                              <XCircle className="w-4 h-4" aria-hidden />
                            </span>
                          )}
                        </button>
                      );
                    })}
                  </div>
                ) : (
                  <div className="grid grid-cols-1 gap-2.5" role="group" aria-label="Answer choices">
                    {(q.choices || []).map((ch, idx) => {
                      const status = selectedChoices[idx];
                      const disabled = isCompleted || submitting || Boolean(ch.eliminated) || Boolean(status);
                      return (
                        <button
                          key={ch.nonce || idx}
                          type="button"
                          disabled={disabled}
                          aria-pressed={pendingChoice === idx || Boolean(status)}
                          onClick={() => handleChoiceSelect(idx)}
                          className={cx(
                            "answer-choice group w-full flex flex-wrap gap-x-3.5 gap-y-2 px-4 py-3 rounded-control border text-left",
                            "transition-[background-color,border-color,opacity,transform,box-shadow] duration-200 ease-out-soft",
                            status === "correct"
                              ? "border-success-line bg-success-soft text-ink animate-pop"
                              : status === "wrong"
                              ? "border-danger-line bg-danger-soft/70 text-ink-muted animate-shake"
                              : ch.eliminated
                              ? "border-dashed border-line opacity-40 cursor-not-allowed"
                              : isCompleted
                              ? "border-line bg-surface opacity-55"
                              : pendingChoice === idx
                              ? "border-accent-line bg-accent-soft text-ink"
                              : "border-line bg-surface hover:border-accent-line hover:bg-accent-soft/35"
                          )}
                        >
                          <Kbd
                            size="md"
                            tone={status === "correct" ? "success" : status === "wrong" ? "danger" : "interactive"}
                          >
                            {idx + 1}
                          </Kbd>
                          <span
                            className={cx(
                              "flex-1 min-w-0 font-read text-[1.125rem] leading-snug break-words",
                              (status === "wrong" || ch.eliminated) && "line-through decoration-1"
                            )}
                          >
                            {ch.text}
                          </span>
                          {pendingChoice === idx && <Loader2 className="w-4 h-4 animate-spin text-accent-ink" aria-hidden />}
                          {status === "correct" && (
                            <span className="inline-flex items-center gap-1 font-nav text-meta font-semibold text-success-ink shrink-0">
                              <CheckCircle2 className="w-4 h-4" aria-hidden />
                              <span>Correct</span>
                            </span>
                          )}
                          {status === "wrong" && (
                            <span className="inline-flex items-center gap-1 font-nav text-meta font-semibold text-danger-ink shrink-0">
                              <XCircle className="w-4 h-4" aria-hidden />
                              <span>Try again</span>
                            </span>
                          )}
                        </button>
                      );
                    })}
                  </div>
                )}
              </div>

              {/* Lifelines & keyboard legend */}
              <div className="flex flex-wrap items-center justify-between gap-3 px-6 md:px-8 py-3 border-t border-line bg-sunken/50">
                <div className="flex flex-wrap items-center gap-1.5 min-h-7">
                  {sessionState.tries === 0 && !isCompleted && q.available_hints && q.available_hints.length > 0 && (
                    <>
                      <span className="inline-flex items-center gap-1.5 mr-1 font-nav text-meta text-ink-subtle">
                        <Lightbulb className="w-3.5 h-3.5 text-warning" aria-hidden />
                        Lifelines
                      </span>
                      {q.available_hints.map((ht) => {
                        const alreadyUsed = activeHints.some((ah) => ah.hint_type === ht);
                        return (
                          <Button
                            key={ht}
                            size="sm"
                            variant="secondary"
                            disabled={alreadyUsed || submitting}
                            onClick={() => handleUseHint(ht)}
                          >
                            {hintLabel(ht)}
                          </Button>
                        );
                      })}
                    </>
                  )}
                </div>

                <div className="hidden sm:flex items-center gap-3 font-nav text-micro text-ink-subtle">
                  {!q.is_spelling && !isCompleted && (
                    <span className="inline-flex items-center gap-1">
                      <Kbd>1</Kbd>–<Kbd>4</Kbd> answer
                    </span>
                  )}
                  {sessionState.tries === 0 && !isCompleted && (q.available_hints?.length || 0) > 0 && (
                    <span className="inline-flex items-center gap-1">
                      <Kbd>H</Kbd> lifeline
                    </span>
                  )}
                  {isCompleted && (
                    <span className="inline-flex items-center gap-1">
                      <Kbd>Enter</Kbd> next question
                    </span>
                  )}
                </div>
              </div>
            </Card>
          )}

          <p className="sr-only" aria-live="polite">
            {feedback
              ? feedback.correct
                ? `Correct answer${feedback.points_earned ? `, ${feedback.points_earned} points` : ""}.`
                : "Incorrect answer, try again."
              : ""}
          </p>

          {/* Post-answer feedback & explanation */}
          {feedback && (
            <section
              aria-label="Answer feedback"
              key={`${feedback.attempt}-${feedback.correct}`}
              className={cx(
                "animate-enter rounded-card border shadow-card overflow-hidden",
                feedback.correct ? "border-success-line/70 bg-success-soft/50" : "border-warning-line/70 bg-warning-soft/60"
              )}
            >
              <div className="flex flex-wrap items-center justify-between gap-3 px-5 py-4">
                <div className="flex items-center gap-3 min-w-0">
                  <span
                    className={cx(
                      "w-9 h-9 rounded-full flex items-center justify-center shrink-0",
                      feedback.correct ? "bg-success text-on-success" : "bg-warning text-on-warning"
                    )}
                  >
                    {feedback.correct ? (
                      <CheckCircle2 className="w-5 h-5" aria-hidden />
                    ) : (
                      <HelpCircle className="w-5 h-5" aria-hidden />
                    )}
                  </span>
                  <div className="min-w-0">
                    <h3 className="font-nav text-[0.9375rem] font-semibold text-ink">
                      {feedback.correct
                        ? feedback.attempt === 1
                          ? `Correct on first try! (+${feedback.points_earned} pts)`
                          : "Correct!"
                        : `Attempt ${feedback.attempt}: Keep trying!`}
                    </h3>
                    <p className="font-nav text-micro text-ink-muted">
                      {feedback.evaluated_locally ? "Practice retry · keep going" : "Answer saved to your session"}
                    </p>
                  </div>
                </div>

                <div className="flex flex-wrap items-center gap-2">
                  {feedback.word && (
                    <>
                      <AudioButton audioUrl={feedback.audio_url} word={feedback.word} size="sm" label="Pronounce" />
                      <Button size="sm" variant="secondary" icon={BookOpen} onClick={() => onOpenWord(feedback.word)}>
                        Open "{feedback.word}"
                      </Button>
                    </>
                  )}
                  {isCompleted && (
                    <Button
                      variant="primary"
                      iconRight={ArrowRight}
                      onClick={handleNextQuestion}
                      disabled={loading}
                      kbd="↵"
                    >
                      Next Question
                    </Button>
                  )}
                </div>
              </div>

              {isCompleted && (feedback.sense_definition || feedback.short_blurb_text || feedback.long_blurb_text) && (
                <div className="px-5 pb-5 pt-4 space-y-3 border-t border-line/70 bg-surface/70">
                  {feedback.sense_definition && (
                    <p className="font-read text-[1.0625rem] text-ink">
                      In this question, <strong className="font-word font-semibold">{feedback.word}</strong> is{" "}
                      {feedback.part_of_speech ? `a ${feedback.part_of_speech} that means ` : ""}
                      <span className="font-semibold">{feedback.sense_definition}</span>.
                    </p>
                  )}
                  {feedback.short_blurb_text && (
                    <p className="font-read text-[1rem] leading-relaxed text-ink">
                      <RichText spans={feedback.short_blurb_spans} fallback={feedback.short_blurb_text} />
                    </p>
                  )}
                  {feedback.long_blurb_text && (
                    <p className="font-read text-[0.9375rem] leading-relaxed text-ink-muted max-w-[68ch]">
                      <RichText spans={feedback.long_blurb_spans} fallback={feedback.long_blurb_text} />
                    </p>
                  )}
                </div>
              )}
            </section>
          )}
        </div>
      )}

      {/* ==================== MODE 2: SPELLING BEE ==================== */}
      {mode === "bee" && (
        <div className="space-y-4">
          <Card pad="md">
            <form onSubmit={handleStartBee} className="flex flex-wrap items-end gap-3">
              <Field label="Vocabulary list ID" htmlFor="bee-list-id" className="w-64">
                <TextInput
                  id="bee-list-id"
                  inputMode="numeric"
                  value={listIdInput}
                  onChange={(e) => setListIdInput(e.target.value)}
                  placeholder="e.g. 285407"
                  className="font-data"
                />
              </Field>
              <Button type="submit" variant="primary" icon={Play} loading={loading}>
                Start Spelling Bee
              </Button>
              <p className="basis-full text-meta text-ink-subtle">
                You can also start a Spelling Bee from any list in My Lists.
              </p>
            </form>
          </Card>

          {beeState?.active && (
            <Card pad="none" className="animate-enter overflow-hidden">
              <div className="flex flex-wrap items-center justify-between gap-4 px-6 py-4 border-b border-line">
                <div className="space-y-1">
                  <p className="eyebrow text-accent-ink">Spelling Bee · List #{beeState.list_id}</p>
                  {beeState.score && (
                    <p className="font-nav text-meta text-ink-muted">
                      Correct <strong className="font-data text-ink">{beeState.score.correct}</strong>
                      <span className="mx-1.5 text-ink-subtle">·</span>
                      Played <strong className="font-data text-ink">{beeState.score.played}</strong>
                      <span className="mx-1.5 text-ink-subtle">·</span>
                      Remaining <strong className="font-data text-ink">{beeState.score.remaining}</strong>
                    </p>
                  )}
                </div>
                <AudioButton audioUrl={beeState.audio_url} label="Play Word Audio" variant="soft" />
              </div>

              <div className="p-6 space-y-5">
                {beeState.formatted_example && (
                  <blockquote className="pl-5 border-l-[3px] border-accent-line font-read text-read-lg text-ink">
                    "{beeState.formatted_example}"
                  </blockquote>
                )}

                {beeState.definitions && beeState.definitions.length > 0 && (
                  <ul className="space-y-2">
                    {beeState.definitions.map((d, i) => (
                      <li key={i} className="flex items-baseline gap-2.5">
                        <Badge tone="accent" className="shrink-0">
                          {d.pos}
                        </Badge>
                        <span className="font-read text-[1.0625rem] text-ink">{d.text}</span>
                      </li>
                    ))}
                  </ul>
                )}

                {!beeState.completed_word ? (
                  <form onSubmit={handleGuessBee} className="flex flex-wrap items-stretch gap-2.5">
                    <input
                      ref={beeGuessRef}
                      type="text"
                      value={beeGuessInput}
                      onChange={(e) => setBeeGuessInput(e.target.value)}
                      placeholder="Spell the word…"
                      aria-label="Spelling Bee guess"
                      autoComplete="off"
                      autoCapitalize="off"
                      spellCheck={false}
                      className={cx(
                        "flex-1 min-w-56 h-12 px-4 rounded-[0.75rem] border bg-sunken font-word text-[1.25rem] tracking-[0.04em] text-ink placeholder:text-ink-subtle placeholder:tracking-normal placeholder:text-[1rem]",
                        "outline-none transition-[border-color,box-shadow] focus:border-accent focus:ring-3 focus:ring-focus/20",
                        beeState.guesses > 0 ? "border-warning-line" : "border-line"
                      )}
                    />
                    <Button
                      type="submit"
                      variant="primary"
                      size="xl"
                      loading={submitting}
                      disabled={!beeGuessInput.trim()}
                    >
                      Submit Guess
                    </Button>
                    {beeState.can_surrender && (
                      <Button
                        variant="danger-outline"
                        size="xl"
                        icon={Flag}
                        onClick={handleSurrenderBee}
                        disabled={submitting}
                      >
                        Surrender
                      </Button>
                    )}
                  </form>
                ) : (
                  <div
                    role="status"
                    className={cx(
                      "animate-pop flex flex-wrap items-center justify-between gap-3 px-4 py-3.5 rounded-[0.75rem] border",
                      beeState.correct ? "border-success-line bg-success-soft" : "border-line bg-sunken"
                    )}
                  >
                    <p className="font-nav text-ui font-semibold text-ink">
                      {beeState.correct ? "Correct!" : "Word Revealed:"}{" "}
                      <span className="font-word text-[1.375rem] text-accent-ink">{beeState.revealed_word}</span>
                    </p>
                    {beeState.has_next_word && (
                      <Button variant="primary" iconRight={ArrowRight} onClick={handleNextBeeWord}>
                        Next Word
                      </Button>
                    )}
                  </div>
                )}

                {beeState.guesses > 0 && !beeState.completed_word && (
                  <p role="status" className="font-nav text-meta text-warning-ink">
                    Incorrect guess #{beeState.guesses}.
                    {beeState.distance
                      ? ` Hint: you are ${beeState.distance} letter(s) off.`
                      : " Listen carefully and try again!"}
                  </p>
                )}
              </div>
            </Card>
          )}

          {!beeState?.active && !loading && (
            <EmptyState
              icon={Volume2}
              title="Hear it, then spell it"
              description="The Spelling Bee plays each word aloud with an example sentence and definition. Pick a list to begin."
            />
          )}
        </div>
      )}

      {/* ==================== MODE 3: QUIZ ==================== */}
      {mode === "quiz" && (
        <div className="space-y-4">
          <Card pad="md">
            <form onSubmit={handleStartQuiz} className="flex flex-wrap items-end gap-3">
              <Field label="Quiz or activity ID" htmlFor="quiz-id" className="w-64">
                <TextInput
                  id="quiz-id"
                  value={quizIdInput}
                  onChange={(e) => setQuizIdInput(e.target.value)}
                  placeholder="Enter Quiz ID"
                  className="font-data"
                />
              </Field>
              <Button type="submit" variant="primary" icon={Play} loading={loading}>
                Start Quiz
              </Button>
            </form>
          </Card>

          {quizState?.active && !quizState.is_complete && quizState.question && (
            <Card as="form" pad="lg" onSubmit={handleSubmitQuiz} className="animate-enter space-y-5">
              <div className="space-y-2">
                <div className="flex items-center justify-between gap-3 font-nav text-meta text-ink-muted">
                  <span>
                    Question <strong className="font-data text-ink">{quizState.question_number}</strong> of{" "}
                    <strong className="font-data text-ink">{quizState.question_total}</strong>
                  </span>
                  <span className="text-ink-subtle">Single attempt · graded</span>
                </div>
                <Meter
                  label="Quiz progress"
                  value={
                    quizState.question_total
                      ? ((quizState.question_number - 1) / quizState.question_total) * 100
                      : 0
                  }
                />
              </div>
              <h3 className="font-read text-[1.375rem] leading-snug text-ink">
                <RichText
                  spans={quizState.question.instructions_spans}
                  fallback={quizState.question.instructions_text}
                  emphasis="word"
                />
              </h3>
              <div className="space-y-2" role="radiogroup" aria-label="Quiz choices">
                {(quizState.question.choices || []).map((ch) => {
                  const selected = quizSelectedNonce === ch.nonce;
                  return (
                    <label
                      key={ch.nonce}
                      className={cx(
                        "flex items-center gap-3 px-4 py-3.5 rounded-[0.8rem] border cursor-pointer transition-[background-color,border-color] duration-150",
                        "has-[:focus-visible]:ring-3 has-[:focus-visible]:ring-focus/25",
                        selected
                          ? "border-accent bg-accent-soft"
                          : "border-line bg-surface hover:border-accent-line hover:bg-accent-soft/30"
                      )}
                    >
                      <input
                        type="radio"
                        name="quiz-choice"
                        disabled={submitting}
                        checked={selected}
                        onChange={() => setQuizSelectedNonce(ch.nonce)}
                        className="w-4 h-4 accent-[var(--c-accent)]"
                      />
                      <span className="font-read text-[1.0625rem] text-ink">{ch.text}</span>
                    </label>
                  );
                })}
              </div>
              <div className="flex justify-end">
                <Button type="submit" variant="primary" loading={submitting} disabled={!quizSelectedNonce}>
                  {submitting ? "Submitting…" : "Submit Final Answer"}
                </Button>
              </div>
            </Card>
          )}

          {quizState?.is_complete && (
            <Card pad="lg" className="animate-enter space-y-4">
              <h3 className="font-nav text-[1.125rem] font-semibold text-ink">Quiz Complete</h3>
              {quizResults && quizResults.length > 0 && (
                <ul className="divide-y divide-line border border-line rounded-[0.75rem] overflow-hidden">
                  {quizResults.map((r) => (
                    <li key={r.question_number} className="px-4 py-3 flex items-center justify-between gap-4 bg-surface">
                      <span className="text-ui text-ink">
                        <span className="font-data text-ink-subtle mr-2">Q{r.question_number}</span>
                        {r.question_text}
                      </span>
                      <span className="font-data text-meta text-ink-muted shrink-0">
                        Choice #{r.user_choice} · {r.people_count} responses
                      </span>
                    </li>
                  ))}
                </ul>
              )}
            </Card>
          )}

          {!quizState?.active && !quizState?.is_complete && !loading && (
            <EmptyState
              icon={ListChecks}
              title="Graded quizzes"
              description="Quizzes assigned on Vocabulary.com are single-attempt. Enter the quiz or activity ID to begin."
            />
          )}
        </div>
      )}
    </div>
  );
};

const RoundMeter: React.FC<{ answered: number; current: number }> = ({ answered, current }) => (
  <div
    className="flex items-center gap-[3px]"
    role="progressbar"
    aria-label="Round progress"
    aria-valuemin={0}
    aria-valuemax={ROUND_LENGTH}
    aria-valuenow={Math.min(answered, ROUND_LENGTH)}
  >
    {Array.from({ length: ROUND_LENGTH }, (_, i) => (
      <span
        key={i}
        className={cx(
          "h-1.5 w-3.5 rounded-full transition-colors duration-300",
          i < answered ? "bg-accent" : i === current ? "bg-accent/35 ring-1 ring-accent/60" : "bg-line"
        )}
      />
    ))}
  </div>
);
