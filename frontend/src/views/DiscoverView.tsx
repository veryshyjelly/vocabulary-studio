import React, { useState, useEffect, useCallback } from "react";
import {
  Calendar,
  BookMarked,
  Filter,
  ArrowLeft,
  BookOpen,
  Search,
} from "lucide-react";
import type {
  WordOfTheDayEntry,
  WODArchiveResponse,
  CategoryCardDTO,
  UnitsResponse,
  Unit,
  WordFinderResponse,
} from "../lib/bridge";
import { getBridge, formatError } from "../lib/bridge";
import { EmptyState, LoadingBlock, PageHeader, Tabs } from "../components/ui";
import { AudioButton, ErrorBanner } from "../components/Shared";

type DiscoverTab = "collections" | "wod" | "wordfinder";

export const DiscoverView: React.FC<{
  onOpenWord: (word: string) => void;
  onOpenList: (listId: number) => void;
  onStartPractice: (listId: number) => void;
}> = ({ onOpenWord, onOpenList, onStartPractice }) => {
  const [tab, setTab] = useState<DiscoverTab>("collections");

  // Curated Units & Categories state
  const [categories, setCategories] = useState<CategoryCardDTO[]>([]);
  const [selectedCategory, setSelectedCategory] = useState<string>("gfpldfb3"); // Literature default
  const [unitQuery, setUnitQuery] = useState("");
  const [unitsResp, setUnitsResp] = useState<UnitsResponse | null>(null);
  const [selectedUnit, setSelectedUnit] = useState<Unit | null>(null);

  // Word of the Day & Archive state
  const [wodDate, setWodDate] = useState("");
  const [wod, setWod] = useState<WordOfTheDayEntry | null>(null);
  const [archive, setArchive] = useState<WODArchiveResponse | null>(null);

  // WordFinder state
  const [wfText, setWfText] = useState("cal");
  const [wfLocation, setWfLocation] = useState("startingwith");
  const [wfLetters, setWfLetters] = useState<string>("6");
  const [wfResult, setWfResult] = useState<WordFinderResponse | null>(null);

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadUnits = useCallback(async (query: string, catId: string) => {
    setLoading(true);
    setError(null);
    setSelectedUnit(null);
    try {
      const res = await getBridge().discovery.SearchUnits(query, catId, 0, 12);
      setUnitsResp(res);
    } catch (err) {
      setError(formatError(err));
    } finally {
      setLoading(false);
    }
  }, []);

  const loadWOD = useCallback(async (date: string) => {
    setLoading(true);
    setError(null);
    try {
      const entry = await getBridge().discovery.GetWordOfTheDay(date);
      setWod(entry);
      const arch = await getBridge().discovery.GetWordOfTheDayArchive(0, 2);
      setArchive(arch);
    } catch (err) {
      setError(formatError(err));
    } finally {
      setLoading(false);
    }
  }, []);

  const handleWordFinder = useCallback(
    async (e?: React.FormEvent) => {
      if (e) e.preventDefault();
      if (!wfText.trim()) return;
      setLoading(true);
      setError(null);
      try {
        const num = wfLetters.trim() ? Number(wfLetters.trim()) : 0;
        const res = await getBridge().discovery.WordFinder(
          wfText.trim(),
          wfLocation,
          num || 0,
          24
        );
        setWfResult(res);
      } catch (err) {
        setError(formatError(err));
      } finally {
        setLoading(false);
      }
    },
    [wfText, wfLocation, wfLetters]
  );

  useEffect(() => {
    getBridge()
      .discovery.GetRootCategories()
      .then((cats) => setCategories(cats || []))
      .catch(() => {});
    loadUnits("", "gfpldfb3");
  }, [loadUnits]);

  const openUnitDetail = async (unitId: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await getBridge().discovery.GetUnit(unitId);
      if (res?.unit) {
        setSelectedUnit(res.unit);
      }
    } catch (err) {
      setError(formatError(err));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-5xl mx-auto space-y-6 pb-12">
      <PageHeader title="Discover Vocabulary" description="Follow your curiosity through books, collections, and new words.">
        <Tabs<DiscoverTab>
          label="Discovery tools"
          value={tab}
          onChange={(id) => {
            setTab(id);
            setError(null);
            if (id === "wod" && !wod) loadWOD("");
            if (id === "wordfinder" && !wfResult) handleWordFinder();
          }}
          items={[
            { id: "collections", label: "Curated Collections", icon: BookMarked },
            { id: "wod", label: "Word of the Day", icon: Calendar },
            { id: "wordfinder", label: "WordFinder", icon: Filter },
          ]}
        />
      </PageHeader>

      {error && <ErrorBanner message={error} onDismiss={() => setError(null)} />}

      {/* ==================== TAB 1: CURATED COLLECTIONS / UNITS ==================== */}
      {tab === "collections" && (
        <div className="space-y-6">
          {selectedUnit ? (
            <div className="space-y-6">
              <button
                type="button"
                onClick={() => setSelectedUnit(null)}
                className="inline-flex items-center gap-1.5 text-xs font-medium text-ink-muted hover:text-ink"
              >
                <ArrowLeft className="w-4 h-4" />
                <span>Back to Collections</span>
              </button>

              <div className="rounded-xl border border-line bg-surface p-6 space-y-3">
                <span className="text-xs font-data uppercase text-accent-ink">
                  {selectedUnit.type} {selectedUnit.author ? `· ${selectedUnit.author}` : ""}
                </span>
                <h2 className="text-2xl font-word font-bold text-ink">
                  {selectedUnit.title}
                </h2>
                {(selectedUnit.description || selectedUnit.short_description) && (
                  <p className="reading-copy text-ink-muted">
                    {selectedUnit.description || selectedUnit.short_description}
                  </p>
                )}
              </div>

              <div className="space-y-3">
                <h3 className="text-xs font-semibold uppercase tracking-wider text-ink-subtle">
                  Sub-Lists in this Collection ({selectedUnit.items?.length || 0})
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                  {(selectedUnit.items || []).map((it, idx) => {
                    const numId = typeof it.id === "number" ? it.id : Number(it.id);
                    return (
                      <div
                        key={idx}
                        className="p-4 rounded-xl border border-line bg-surface flex items-center justify-between gap-3"
                      >
                        <div>
                          <p className="text-sm font-semibold text-ink">
                            {it.name || it.title}
                          </p>
                          {it.word_count ? (
                            <p className="text-xs text-ink-subtle">{it.word_count} words</p>
                          ) : null}
                        </div>
                        {numId > 0 && (
                          <div className="flex items-center gap-2 shrink-0">
                            <button
                              type="button"
                              onClick={() => onOpenList(numId)}
                              className="px-2.5 py-1 text-xs rounded border border-line hover:bg-sunken"
                            >
                              Inspect
                            </button>
                            <button
                              type="button"
                              onClick={() => onStartPractice(numId)}
                              className="px-2.5 py-1 text-xs rounded bg-accent text-on-accent"
                            >
                              Practice
                            </button>
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>
          ) : (
            <>
              {/* Category Chips */}
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-2.5">
                {categories.map((cat) => {
                  const active = selectedCategory === cat.unit_id && !unitQuery;
                  return (
                    <button
                      key={cat.unit_id}
                      type="button"
                      aria-pressed={selectedCategory === cat.unit_id && !unitQuery}
                      onClick={() => {
                        setSelectedCategory(cat.unit_id);
                        setUnitQuery("");
                        loadUnits("", cat.unit_id);
                      }}
                      className={`p-3.5 rounded-xl border text-left transition-colors ${
                        active
                          ? "border-accent-line bg-accent-soft/70"
                          : "border-line/80 bg-surface hover:border-line-strong"
                      }`}
                    >
                      <div className="flex flex-wrap items-center justify-between gap-3">
                        <span className="text-xs font-semibold text-ink">
                          {cat.title}
                        </span>
                        <span className="text-[10px] font-data text-accent-ink">
                          {cat.tag}
                        </span>
                      </div>
                      <p className="text-[11px] text-ink-subtle mt-1 line-clamp-1">
                        {cat.description}
                      </p>
                    </button>
                  );
                })}
              </div>

              {/* Search Curated Units */}
              <form
                onSubmit={(e) => {
                  e.preventDefault();
                  loadUnits(unitQuery, unitQuery ? "" : selectedCategory);
                }}
                className="flex items-center gap-2"
              >
                <div className="relative flex-1">
                  <Search className="w-4 h-4 text-ink-subtle absolute left-3.5 top-1/2 -translate-y-1/2" />
                  <input
                    aria-label="Search curated collections"
                    type="text"
                    value={unitQuery}
                    onChange={(e) => setUnitQuery(e.target.value)}
                    placeholder="Search books, authors, or subjects (e.g. Brave New World, Jane Austen)..."
                    className="w-full pl-10 pr-4 py-2 text-sm rounded-xl border border-line-strong bg-surface"
                  />
                </div>
                <button
                  type="submit"
                  disabled={loading}
                  className="px-4 py-2 rounded-xl bg-accent hover:bg-accent-hover text-on-accent text-xs font-medium"
                >
                  Search
                </button>
              </form>

              {loading ? (
                <LoadingBlock label="Finding collections…" />
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {!unitsResp?.units?.length && !error && <EmptyState className="col-span-full" icon={Search} title="No collections found" description="Try a different book, author, or subject." />}
                  {(unitsResp?.units || []).map((u) => (
                    <div
                      key={u.id}
                      className="library-item p-5 flex flex-col justify-between gap-3"
                    >
                      <div className="space-y-1">
                        <span className="text-[11px] font-data uppercase text-accent-ink">
                          {u.type} {u.author ? `· ${u.author}` : ""}
                        </span>
                        <h3 className="font-word text-[1.125rem] leading-relaxed font-semibold text-ink break-words">
                          {u.title}
                        </h3>
                        {(u.short_description || u.description) && (
                          <p className="font-read text-read text-ink-muted line-clamp-2">
                            {u.short_description || u.description}
                          </p>
                        )}
                      </div>
                      <div className="flex items-center justify-between pt-2 border-t border-line">
                        <span className="text-xs text-ink-subtle">
                          {u.items?.length || 0} lists
                        </span>
                        <button
                          type="button"
                          onClick={() => openUnitDetail(u.id)}
                          className="px-3 py-1.5 rounded-md bg-accent-soft text-accent-ink text-xs font-medium hover:bg-accent-soft-hover"
                        >
                          Open Collection →
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </>
          )}
        </div>
      )}

      {/* ==================== TAB 2: WORD OF THE DAY & ARCHIVE ==================== */}
      {tab === "wod" && (
        <div className="space-y-6">
          <div className="flex flex-wrap items-center gap-3">
            <input
              aria-label="Word of the Day date"
              type="date"
              value={wodDate}
              onChange={(e) => {
                setWodDate(e.target.value);
                loadWOD(e.target.value);
              }}
              className="px-3 py-1.5 text-xs rounded-lg border border-line-strong bg-surface"
            />
            <button
              type="button"
              onClick={() => {
                setWodDate("");
                loadWOD("");
              }}
              className="px-3 py-1.5 text-xs font-medium rounded-lg border border-line bg-surface"
            >
              Today's Word
            </button>
          </div>

          {loading && <LoadingBlock label="Loading Word of the Day…" />}
          {!loading && wod && (
            <div className="rounded-xl border border-line/90 bg-surface p-8 space-y-4">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <span className="text-xs font-medium uppercase tracking-wider text-accent-ink">
                  {wod.date || "Word of the Day"}
                </span>
                <div className="flex items-center gap-2">
                  <AudioButton audioUrl={wod.audio_url} word={wod.word} label="Pronounce" />
                  <button
                    type="button"
                    onClick={() => onOpenWord(wod.word)}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-accent text-on-accent text-xs font-medium"
                  >
                    <BookOpen className="w-3.5 h-3.5" />
                    <span>Open in Dictionary</span>
                  </button>
                </div>
              </div>
              <h2 className="word-display text-ink">
                {wod.word}
              </h2>
              {wod.short_usage && (
                <p className="reading-copy text-ink text-read-lg">
                  {wod.short_usage}
                </p>
              )}
              {wod.long_description && (
                <p className="reading-copy text-ink-muted">
                  {wod.long_description}
                </p>
              )}
            </div>
          )}

          {archive?.results && archive.results.length > 0 && (
            <div className="space-y-4">
              <h3 className="text-xs font-semibold uppercase tracking-wider text-ink-subtle">
                Recent Word of the Day Archive
              </h3>
              {archive.results.map((month) => (
                <div
                  key={month.yearMonth}
                  className="rounded-xl border border-line bg-surface p-5 space-y-3"
                >
                  <h4 className="text-sm font-semibold text-ink">
                    {month.firstOfMonth.slice(0, 7)}
                  </h4>
                  <div className="flex flex-wrap gap-2">
                    {(month.wordOfTheDays || []).map((item) => (
                      <button
                        key={item.date}
                        type="button"
                        onClick={() => onOpenWord(item.word)}
                        className="px-3 py-1.5 rounded-lg border border-line hover:border-accent-line text-xs flex items-center gap-2"
                      >
                        <span className="font-data text-[10px] text-ink-subtle">
                          {item.date.slice(5)}
                        </span>
                        <span className="font-word font-semibold text-ink">
                          {item.word}
                        </span>
                      </button>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* ==================== TAB 3: ADVANCED WORDFINDER ==================== */}
      {tab === "wordfinder" && (
        <div className="space-y-6">
          <form
            onSubmit={handleWordFinder}
            className="rounded-xl border border-line bg-surface p-6 grid grid-cols-1 sm:grid-cols-4 gap-3 items-end"
          >
            <div>
              <label htmlFor="finder-letters" className="block font-nav text-meta font-medium text-ink-muted mb-1">Letters</label>
              <input
                type="text"
                id="finder-letters" value={wfText}
                onChange={(e) => setWfText(e.target.value)}
                placeholder="e.g. cal"
                className="w-full px-3 py-2 text-sm rounded-lg border border-line-strong bg-sunken"
              />
            </div>
            <div>
              <label htmlFor="finder-rule" className="block font-nav text-meta font-medium text-ink-muted mb-1">Match Rule</label>
              <select
                id="finder-rule" value={wfLocation}
                onChange={(e) => setWfLocation(e.target.value)}
                className="w-full px-3 py-2 text-sm rounded-lg border border-line-strong bg-sunken"
              >
                <option value="startingwith">Starting with</option>
                <option value="endingwith">Ending with</option>
                <option value="containinginorder">Containing in order</option>
                <option value="including">Including letters</option>
              </select>
            </div>
            <div>
              <label htmlFor="finder-length" className="block font-nav text-meta font-medium text-ink-muted mb-1">
                Word Length (optional)
              </label>
              <input
                type="number"
                min={0}
                max={25}
                id="finder-length" value={wfLetters}
                onChange={(e) => setWfLetters(e.target.value)}
                placeholder="Any"
                className="w-full px-3 py-2 text-sm rounded-lg border border-line-strong bg-sunken"
              />
            </div>
            <button
              type="submit"
              disabled={loading}
              className="px-4 py-2 rounded-lg bg-accent hover:bg-accent-hover text-on-accent text-xs font-medium h-9.5"
            >
              Find Words
            </button>
          </form>

          {wfResult?.words && (
            <div className="space-y-3">
              <p className="text-xs text-ink-subtle">
                Found {wfResult.totalResults} matching words:
              </p>
              {wfResult.words.length === 0 && <EmptyState icon={Filter} title="No words match these letters" description="Try a different match rule or leave the word length blank." />}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                {wfResult.words.map((w) => (
                  <button
                    key={w.word}
                    type="button"
                    onClick={() => onOpenWord(w.word)}
                    className="p-4 rounded-xl border border-line bg-surface text-left hover:border-accent-line transition-colors"
                  >
                    <span className="font-word text-[1.125rem] leading-relaxed font-semibold text-ink break-words">
                      {w.word}
                    </span>
                    {w.senses?.[0] && (
                      <p className="font-read text-read text-ink-muted mt-2 line-clamp-2">
                        [{w.senses[0].pos}] {w.senses[0].definition}
                      </p>
                    )}
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
};
