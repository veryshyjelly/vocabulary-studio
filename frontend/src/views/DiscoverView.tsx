import React, { useState, useEffect, useCallback } from "react";
import {
  Compass,
  Calendar,
  BookMarked,
  Filter,
  ArrowLeft,
  Play,
  BookOpen,
  Loader2,
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
      <div className="flex flex-wrap items-center justify-between gap-4 pb-4 border-b border-stone-200/80 dark:border-stone-800/80">
        <div>
          <h1 className="text-2xl font-editorial font-semibold text-stone-900 dark:text-stone-100">
            Discover Vocabulary
          </h1>
          <p className="text-xs text-stone-500 dark:text-stone-400">
            Explore curated literature & test-prep collections, the Word of the Day archive, and pattern-based WordFinder.
          </p>
        </div>

        <div className="inline-flex p-1 rounded-lg bg-stone-200/70 dark:bg-stone-900 border border-stone-200 dark:border-stone-800">
          {[
            { id: "collections" as const, label: "Curated Collections", icon: BookMarked },
            { id: "wod" as const, label: "Word of the Day", icon: Calendar },
            { id: "wordfinder" as const, label: "WordFinder", icon: Filter },
          ].map((item) => {
            const Icon = item.icon;
            const active = tab === item.id;
            return (
              <button
                key={item.id}
                type="button"
                onClick={() => {
                  setTab(item.id);
                  setError(null);
                  if (item.id === "wod" && !wod) loadWOD("");
                  if (item.id === "wordfinder" && !wfResult) handleWordFinder();
                }}
                className={`inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-md text-xs font-medium transition-colors ${
                  active
                    ? "bg-white dark:bg-stone-800 text-stone-900 dark:text-stone-100 shadow-2xs"
                    : "text-stone-600 dark:text-stone-400 hover:text-stone-900 dark:hover:text-stone-200"
                }`}
              >
                <Icon className="w-3.5 h-3.5 text-emerald-700 dark:text-emerald-400" />
                <span>{item.label}</span>
              </button>
            );
          })}
        </div>
      </div>

      {error && <ErrorBanner message={error} onDismiss={() => setError(null)} />}

      {/* ==================== TAB 1: CURATED COLLECTIONS / UNITS ==================== */}
      {tab === "collections" && (
        <div className="space-y-6">
          {selectedUnit ? (
            <div className="space-y-6">
              <button
                type="button"
                onClick={() => setSelectedUnit(null)}
                className="inline-flex items-center gap-1.5 text-xs font-medium text-stone-600 dark:text-stone-400 hover:text-stone-900 dark:hover:text-stone-100"
              >
                <ArrowLeft className="w-4 h-4" />
                <span>Back to Collections</span>
              </button>

              <div className="rounded-xl border border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 p-6 space-y-3">
                <span className="text-xs font-mono uppercase text-emerald-700 dark:text-emerald-400">
                  {selectedUnit.type} {selectedUnit.author ? `· ${selectedUnit.author}` : ""}
                </span>
                <h2 className="text-2xl font-editorial font-bold text-stone-900 dark:text-stone-100">
                  {selectedUnit.title}
                </h2>
                {(selectedUnit.description || selectedUnit.short_description) && (
                  <p className="text-sm text-stone-600 dark:text-stone-300 leading-relaxed">
                    {selectedUnit.description || selectedUnit.short_description}
                  </p>
                )}
              </div>

              <div className="space-y-3">
                <h3 className="text-xs font-semibold uppercase tracking-wider text-stone-500">
                  Sub-Lists in this Collection ({selectedUnit.items?.length || 0})
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                  {(selectedUnit.items || []).map((it, idx) => {
                    const numId = typeof it.id === "number" ? it.id : Number(it.id);
                    return (
                      <div
                        key={idx}
                        className="p-4 rounded-xl border border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 flex items-center justify-between gap-3"
                      >
                        <div>
                          <p className="text-sm font-semibold text-stone-900 dark:text-stone-100">
                            {it.name || it.title}
                          </p>
                          {it.word_count ? (
                            <p className="text-xs text-stone-500">{it.word_count} words</p>
                          ) : null}
                        </div>
                        {numId > 0 && (
                          <div className="flex items-center gap-2 shrink-0">
                            <button
                              type="button"
                              onClick={() => onOpenList(numId)}
                              className="px-2.5 py-1 text-xs rounded border border-stone-200 dark:border-stone-800 hover:bg-stone-100 dark:hover:bg-stone-800"
                            >
                              Inspect
                            </button>
                            <button
                              type="button"
                              onClick={() => onStartPractice(numId)}
                              className="px-2.5 py-1 text-xs rounded bg-emerald-700 text-white"
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
                      onClick={() => {
                        setSelectedCategory(cat.unit_id);
                        setUnitQuery("");
                        loadUnits("", cat.unit_id);
                      }}
                      className={`p-3.5 rounded-xl border text-left transition-colors ${
                        active
                          ? "border-emerald-600 bg-emerald-50/70 dark:bg-emerald-950/40"
                          : "border-stone-200/80 dark:border-stone-800 bg-white dark:bg-stone-900/70 hover:border-stone-300"
                      }`}
                    >
                      <div className="flex items-center justify-between">
                        <span className="text-xs font-semibold text-stone-900 dark:text-stone-100">
                          {cat.title}
                        </span>
                        <span className="text-[10px] font-mono text-emerald-700 dark:text-emerald-400">
                          {cat.tag}
                        </span>
                      </div>
                      <p className="text-[11px] text-stone-500 dark:text-stone-400 mt-1 line-clamp-1">
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
                  <Search className="w-4 h-4 text-stone-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
                  <input
                    type="text"
                    value={unitQuery}
                    onChange={(e) => setUnitQuery(e.target.value)}
                    placeholder="Search books, authors, or subjects (e.g. Brave New World, Jane Austen)..."
                    className="w-full pl-10 pr-4 py-2 text-sm rounded-xl border border-stone-300 dark:border-stone-700 bg-white dark:bg-stone-900"
                  />
                </div>
                <button
                  type="submit"
                  disabled={loading}
                  className="px-4 py-2 rounded-xl bg-emerald-700 hover:bg-emerald-800 text-white text-xs font-medium"
                >
                  Search
                </button>
              </form>

              {loading ? (
                <div className="p-12 flex justify-center">
                  <Loader2 className="w-6 h-6 animate-spin text-emerald-700" />
                </div>
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {(unitsResp?.units || []).map((u) => (
                    <div
                      key={u.id}
                      className="p-5 rounded-xl border border-stone-200/90 dark:border-stone-800 bg-white dark:bg-stone-900/80 flex flex-col justify-between gap-3"
                    >
                      <div className="space-y-1">
                        <span className="text-[11px] font-mono uppercase text-emerald-700 dark:text-emerald-400">
                          {u.type} {u.author ? `· ${u.author}` : ""}
                        </span>
                        <h3 className="text-lg font-editorial font-semibold text-stone-900 dark:text-stone-100">
                          {u.title}
                        </h3>
                        {(u.short_description || u.description) && (
                          <p className="text-xs text-stone-500 dark:text-stone-400 line-clamp-2">
                            {u.short_description || u.description}
                          </p>
                        )}
                      </div>
                      <div className="flex items-center justify-between pt-2 border-t border-stone-100 dark:border-stone-800">
                        <span className="text-xs text-stone-400">
                          {u.items?.length || 0} lists
                        </span>
                        <button
                          type="button"
                          onClick={() => openUnitDetail(u.id)}
                          className="px-3 py-1.5 rounded-md bg-emerald-50 dark:bg-emerald-950/60 text-emerald-800 dark:text-emerald-300 text-xs font-medium hover:bg-emerald-100"
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
          <div className="flex items-center gap-3">
            <input
              type="date"
              value={wodDate}
              onChange={(e) => {
                setWodDate(e.target.value);
                loadWOD(e.target.value);
              }}
              className="px-3 py-1.5 text-xs rounded-lg border border-stone-300 dark:border-stone-700 bg-white dark:bg-stone-900"
            />
            <button
              type="button"
              onClick={() => {
                setWodDate("");
                loadWOD("");
              }}
              className="px-3 py-1.5 text-xs font-medium rounded-lg border border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900"
            >
              Today's Word
            </button>
          </div>

          {wod && (
            <div className="rounded-xl border border-stone-200/90 dark:border-stone-800 bg-white dark:bg-stone-900 p-8 space-y-4">
              <div className="flex items-center justify-between">
                <span className="text-xs font-medium uppercase tracking-wider text-emerald-700 dark:text-emerald-400">
                  {wod.date || "Word of the Day"}
                </span>
                <div className="flex items-center gap-2">
                  <AudioButton audioUrl={wod.audio_url} word={wod.word} label="Pronounce" />
                  <button
                    type="button"
                    onClick={() => onOpenWord(wod.word)}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-emerald-700 text-white text-xs font-medium"
                  >
                    <BookOpen className="w-3.5 h-3.5" />
                    <span>Open in Dictionary</span>
                  </button>
                </div>
              </div>
              <h2 className="text-4xl font-editorial font-bold text-stone-900 dark:text-stone-100">
                {wod.word}
              </h2>
              {wod.short_usage && (
                <p className="text-base font-editorial text-stone-800 dark:text-stone-200 leading-relaxed">
                  {wod.short_usage}
                </p>
              )}
              {wod.long_description && (
                <p className="text-sm text-stone-600 dark:text-stone-400 leading-relaxed">
                  {wod.long_description}
                </p>
              )}
            </div>
          )}

          {archive?.results && archive.results.length > 0 && (
            <div className="space-y-4">
              <h3 className="text-xs font-semibold uppercase tracking-wider text-stone-500">
                Recent Word of the Day Archive
              </h3>
              {archive.results.map((month) => (
                <div
                  key={month.yearMonth}
                  className="rounded-xl border border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 p-5 space-y-3"
                >
                  <h4 className="text-sm font-semibold text-stone-900 dark:text-stone-100">
                    {month.firstOfMonth.slice(0, 7)}
                  </h4>
                  <div className="flex flex-wrap gap-2">
                    {(month.wordOfTheDays || []).map((item) => (
                      <button
                        key={item.date}
                        type="button"
                        onClick={() => onOpenWord(item.word)}
                        className="px-3 py-1.5 rounded-lg border border-stone-200 dark:border-stone-800 hover:border-emerald-500 text-xs flex items-center gap-2"
                      >
                        <span className="font-mono text-[10px] text-stone-400">
                          {item.date.slice(5)}
                        </span>
                        <span className="font-editorial font-semibold text-stone-800 dark:text-stone-200">
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
            className="rounded-xl border border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 p-6 grid grid-cols-1 sm:grid-cols-4 gap-3 items-end"
          >
            <div>
              <label className="block text-xs font-medium text-stone-500 mb-1">Letters</label>
              <input
                type="text"
                value={wfText}
                onChange={(e) => setWfText(e.target.value)}
                placeholder="e.g. cal"
                className="w-full px-3 py-2 text-sm rounded-lg border border-stone-300 dark:border-stone-700 bg-stone-50 dark:bg-stone-950"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-stone-500 mb-1">Match Rule</label>
              <select
                value={wfLocation}
                onChange={(e) => setWfLocation(e.target.value)}
                className="w-full px-3 py-2 text-sm rounded-lg border border-stone-300 dark:border-stone-700 bg-stone-50 dark:bg-stone-950"
              >
                <option value="startingwith">Starting with</option>
                <option value="endingwith">Ending with</option>
                <option value="containinginorder">Containing in order</option>
                <option value="including">Including letters</option>
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-stone-500 mb-1">
                Word Length (optional)
              </label>
              <input
                type="number"
                min={0}
                max={25}
                value={wfLetters}
                onChange={(e) => setWfLetters(e.target.value)}
                placeholder="Any"
                className="w-full px-3 py-2 text-sm rounded-lg border border-stone-300 dark:border-stone-700 bg-stone-50 dark:bg-stone-950"
              />
            </div>
            <button
              type="submit"
              disabled={loading}
              className="px-4 py-2 rounded-lg bg-emerald-700 hover:bg-emerald-800 text-white text-xs font-medium h-9.5"
            >
              Find Words
            </button>
          </form>

          {wfResult?.words && (
            <div className="space-y-3">
              <p className="text-xs text-stone-500">
                Found {wfResult.totalResults} matching words:
              </p>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                {wfResult.words.map((w) => (
                  <button
                    key={w.word}
                    type="button"
                    onClick={() => onOpenWord(w.word)}
                    className="p-4 rounded-xl border border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 text-left hover:border-emerald-500 transition-colors"
                  >
                    <span className="text-lg font-editorial font-semibold text-stone-900 dark:text-stone-100">
                      {w.word}
                    </span>
                    {w.senses?.[0] && (
                      <p className="text-xs text-stone-500 dark:text-stone-400 mt-1 line-clamp-2">
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
