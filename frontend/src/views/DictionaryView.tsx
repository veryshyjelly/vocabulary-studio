import React, { useState, useEffect, useCallback, useRef } from "react";
import {
  Search,
  Shuffle,
  Plus,
  BookmarkPlus,
  ExternalLink,
  Trash2,
  GitBranch,
  Quote,
  Sparkles,
  BookText,
  ChevronDown,
  ListChecks,
  SearchX,
} from "lucide-react";
import type {
  WordDetailDTO,
  AutocompleteSuggestion,
  CorpusSentenceDTO,
  AccountStatusDTO,
  ProfileListSummary,
} from "../lib/bridge";
import { getBridge, formatError } from "../lib/bridge";
import { acceptsPlainShortcut } from "../lib/keys";
import { AudioButton, CopyButton, ErrorBanner } from "../components/Shared";
import {
  Badge,
  Button,
  Card,
  Chip,
  EmptyState,
  Kbd,
  Meter,
  SectionTitle,
  Skeleton,
  Spinner,
  cx,
  useToast,
} from "../components/ui";

const CORPUS_DOMAINS = [
  { code: "", label: "All Sources" },
  { code: "F", label: "Literature" },
  { code: "N", label: "News" },
  { code: "A", label: "Arts & Culture" },
  { code: "M", label: "Science & Med" },
  { code: "B", label: "Business" },
  { code: "T", label: "Technology" },
  { code: "S", label: "Sports" },
];

const PRIORITIES = [
  { val: -1, label: "Low" },
  { val: 0, label: "Normal" },
  { val: 1, label: "High" },
];

const COLLAPSED_SENSE_COUNT = 4;

export const DictionaryView: React.FC<{
  account: AccountStatusDTO | null;
  initialWord?: string;
  preferredAccent?: string;
  focusSearchToken?: number;
  onOpenList: (listId: number) => void;
  onRequestLogin: () => void;
}> = ({ account, initialWord = "", preferredAccent = "US", focusSearchToken = 0, onOpenList, onRequestLogin }) => {
  const toast = useToast();
  const [query, setQuery] = useState(initialWord);
  const [suggestions, setSuggestions] = useState<AutocompleteSuggestion[]>([]);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [activeSuggestionIdx, setActiveSuggestionIdx] = useState(-1);

  const [detail, setDetail] = useState<WordDetailDTO | null>(null);
  const [examples, setExamples] = useState<CorpusSentenceDTO[]>([]);
  const [selectedDomain, setSelectedDomain] = useState("");
  const [history, setHistory] = useState<string[]>([]);
  const [showAllSenses, setShowAllSenses] = useState(false);

  const [loading, setLoading] = useState(false);
  const [examplesLoading, setExamplesLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Add to list popover state
  const [showAddToList, setShowAddToList] = useState(false);
  const [userLists, setUserLists] = useState<ProfileListSummary[]>([]);
  const addToListRef = useRef<HTMLDivElement | null>(null);

  const searchInputRef = useRef<HTMLInputElement | null>(null);
  const latestLookupId = useRef(0);
  const latestAutocompleteId = useRef(0);

  const refreshHistory = useCallback(async () => {
    try {
      const h = await getBridge().dictionary.GetSearchHistory();
      setHistory(h || []);
    } catch {
      // ignore
    }
  }, []);

  const lookupWord = useCallback(
    async (targetWord: string) => {
      const trimmed = targetWord.trim();
      if (!trimmed) return;

      const reqId = ++latestLookupId.current;
      setShowSuggestions(false);
      setQuery(trimmed);
      setLoading(true);
      setError(null);
      setSelectedDomain("");
      setShowAddToList(false);

      try {
        const res = await getBridge().dictionary.Lookup(trimmed);
        if (reqId !== latestLookupId.current) return; // Suppress stale response
        if (res) {
          setDetail(res);
          setExamples(res.examples || []);
          setShowAllSenses(false);
        }
        refreshHistory();
      } catch (err) {
        if (reqId !== latestLookupId.current) return;
        setError(formatError(err));
      } finally {
        if (reqId === latestLookupId.current) {
          setLoading(false);
        }
      }
    },
    [refreshHistory]
  );

  useEffect(() => {
    refreshHistory();
    if (initialWord.trim()) {
      lookupWord(initialWord);
    } else if (!detail) {
      lookupWord("callow");
    }
  }, [initialWord]);

  useEffect(() => {
    if (focusSearchToken > 0) {
      searchInputRef.current?.focus();
      searchInputRef.current?.select();
    }
  }, [focusSearchToken]);

  // Global "/" shortcut to focus dictionary search
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "/" && acceptsPlainShortcut(e)) {
        e.preventDefault();
        searchInputRef.current?.focus();
        searchInputRef.current?.select();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  // Close the add-to-list popover on outside click or Escape
  useEffect(() => {
    if (!showAddToList) return;
    const onPointer = (e: MouseEvent) => {
      if (addToListRef.current && !addToListRef.current.contains(e.target as Node)) setShowAddToList(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setShowAddToList(false);
    };
    document.addEventListener("mousedown", onPointer);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onPointer);
      document.removeEventListener("keydown", onKey);
    };
  }, [showAddToList]);

  // Debounced autocomplete with stale-response suppression
  useEffect(() => {
    const trimmed = query.trim();
    if (!trimmed || !showSuggestions) {
      setSuggestions([]);
      return;
    }
    const reqId = ++latestAutocompleteId.current;
    const timer = setTimeout(async () => {
      try {
        const items = await getBridge().dictionary.Autocomplete(trimmed, 8);
        if (reqId === latestAutocompleteId.current) {
          setSuggestions(items || []);
          setActiveSuggestionIdx(-1);
        }
      } catch {
        if (reqId === latestAutocompleteId.current) {
          setSuggestions([]);
        }
      }
    }, 200);

    return () => clearTimeout(timer);
  }, [query, showSuggestions]);

  const handleDomainChange = async (domain: string) => {
    if (!detail?.entry?.word) return;
    setSelectedDomain(domain);
    setExamplesLoading(true);
    try {
      const list = await getBridge().dictionary.GetExamples(detail.entry.word, domain, 0, 10);
      setExamples(list || []);
    } catch (err) {
      setError(formatError(err));
    } finally {
      setExamplesLoading(false);
    }
  };

  const handleRandomWord = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await getBridge().dictionary.GetRandomWord();
      if (res?.entry?.word) {
        await lookupWord(res.entry.word);
      }
    } catch (err) {
      setError(formatError(err));
      setLoading(false);
    }
  };

  const handleStartLearning = async () => {
    if (!detail?.entry?.word) return;
    if (!account?.authenticated) {
      onRequestLogin();
      return;
    }
    try {
      await getBridge().progress.StartLearningWord(detail.entry.word);
      toast(`Added "${detail.entry.word}" to your learning queue.`);
      const wp = await getBridge().progress.GetWordProgress(detail.entry.word);
      setDetail((prev) => (prev ? { ...prev, progress: wp } : prev));
    } catch (err) {
      setError(formatError(err));
    }
  };

  const handleSetPriority = async (priority: number) => {
    if (!detail?.entry?.word) return;
    if (!account?.authenticated) {
      onRequestLogin();
      return;
    }
    try {
      const wp = await getBridge().progress.SetWordPriority(detail.entry.word, priority);
      setDetail((prev) => (prev ? { ...prev, progress: wp } : prev));
      toast(
        `Priority for "${detail.entry.word}" set to ${priority === 1 ? "High" : priority === -1 ? "Low" : "Normal"}.`
      );
    } catch (err) {
      setError(formatError(err));
    }
  };

  const handleOpenAddToList = async () => {
    if (!account?.authenticated) {
      onRequestLogin();
      return;
    }
    setShowAddToList((v) => !v);
    if (userLists.length === 0) {
      try {
        const lists = await getBridge().lists.GetUserLists("created");
        setUserLists(lists || []);
      } catch (err) {
        setError(formatError(err));
      }
    }
  };

  const handleAddWordToList = async (listId: number, listName: string) => {
    if (!detail?.entry?.word) return;
    try {
      await getBridge().lists.AddWordsToList(listId, [{ word: detail.entry.word, lang: "en" }]);
      setShowAddToList(false);
      toast(`Added "${detail.entry.word}" to "${listName}".`);
    } catch (err) {
      setError(formatError(err));
    }
  };

  const handleClearHistory = async () => {
    await getBridge().dictionary.ClearSearchHistory();
    setHistory([]);
  };

  const entry = detail?.entry;
  const progress = detail?.progress;
  const senses = entry?.senses || [];
  const visibleSenses = showAllSenses ? senses : senses.slice(0, COLLAPSED_SENSE_COUNT);
  const maxFamilyFreq = Math.max(0.01, ...(entry?.word_family || []).map((f) => f.freq || 0));
  const listboxOpen = showSuggestions && suggestions.length > 0;

  // Sort pronunciations so preferredAccent comes first
  const sortedPronunciations = [...(entry?.pronunciations || [])].sort((a, b) => {
    if (a.region === preferredAccent && b.region !== preferredAccent) return -1;
    if (b.region === preferredAccent && a.region !== preferredAccent) return 1;
    return 0;
  });

  return (
    <div className="space-y-6">
      {/* Search */}
      <div className="space-y-3">
        <div className="flex items-center gap-2.5">
          <form
            role="search"
            onSubmit={(e) => {
              e.preventDefault();
              if (activeSuggestionIdx >= 0 && suggestions[activeSuggestionIdx]) {
                lookupWord(suggestions[activeSuggestionIdx].word);
              } else {
                lookupWord(query);
              }
            }}
            className="relative flex-1"
          >
            <Search
              className="w-4.5 h-4.5 text-ink-subtle absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none"
              aria-hidden
            />
            <input
              ref={searchInputRef}
              type="text"
              value={query}
              role="combobox"
              aria-expanded={listboxOpen}
              aria-controls="dictionary-suggestions"
              aria-autocomplete="list"
              aria-activedescendant={activeSuggestionIdx >= 0 ? `dict-sug-${activeSuggestionIdx}` : undefined}
              onFocus={() => setShowSuggestions(true)}
              onBlur={() => window.setTimeout(() => setShowSuggestions(false), 120)}
              onChange={(e) => {
                setQuery(e.target.value);
                setShowSuggestions(true);
              }}
              onKeyDown={(e) => {
                if (e.key === "ArrowDown" && suggestions.length > 0) {
                  e.preventDefault();
                  setActiveSuggestionIdx((prev) => (prev < suggestions.length - 1 ? prev + 1 : 0));
                } else if (e.key === "ArrowUp" && suggestions.length > 0) {
                  e.preventDefault();
                  setActiveSuggestionIdx((prev) => (prev > 0 ? prev - 1 : suggestions.length - 1));
                } else if (e.key === "Escape") {
                  setShowSuggestions(false);
                }
              }}
              placeholder="Search any word or phrase"
              aria-label="Dictionary search"
              autoComplete="off"
              spellCheck={false}
              className="w-full h-12 pl-11 pr-32 rounded-[0.8rem] border border-line bg-surface shadow-card font-word text-[1.0625rem] text-ink placeholder:font-ui placeholder:text-ui placeholder:text-ink-subtle hover:border-line-strong focus:border-accent focus:ring-3 focus:ring-focus/20 outline-none transition-[border-color,box-shadow]"
            />
            <div className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center gap-2">
              <Kbd aria-hidden>/</Kbd>
              <Button type="submit" size="sm" variant="primary">
                Look up
              </Button>
            </div>

            {listboxOpen && (
              <ul
                id="dictionary-suggestions"
                role="listbox"
                aria-label="Suggestions"
                className="animate-fade absolute z-30 left-0 right-0 mt-1.5 p-1 rounded-[0.8rem] border border-line bg-raised shadow-pop overflow-hidden"
              >
                {suggestions.map((sug, idx) => (
                  <li
                    key={sug.word}
                    id={`dict-sug-${idx}`}
                    role="option"
                    aria-selected={idx === activeSuggestionIdx}
                    onMouseDown={(e) => e.preventDefault()}
                    onClick={() => lookupWord(sug.word)}
                    onMouseEnter={() => setActiveSuggestionIdx(idx)}
                    className={cx(
                      "flex items-center justify-between gap-4 px-3 py-2 rounded-[0.55rem] cursor-pointer transition-colors",
                      idx === activeSuggestionIdx ? "bg-accent-soft" : "hover:bg-sunken"
                    )}
                  >
                    <div className="min-w-0">
                      <div className="flex items-baseline gap-2">
                        <span className="font-word font-semibold text-ink">{sug.word}</span>
                        {sug.ipa && <span className="font-data text-micro text-ink-subtle">{sug.ipa}</span>}
                      </div>
                      {sug.definition && (
                        <p className="font-read text-[0.9375rem] text-ink-muted truncate">{sug.definition}</p>
                      )}
                    </div>
                    {sug.learnable && (
                      <Badge tone="accent" className="shrink-0">
                        Learnable
                      </Badge>
                    )}
                  </li>
                ))}
              </ul>
            )}
          </form>

          <Button size="xl" icon={Shuffle} onClick={handleRandomWord} disabled={loading}>
            Random Word
          </Button>
        </div>

        {history.length > 0 && (
          <div className="flex items-center justify-between gap-3">
            <div className="flex items-center gap-1.5 flex-wrap min-w-0">
              <span className="eyebrow text-ink-subtle mr-1">Recent</span>
              {history.slice(0, 10).map((w) => (
                <Chip
                  key={w}
                  size="sm"
                  active={entry?.word?.toLowerCase() === w.toLowerCase()}
                  onClick={() => lookupWord(w)}
                >
                  {w}
                </Chip>
              ))}
            </div>
            <Button size="sm" variant="ghost" icon={Trash2} onClick={handleClearHistory} title="Clear search history">
              Clear
            </Button>
          </div>
        )}
      </div>

      {error && <ErrorBanner message={error} onRetry={() => lookupWord(query)} />}

      {loading && !entry && (
        <Card pad="lg" className="space-y-4" aria-busy>
          <span className="sr-only" role="status">
            Looking up "{query}"...
          </span>
          <Skeleton className="h-11 w-64" />
          <Skeleton className="h-5 w-40" />
          <Skeleton className="h-4 w-full max-w-2xl" />
          <Skeleton className="h-4 w-5/6 max-w-xl" />
        </Card>
      )}

      {!loading && !entry && !error && (
        <EmptyState icon={SearchX} title="No entry to show" description="Search for a word to see its definitions." />
      )}

      {entry && (
        <div key={entry.word} className={cx("space-y-6 animate-enter", loading && "opacity-60 transition-opacity")}>
          {/* Hero */}
          <Card pad="lg" className="space-y-6">
            <div className="flex flex-col lg:flex-row lg:items-start justify-between gap-5">
              <div className="space-y-3 min-w-0">
                <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
                  <h1 className="font-word text-[2.5rem] md:text-[3rem] leading-[1.05] font-semibold tracking-[-0.03em] text-ink break-words">
                    {entry.word}
                  </h1>
                  <div className="flex items-center gap-1.5">
                    {entry.learnable && <Badge tone="accent">Learnable</Badge>}
                    {progress?.progress && (
                      <Badge font="data">
                        Mastery {Math.round((progress.progress.progress || 0) * 100)}%
                      </Badge>
                    )}
                  </div>
                </div>

                {entry.word_forms && entry.word_forms.length > 0 && (
                  <p className="text-meta text-ink-subtle">
                    Other forms: <span className="font-word text-ink-muted">{entry.word_forms.join(" · ")}</span>
                  </p>
                )}

                <div className="flex flex-wrap items-center gap-2">
                  {sortedPronunciations.length > 0 ? (
                    sortedPronunciations.map((p, idx) => (
                      <div
                        key={idx}
                        className="inline-flex items-center gap-2 h-8 pl-2.5 pr-0.5 rounded-control border border-line bg-sunken"
                      >
                        <span className="eyebrow text-ink-subtle">{p.region}</span>
                        {p.ipa && <span className="font-data text-meta text-ink">{p.ipa}</span>}
                        {(p.audio_url || entry.word) && (
                          <AudioButton audioUrl={p.audio_url} word={entry.word} size="sm" variant="ghost" />
                        )}
                      </div>
                    ))
                  ) : (
                    <AudioButton word={entry.word} label="Pronounce" size="sm" />
                  )}
                </div>
              </div>

              <div className="flex flex-wrap items-center gap-2 relative shrink-0" ref={addToListRef}>
                {entry.learnable && (
                  <Button variant="primary" icon={Sparkles} onClick={handleStartLearning}>
                    Learn Word
                  </Button>
                )}
                <Button
                  icon={BookmarkPlus}
                  onClick={handleOpenAddToList}
                  aria-expanded={showAddToList}
                  aria-haspopup="dialog"
                >
                  Add to List
                </Button>
                <CopyButton text={`${entry.word}: ${entry.short_blurb || entry.senses?.[0]?.definition || ""}`} label="Copy" />

                {showAddToList && (
                  <div
                    role="dialog"
                    aria-label={`Add "${entry.word}" to a list`}
                    className="animate-dialog absolute right-0 top-11 z-20 w-72 rounded-[0.8rem] border border-line bg-raised shadow-pop p-2"
                  >
                    <p className="px-2 pt-1 pb-2 eyebrow text-ink-subtle">Add "{entry.word}" to list</p>
                    {userLists.length > 0 ? (
                      <ul className="max-h-56 overflow-y-auto">
                        {userLists.map((ul) => (
                          <li key={ul.wordlistid}>
                            <button
                              type="button"
                              onClick={() => handleAddWordToList(ul.wordlistid, ul.name)}
                              className="w-full flex items-center justify-between gap-2 px-2 py-2 rounded-[0.5rem] text-left hover:bg-sunken transition-colors"
                            >
                              <span className="truncate font-nav text-ui text-ink">{ul.name}</span>
                              <span className="flex items-center gap-2 shrink-0">
                                <span className="font-data text-micro text-ink-subtle">{ul.wordcount}</span>
                                <Plus className="w-3.5 h-3.5 text-accent-ink" aria-hidden />
                              </span>
                            </button>
                          </li>
                        ))}
                      </ul>
                    ) : (
                      <p className="px-2 pb-2 text-meta text-ink-muted">
                        No personal lists found. Create one in My Lists first.
                      </p>
                    )}
                  </div>
                )}
              </div>
            </div>

            {account?.authenticated && entry.learnable && (
              <div className="flex flex-wrap items-center gap-3">
                <span className="eyebrow text-ink-subtle" id="priority-label">
                  Study priority
                </span>
                <div
                  role="group"
                  aria-labelledby="priority-label"
                  className="inline-flex p-0.75 rounded-[0.6rem] bg-sunken border border-line"
                >
                  {PRIORITIES.map((p) => {
                    const active = (progress?.progress?.priority ?? 0) === p.val;
                    return (
                      <button
                        key={p.val}
                        type="button"
                        aria-pressed={active}
                        onClick={() => handleSetPriority(p.val)}
                        className={cx(
                          "h-6.5 px-3 rounded-[0.4rem] font-nav text-meta transition-[background-color,color,box-shadow] duration-150",
                          active ? "bg-raised text-ink font-medium shadow-raised" : "text-ink-muted hover:text-ink"
                        )}
                      >
                        {p.label}
                      </button>
                    );
                  })}
                </div>
              </div>
            )}

            {(entry.short_blurb || entry.long_blurb) && (
              <div className="space-y-3 max-w-[70ch] pt-1 border-t border-line">
                {entry.short_blurb && <p className="pt-5 font-read text-read-lg text-ink">{entry.short_blurb}</p>}
                {entry.long_blurb && <p className="font-read text-read text-ink-muted">{entry.long_blurb}</p>}
              </div>
            )}
          </Card>

          <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
            {/* Definitions */}
            <div className="lg:col-span-7 space-y-3">
              <SectionTitle icon={BookText}>Definitions & Senses ({senses.length})</SectionTitle>
              {senses.length > 0 ? (
                <Card pad="none">
                  <ol className="divide-y divide-line">
                    {visibleSenses.map((sense, idx) => (
                      <li key={sense.synset_id || idx} className="flex gap-4 px-5 py-4">
                        <span className="font-data text-meta text-ink-subtle w-5 shrink-0 pt-1 tabular-nums">
                          {idx + 1}
                        </span>
                        <div className="min-w-0 flex-1 space-y-2.5">
                          <div className="flex flex-wrap items-baseline gap-2">
                            <Badge tone="accent">{sense.part_of_speech || sense.pos_code || "sense"}</Badge>
                            <p className="font-read text-[1.0625rem] leading-snug text-ink">{sense.definition}</p>
                          </div>

                          {sense.examples && sense.examples.length > 0 && (
                            <div className="pl-3.5 border-l-2 border-line space-y-1">
                              {sense.examples.map((ex, i) => (
                                <p key={i} className="font-read text-[0.9375rem] italic text-ink-muted">
                                  {ex}
                                </p>
                              ))}
                            </div>
                          )}

                          {sense.related && sense.related.length > 0 && (
                            <div className="space-y-1.5">
                              {sense.related.map((rel, rIdx) => (
                                <div key={rIdx} className="flex flex-wrap items-center gap-1.5">
                                  <span className="font-nav text-micro text-ink-subtle capitalize mr-0.5">
                                    {rel.relation}
                                  </span>
                                  {(rel.words || []).slice(0, 10).map((rw) => (
                                    <button
                                      key={rw}
                                      type="button"
                                      onClick={() => lookupWord(rw)}
                                      className="h-6 px-2 rounded-chip border border-line bg-sunken font-word text-meta text-ink-muted hover:border-accent-line hover:text-accent-ink hover:bg-accent-soft/50 transition-colors"
                                    >
                                      {rw}
                                    </button>
                                  ))}
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                      </li>
                    ))}
                  </ol>
                  {senses.length > COLLAPSED_SENSE_COUNT && (
                    <button
                      type="button"
                      onClick={() => setShowAllSenses((v) => !v)}
                      aria-expanded={showAllSenses}
                      className="w-full flex items-center justify-center gap-1.5 h-10 border-t border-line font-nav text-meta text-ink-muted hover:text-ink hover:bg-sunken/60 transition-colors rounded-b-card"
                    >
                      <ChevronDown
                        className={cx("w-4 h-4 transition-transform duration-200", showAllSenses && "rotate-180")}
                        aria-hidden
                      />
                      {showAllSenses ? "Show fewer definitions" : `Show all ${senses.length} definitions`}
                    </button>
                  )}
                </Card>
              ) : (
                <EmptyState icon={BookText} title="No structured senses found for this entry." />
              )}

              {entry.related_lists && entry.related_lists.length > 0 && (
                <div className="pt-3 space-y-3">
                  <SectionTitle icon={ListChecks} as="h3">
                    Lists featuring "{entry.word}"
                  </SectionTitle>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    {entry.related_lists.map((rl, i) => {
                      const matchId = rl.url.match(/\/lists\/(\d+)/);
                      const numericId = matchId ? Number(matchId[1]) : 0;
                      return (
                        <button
                          key={i}
                          type="button"
                          disabled={!numericId}
                          onClick={() => numericId && onOpenList(numericId)}
                          className="group text-left p-4 rounded-card border border-line bg-surface shadow-card hover:border-accent-line hover:-translate-y-px hover:shadow-raised transition-[border-color,transform,box-shadow] duration-150 space-y-1"
                        >
                          <span className="block font-nav text-ui font-semibold text-ink group-hover:text-accent-ink transition-colors">
                            {rl.title}
                          </span>
                          {rl.description && (
                            <span className="block text-meta text-ink-muted line-clamp-2">{rl.description}</span>
                          )}
                        </button>
                      );
                    })}
                  </div>
                </div>
              )}
            </div>

            {/* Family & usage */}
            <div className="lg:col-span-5 space-y-6">
              {entry.word_family && entry.word_family.length > 0 && (
                <div className="space-y-3">
                  <SectionTitle icon={GitBranch}>Word Family</SectionTitle>
                  <Card pad="sm" className="space-y-1">
                    {entry.word_family.map((fam) => {
                      const isCurrent = fam.word.toLowerCase() === entry.word.toLowerCase();
                      return (
                        <button
                          key={fam.word}
                          type="button"
                          onClick={() => lookupWord(fam.word)}
                          aria-current={isCurrent ? "true" : undefined}
                          className={cx(
                            "w-full grid grid-cols-[1fr_5.5rem] items-center gap-3 px-2.5 py-2 rounded-[0.55rem] text-left transition-colors",
                            isCurrent ? "bg-accent-soft" : "hover:bg-sunken"
                          )}
                        >
                          <span className="min-w-0 flex items-baseline gap-2">
                            <span
                              className={cx(
                                "font-word text-ui truncate",
                                isCurrent ? "font-semibold text-accent-ink" : "text-ink"
                              )}
                            >
                              {fam.word}
                            </span>
                            {fam.parent && (
                              <span className="font-note text-micro text-ink-subtle truncate">from {fam.parent}</span>
                            )}
                          </span>
                          <span className="space-y-1">
                            <span className="block text-right font-data text-micro text-ink-subtle tabular-nums">
                              {fam.freq ? `${fam.freq.toFixed(2)}/4k` : "root"}
                            </span>
                            <Meter
                              decorative
                              label={`${fam.word} usage frequency`}
                              value={((fam.freq || 0) / maxFamilyFreq) * 100}
                              size="xs"
                            />
                          </span>
                        </button>
                      );
                    })}
                  </Card>
                </div>
              )}

              <div className="space-y-3">
                <SectionTitle icon={Quote} action={examplesLoading ? <Spinner label="Loading examples" /> : undefined}>
                  Usage in the Wild
                </SectionTitle>
                <Card pad="none">
                  <div className="flex flex-wrap gap-1.5 p-3 border-b border-line" role="group" aria-label="Example source">
                    {CORPUS_DOMAINS.map((dom) => (
                      <Chip
                        key={dom.code}
                        size="sm"
                        active={selectedDomain === dom.code}
                        onClick={() => handleDomainChange(dom.code)}
                      >
                        {dom.label}
                      </Chip>
                    ))}
                  </div>

                  {examples.length > 0 ? (
                    <ul className={cx("divide-y divide-line transition-opacity", examplesLoading && "opacity-50")}>
                      {examples.map((ex, i) => (
                        <li key={i} className="px-4 py-3.5 space-y-1.5">
                          <p className="font-read text-[0.9875rem] leading-relaxed text-ink">
                            {renderHighlightedSentence(ex.sentence, ex.offsets)}
                          </p>
                          <div className="flex items-center justify-between gap-2">
                            <span className="font-note text-micro text-ink-subtle truncate">
                              {ex.source_title || ex.source_name || "Corpus"}
                              {ex.author ? ` · ${ex.author}` : ""}
                            </span>
                            {ex.source_url && (
                              <a
                                href={ex.source_url}
                                target="_blank"
                                rel="noreferrer"
                                className="inline-flex items-center gap-1 font-nav text-micro text-accent-ink hover:underline shrink-0 rounded"
                              >
                                <span>Source</span>
                                <ExternalLink className="w-3 h-3" aria-hidden />
                              </a>
                            )}
                          </div>
                        </li>
                      ))}
                    </ul>
                  ) : (
                    <p className="px-4 py-6 text-meta text-ink-muted">No usage examples found for this filter.</p>
                  )}
                </Card>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

function renderHighlightedSentence(sentence: string, offsets?: number[] | null) {
  if (!offsets || offsets.length < 2) {
    return sentence;
  }
  const [start, end] = offsets;
  if (start < 0 || end <= start || end > sentence.length) {
    return sentence;
  }
  return (
    <>
      {sentence.slice(0, start)}
      <mark className="bg-transparent word-mark">{sentence.slice(start, end)}</mark>
      {sentence.slice(end)}
    </>
  );
}
