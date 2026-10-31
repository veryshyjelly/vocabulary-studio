import React, { useState, useEffect, useCallback } from "react";
import {
  ListChecks,
  Search,
  Plus,
  Sparkles,
  Play,
  SpellCheck,
  Layers,
  Trash2,
  Star,
  ArrowLeft,
  ArrowRight,
  RotateCw,
  Loader2,
  BookOpen,
} from "lucide-react";
import type {
  AccountStatusDTO,
  ProfileListSummary,
  SmartSearchResponse,
  VocabularyList,
  FlashcardListResponse,
  ListProgressStatus,
  GrabbedWord,
} from "../lib/bridge";
import { getBridge, formatError } from "../lib/bridge";
import { AudioButton, ErrorBanner, Kbd } from "../components/Shared";

type ListsTab = "my-lists" | "search" | "create" | "instalist";

export const ListsView: React.FC<{
  account: AccountStatusDTO | null;
  initialListId?: number;
  onStartPractice: (listId: number) => void;
  onStartBee: (listId: number) => void;
  onOpenWord: (word: string) => void;
  onRequestLogin: () => void;
}> = ({
  account,
  initialListId = 0,
  onStartPractice,
  onStartBee,
  onOpenWord,
  onRequestLogin,
}) => {
  const [activeTab, setActiveTab] = useState<ListsTab>(
    account?.authenticated ? "my-lists" : "search"
  );
  const [userLists, setUserLists] = useState<ProfileListSummary[]>([]);
  const [searchQuery, setSearchQuery] = useState("GRE high frequency");
  const [searchResults, setSearchResults] = useState<SmartSearchResponse | null>(null);

  // Selected List Detail & Flashcard state
  const [selectedList, setSelectedList] = useState<VocabularyList | null>(null);
  const [listProgress, setListProgress] = useState<ListProgressStatus | null>(null);
  const [flashcards, setFlashcards] = useState<FlashcardListResponse | null>(null);
  const [showFlashcards, setShowFlashcards] = useState(false);
  const [cardIndex, setCardIndex] = useState(0);
  const [cardFlipped, setCardFlipped] = useState(false);

  // Create List & Add Words state
  const [newListName, setNewListName] = useState("");
  const [newListDesc, setNewListDesc] = useState("");
  const [newListWordsRaw, setNewListWordsRaw] = useState("");
  const [addWordsInput, setAddWordsInput] = useState("");
  const [confirmDelete, setConfirmDelete] = useState(false);

  // InstaList (VocabGrabber) state
  const [grabText, setGrabText] = useState("");
  const [grabbedWords, setGrabbedWords] = useState<GrabbedWord[]>([]);

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const loadUserLists = useCallback(async () => {
    if (!account?.authenticated) return;
    setLoading(true);
    setError(null);
    try {
      const lists = await getBridge().lists.GetUserLists("");
      setUserLists(lists || []);
    } catch (err) {
      setError(formatError(err));
    } finally {
      setLoading(false);
    }
  }, [account?.authenticated]);

  const handleSearchLists = useCallback(
    async (e?: React.FormEvent) => {
      if (e) e.preventDefault();
      if (!searchQuery.trim()) return;
      setLoading(true);
      setError(null);
      try {
        const res = await getBridge().lists.SearchLists(searchQuery.trim(), 0, 20);
        setSearchResults(res);
      } catch (err) {
        setError(formatError(err));
      } finally {
        setLoading(false);
      }
    },
    [searchQuery]
  );

  const openListDetail = useCallback(
    async (listId: number) => {
      if (!listId || listId <= 0) return;
      setLoading(true);
      setError(null);
      setShowFlashcards(false);
      setConfirmDelete(false);
      try {
        const wl = await getBridge().lists.GetListDetail(listId);
        setSelectedList(wl);
        if (account?.authenticated) {
          getBridge()
            .progress.GetListProgress(listId)
            .then((lp) => setListProgress(lp))
            .catch(() => setListProgress(null));
        }
      } catch (err) {
        setError(formatError(err));
      } finally {
        setLoading(false);
      }
    },
    [account?.authenticated]
  );

  useEffect(() => {
    if (initialListId > 0) {
      openListDetail(initialListId);
    } else if (account?.authenticated) {
      loadUserLists();
    } else {
      handleSearchLists();
    }
  }, [initialListId, account?.authenticated]);

  const handleOpenFlashcards = async () => {
    if (!selectedList) return;
    setLoading(true);
    setError(null);
    try {
      const fc = await getBridge().lists.GetFlashcards(selectedList.wordlistid);
      setFlashcards(fc);
      setCardIndex(0);
      setCardFlipped(false);
      setShowFlashcards(true);
    } catch (err) {
      setError(formatError(err));
    } finally {
      setLoading(false);
    }
  };

  // Keyboard navigation for flashcards
  useEffect(() => {
    if (!showFlashcards) return;
    const words = flashcards?.wordlist?.words || [];
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === " " || e.key === "Enter") {
        e.preventDefault();
        setCardFlipped((f) => !f);
      } else if (e.key === "ArrowRight") {
        e.preventDefault();
        setCardFlipped(false);
        setCardIndex((i) => (i + 1 < words.length ? i + 1 : 0));
      } else if (e.key === "ArrowLeft") {
        e.preventDefault();
        setCardFlipped(false);
        setCardIndex((i) => (i > 0 ? i - 1 : Math.max(0, words.length - 1)));
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [showFlashcards, flashcards]);

  const handleCreateList = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!account?.authenticated) {
      onRequestLogin();
      return;
    }
    const words = newListWordsRaw
      .split(/[\n,]+/)
      .map((w) => w.trim())
      .filter(Boolean)
      .map((word) => ({ word, lang: "en" }));

    if (!newListName.trim() || words.length === 0) {
      setError("Please provide a list name and at least one word.");
      return;
    }

    setLoading(true);
    setError(null);
    try {
      const newId = await getBridge().lists.CreateList(
        newListName.trim(),
        newListDesc.trim(),
        true,
        words
      );
      setNewListName("");
      setNewListDesc("");
      setNewListWordsRaw("");
      setNotice(`Created list #${newId} successfully.`);
      if (newId > 0) {
        await openListDetail(newId);
      }
    } catch (err) {
      setError(formatError(err));
    } finally {
      setLoading(false);
    }
  };

  const handleGrabVocabulary = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!grabText.trim()) return;
    setLoading(true);
    setError(null);
    try {
      const res = await getBridge().lists.GrabVocabulary(grabText);
      setGrabbedWords(res?.words || []);
    } catch (err) {
      setError(formatError(err));
    } finally {
      setLoading(false);
    }
  };

  const handleAddWordsToCurrentList = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedList || !addWordsInput.trim()) return;
    const words = addWordsInput
      .split(/[\n,]+/)
      .map((w) => w.trim())
      .filter(Boolean)
      .map((word) => ({ word, lang: "en" }));

    setLoading(true);
    setError(null);
    try {
      await getBridge().lists.AddWordsToList(selectedList.wordlistid, words);
      setAddWordsInput("");
      setNotice(`Added ${words.length} word(s) to "${selectedList.name}".`);
      await openListDetail(selectedList.wordlistid);
    } catch (err) {
      setError(formatError(err));
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteCurrentList = async () => {
    if (!selectedList) return;
    setLoading(true);
    setError(null);
    try {
      await getBridge().lists.DeleteList(selectedList.wordlistid);
      setNotice(`Deleted list "${selectedList.name}".`);
      setSelectedList(null);
      setConfirmDelete(false);
      loadUserLists();
    } catch (err) {
      setError(formatError(err));
    } finally {
      setLoading(false);
    }
  };

  const handleToggleFavorite = async (fav: boolean) => {
    if (!selectedList) return;
    try {
      await getBridge().lists.SetFavorite(selectedList.wordlistid, fav);
      setNotice(fav ? "Bookmarked list." : "Removed bookmark.");
    } catch (err) {
      setError(formatError(err));
    }
  };

  // ==================== SELECTED LIST DETAIL VIEW ====================
  if (selectedList) {
    const fcWords = flashcards?.wordlist?.words || [];
    const currentCard = fcWords[cardIndex];

    return (
      <div className="max-w-5xl mx-auto space-y-6 pb-12">
        <div className="flex items-center justify-between">
          <button
            type="button"
            onClick={() => {
              setSelectedList(null);
              setShowFlashcards(false);
            }}
            className="inline-flex items-center gap-1.5 text-xs font-medium text-stone-600 dark:text-stone-400 hover:text-stone-900 dark:hover:text-stone-100"
          >
            <ArrowLeft className="w-4 h-4" />
            <span>Back to Vocabulary Lists</span>
          </button>

          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => handleToggleFavorite(true)}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 text-xs font-medium text-stone-700 dark:text-stone-200 hover:bg-stone-100 dark:hover:bg-stone-800"
            >
              <Star className="w-3.5 h-3.5 text-amber-500" />
              <span>Bookmark</span>
            </button>
            {account?.authenticated && (
              <button
                type="button"
                onClick={() => setConfirmDelete(true)}
                className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-rose-200 dark:border-rose-900/60 text-rose-700 dark:text-rose-300 text-xs font-medium hover:bg-rose-50 dark:hover:bg-rose-950/40"
              >
                <Trash2 className="w-3.5 h-3.5" />
                <span>Delete</span>
              </button>
            )}
          </div>
        </div>

        {error && <ErrorBanner message={error} onDismiss={() => setError(null)} />}
        {notice && (
          <div className="p-3 rounded-lg bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-200 dark:border-emerald-900 text-xs text-emerald-900 dark:text-emerald-200">
            {notice}
          </div>
        )}

        {/* Destructive Delete Confirmation Dialog */}
        {confirmDelete && (
          <div className="p-4 rounded-xl border border-rose-300 dark:border-rose-800 bg-rose-50/80 dark:bg-rose-950/40 flex items-center justify-between gap-4">
            <p className="text-xs text-rose-900 dark:text-rose-200 font-medium">
              Are you sure you want to permanently delete "{selectedList.name}"?
            </p>
            <div className="flex items-center gap-2 shrink-0">
              <button
                type="button"
                onClick={() => setConfirmDelete(false)}
                className="px-3 py-1.5 rounded text-xs bg-white dark:bg-stone-900 border border-stone-200 dark:border-stone-700"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleDeleteCurrentList}
                className="px-3 py-1.5 rounded text-xs bg-rose-600 text-white font-medium"
              >
                Confirm Delete
              </button>
            </div>
          </div>
        )}

        {/* List Header Card */}
        <div className="rounded-xl border border-stone-200/90 dark:border-stone-800 bg-white dark:bg-stone-900/90 p-6 md:p-8 space-y-5">
          <div className="flex flex-col md:flex-row md:items-start justify-between gap-4">
            <div className="space-y-2 max-w-2xl">
              <span className="text-xs font-mono text-stone-400">
                List #{selectedList.wordlistid} · {selectedList.words?.length || 0} words
                {listProgress
                  ? ` · Mastery ${Math.round((listProgress.progress || 0) * 100)}%`
                  : ""}
              </span>
              <h1 className="text-2xl md:text-3xl font-editorial font-bold text-stone-900 dark:text-stone-100">
                {selectedList.name}
              </h1>
              {selectedList.description && (
                <p className="text-sm text-stone-600 dark:text-stone-300 leading-relaxed">
                  {selectedList.description}
                </p>
              )}
            </div>

            <div className="flex flex-wrap items-center gap-2 shrink-0">
              <button
                type="button"
                onClick={() => onStartPractice(selectedList.wordlistid)}
                className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-700 hover:bg-emerald-800 text-white text-xs font-medium shadow-xs"
              >
                <Play className="w-3.5 h-3.5 fill-current" />
                <span>Practice List</span>
              </button>
              <button
                type="button"
                onClick={() => onStartBee(selectedList.wordlistid)}
                className="inline-flex items-center gap-2 px-3.5 py-2 rounded-lg border border-stone-200 dark:border-stone-800 hover:bg-stone-100 dark:hover:bg-stone-800 text-xs font-medium text-stone-700 dark:text-stone-200"
              >
                <SpellCheck className="w-3.5 h-3.5 text-emerald-700 dark:text-emerald-400" />
                <span>Spelling Bee</span>
              </button>
              <button
                type="button"
                onClick={() =>
                  showFlashcards ? setShowFlashcards(false) : handleOpenFlashcards()
                }
                className="inline-flex items-center gap-2 px-3.5 py-2 rounded-lg border border-stone-200 dark:border-stone-800 hover:bg-stone-100 dark:hover:bg-stone-800 text-xs font-medium text-stone-700 dark:text-stone-200"
              >
                <Layers className="w-3.5 h-3.5 text-emerald-700 dark:text-emerald-400" />
                <span>{showFlashcards ? "Hide Flashcards" : "Flashcards"}</span>
              </button>
            </div>
          </div>

          {/* Add Words Form */}
          {account?.authenticated && (
            <form onSubmit={handleAddWordsToCurrentList} className="flex items-center gap-2 pt-2">
              <input
                type="text"
                value={addWordsInput}
                onChange={(e) => setAddWordsInput(e.target.value)}
                placeholder="Add words to this list (comma-separated)..."
                className="flex-1 px-3 py-1.5 text-xs rounded-lg border border-stone-200 dark:border-stone-800 bg-stone-50 dark:bg-stone-950"
              />
              <button
                type="submit"
                disabled={!addWordsInput.trim() || loading}
                className="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg bg-stone-900 dark:bg-stone-100 text-white dark:text-stone-900 text-xs font-medium disabled:opacity-40"
              >
                <Plus className="w-3.5 h-3.5" />
                <span>Add Words</span>
              </button>
            </form>
          )}
        </div>

        {/* Interactive Flashcards Deck */}
        {showFlashcards && currentCard && (
          <div className="rounded-xl border border-emerald-200 dark:border-emerald-900/60 bg-white dark:bg-stone-900 p-8 shadow-sm space-y-6">
            <div className="flex items-center justify-between text-xs text-stone-500">
              <span>
                Flashcard {cardIndex + 1} of {fcWords.length}
              </span>
              <span className="flex items-center gap-1.5">
                <Kbd>Space</Kbd> Flip · <Kbd>←</Kbd> <Kbd>→</Kbd> Navigate
              </span>
            </div>

            <div
              onClick={() => setCardFlipped((f) => !f)}
              role="button"
              tabIndex={0}
              onKeyDown={(e) => {
                if (e.key === " " || e.key === "Enter") setCardFlipped((f) => !f);
              }}
              className="min-h-56 flex flex-col items-center justify-center text-center p-8 rounded-xl border border-stone-200 dark:border-stone-800 bg-stone-50/70 dark:bg-stone-950/60 cursor-pointer select-none space-y-4"
            >
              {!cardFlipped ? (
                <>
                  <h2 className="text-4xl font-editorial font-bold text-stone-900 dark:text-stone-100">
                    {currentCard.word}
                  </h2>
                  {currentCard.sense?.ipas?.[0]?.formatted_ipastring && (
                    <p className="text-sm font-mono text-stone-500">
                      {currentCard.sense.ipas[0].formatted_ipastring}
                    </p>
                  )}
                  <AudioButton word={currentCard.word} size="sm" label="Listen" />
                  <p className="text-xs text-stone-400 pt-2">Click or press Space to reveal definition</p>
                </>
              ) : (
                <>
                  {currentCard.sense?.part_of_speech && (
                    <span className="px-2.5 py-0.5 rounded text-xs font-mono bg-emerald-50 dark:bg-emerald-950 text-emerald-700 dark:text-emerald-300">
                      {currentCard.sense.part_of_speech}
                    </span>
                  )}
                  <p className="text-xl font-medium text-stone-900 dark:text-stone-100 max-w-xl">
                    {currentCard.sense?.definition || "Definition unavailable"}
                  </p>
                  {currentCard.example?.text && (
                    <p className="text-sm font-editorial italic text-stone-600 dark:text-stone-400 max-w-xl">
                      "{currentCard.example.text}"
                    </p>
                  )}
                </>
              )}
            </div>

            <div className="flex items-center justify-between">
              <button
                type="button"
                onClick={() => {
                  setCardFlipped(false);
                  setCardIndex((i) => (i > 0 ? i - 1 : fcWords.length - 1));
                }}
                className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg border border-stone-200 dark:border-stone-800 text-xs font-medium"
              >
                <ArrowLeft className="w-3.5 h-3.5" />
                <span>Previous</span>
              </button>
              <button
                type="button"
                onClick={() => setCardFlipped((f) => !f)}
                className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg border border-stone-200 dark:border-stone-800 text-xs font-medium"
              >
                <RotateCw className="w-3.5 h-3.5" />
                <span>Flip Card</span>
              </button>
              <button
                type="button"
                onClick={() => {
                  setCardFlipped(false);
                  setCardIndex((i) => (i + 1 < fcWords.length ? i + 1 : 0));
                }}
                className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg border border-stone-200 dark:border-stone-800 text-xs font-medium"
              >
                <span>Next</span>
                <ArrowRight className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        )}

        {/* Words in List */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {(selectedList.words || []).map((w, idx) => (
            <div
              key={`${w.word}-${idx}`}
              className="p-4 rounded-xl border border-stone-200/80 dark:border-stone-800 bg-white dark:bg-stone-900/80 flex flex-col justify-between gap-2"
            >
              <div className="space-y-1">
                <div className="flex items-center justify-between gap-2">
                  <button
                    type="button"
                    onClick={() => onOpenWord(w.word)}
                    className="text-lg font-editorial font-semibold text-stone-900 dark:text-stone-100 hover:text-emerald-700 dark:hover:text-emerald-400"
                  >
                    {w.word}
                  </button>
                  <AudioButton word={w.word} size="sm" />
                </div>
                <p className="text-xs md:text-sm text-stone-700 dark:text-stone-300">
                  {w.definition || w.shortdefinition}
                </p>
                {w.example?.text && (
                  <p className="text-xs font-editorial italic text-stone-500 dark:text-stone-400 pt-1">
                    "{w.example.text}"
                  </p>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  // ==================== LISTS OVERVIEW & TABS ====================
  return (
    <div className="max-w-5xl mx-auto space-y-6 pb-12">
      <div className="flex flex-wrap items-center justify-between gap-4 pb-4 border-b border-stone-200/80 dark:border-stone-800/80">
        <div>
          <h1 className="text-2xl font-editorial font-semibold text-stone-900 dark:text-stone-100">
            Vocabulary Lists
          </h1>
          <p className="text-xs text-stone-500 dark:text-stone-400">
            Study curated collections, manage your personal lists, or extract words from any text.
          </p>
        </div>

        <div className="inline-flex p-1 rounded-lg bg-stone-200/70 dark:bg-stone-900 border border-stone-200 dark:border-stone-800">
          {[
            { id: "my-lists" as const, label: "My Lists", icon: ListChecks },
            { id: "search" as const, label: "Search Lists", icon: Search },
            { id: "create" as const, label: "Create List", icon: Plus },
            { id: "instalist" as const, label: "InstaList Extractor", icon: Sparkles },
          ].map((tab) => {
            const Icon = tab.icon;
            const active = activeTab === tab.id;
            return (
              <button
                key={tab.id}
                type="button"
                onClick={() => {
                  setActiveTab(tab.id);
                  setError(null);
                  if (tab.id === "my-lists") loadUserLists();
                  if (tab.id === "search" && !searchResults) handleSearchLists();
                }}
                className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
                  active
                    ? "bg-white dark:bg-stone-800 text-stone-900 dark:text-stone-100 shadow-2xs"
                    : "text-stone-600 dark:text-stone-400 hover:text-stone-900 dark:hover:text-stone-200"
                }`}
              >
                <Icon className="w-3.5 h-3.5 text-emerald-700 dark:text-emerald-400" />
                <span>{tab.label}</span>
              </button>
            );
          })}
        </div>
      </div>

      {error && <ErrorBanner message={error} onDismiss={() => setError(null)} />}

      {/* TAB 1: MY LISTS */}
      {activeTab === "my-lists" && (
        <div>
          {!account?.authenticated ? (
            <div className="p-8 rounded-xl border border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 text-center space-y-3">
              <p className="text-sm text-stone-600 dark:text-stone-400">
                Sign in to view and manage your personal Vocabulary.com lists.
              </p>
              <button
                type="button"
                onClick={onRequestLogin}
                className="px-4 py-2 rounded-lg bg-emerald-700 text-white text-xs font-medium"
              >
                Sign In
              </button>
            </div>
          ) : loading ? (
            <div className="p-12 flex justify-center">
              <Loader2 className="w-6 h-6 animate-spin text-emerald-700" />
            </div>
          ) : userLists.length > 0 ? (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {userLists.map((ul) => (
                <div
                  key={ul.wordlistid}
                  className="p-5 rounded-xl border border-stone-200/90 dark:border-stone-800 bg-white dark:bg-stone-900/80 flex flex-col justify-between gap-4"
                >
                  <div>
                    <div className="flex items-center justify-between gap-2">
                      <button
                        type="button"
                        onClick={() => openListDetail(ul.wordlistid)}
                        className="text-base font-editorial font-semibold text-stone-900 dark:text-stone-100 hover:text-emerald-700 dark:hover:text-emerald-400 text-left"
                      >
                        {ul.name}
                      </button>
                      <span className="text-xs font-mono text-stone-400 shrink-0">
                        {ul.wordcount} words
                      </span>
                    </div>
                    {ul.description && (
                      <p className="text-xs text-stone-500 dark:text-stone-400 mt-1 line-clamp-2">
                        {ul.description}
                      </p>
                    )}
                  </div>
                  <div className="flex items-center gap-2 pt-2 border-t border-stone-100 dark:border-stone-800">
                    <button
                      type="button"
                      onClick={() => openListDetail(ul.wordlistid)}
                      className="px-3 py-1.5 text-xs font-medium rounded-md border border-stone-200 dark:border-stone-800 hover:bg-stone-100 dark:hover:bg-stone-800"
                    >
                      Open List
                    </button>
                    <button
                      type="button"
                      onClick={() => onStartPractice(ul.wordlistid)}
                      className="px-3 py-1.5 text-xs font-medium rounded-md bg-emerald-700 hover:bg-emerald-800 text-white"
                    >
                      Practice
                    </button>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="p-8 rounded-xl border border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 text-center text-sm text-stone-500">
              You haven't created or practiced any personal lists yet.
            </div>
          )}
        </div>
      )}

      {/* TAB 2: SEARCH PUBLIC LISTS */}
      {activeTab === "search" && (
        <div className="space-y-4">
          <form onSubmit={handleSearchLists} className="flex items-center gap-2">
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search 70,000+ vocabulary lists (e.g. SAT, Shakespeare, Biology)..."
              className="flex-1 px-4 py-2 text-sm rounded-xl border border-stone-300 dark:border-stone-700 bg-white dark:bg-stone-900"
            />
            <button
              type="submit"
              disabled={loading}
              className="px-4 py-2 rounded-xl bg-emerald-700 hover:bg-emerald-800 text-white text-xs font-medium"
            >
              Search
            </button>
          </form>

          {searchResults?.results && searchResults.results.length > 0 && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {searchResults.results.map((item) => (
                <div
                  key={item.wordlistid}
                  className="p-5 rounded-xl border border-stone-200/90 dark:border-stone-800 bg-white dark:bg-stone-900/80 flex flex-col justify-between gap-3"
                >
                  <div>
                    <div className="flex items-center justify-between gap-2">
                      <button
                        type="button"
                        onClick={() => openListDetail(item.wordlistid)}
                        className="text-base font-editorial font-semibold text-stone-900 dark:text-stone-100 hover:text-emerald-700 dark:hover:text-emerald-400 text-left"
                      >
                        {item.name}
                      </button>
                      <span className="text-xs font-mono text-stone-400 shrink-0">
                        {item.wordcount} words
                      </span>
                    </div>
                    {item.description && (
                      <p className="text-xs text-stone-500 dark:text-stone-400 mt-1 line-clamp-2">
                        {item.description}
                      </p>
                    )}
                  </div>
                  <div className="flex items-center gap-2 pt-2 border-t border-stone-100 dark:border-stone-800">
                    <button
                      type="button"
                      onClick={() => openListDetail(item.wordlistid)}
                      className="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium rounded-md border border-stone-200 dark:border-stone-800 hover:bg-stone-100 dark:hover:bg-stone-800"
                    >
                      <BookOpen className="w-3.5 h-3.5" />
                      <span>Inspect Words</span>
                    </button>
                    <button
                      type="button"
                      onClick={() => onStartPractice(item.wordlistid)}
                      className="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium rounded-md bg-emerald-700 hover:bg-emerald-800 text-white"
                    >
                      <Play className="w-3 h-3 fill-current" />
                      <span>Practice</span>
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* TAB 3: CREATE NEW LIST */}
      {activeTab === "create" && (
        <form
          onSubmit={handleCreateList}
          className="rounded-xl border border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 p-6 space-y-4 max-w-2xl"
        >
          <h2 className="text-base font-semibold text-stone-900 dark:text-stone-100">
            Create a New Vocabulary List
          </h2>
          <div>
            <label className="block text-xs font-medium text-stone-500 mb-1">List Title</label>
            <input
              type="text"
              value={newListName}
              onChange={(e) => setNewListName(e.target.value)}
              placeholder="e.g. Economist Reading Words"
              className="w-full px-3.5 py-2 text-sm rounded-lg border border-stone-300 dark:border-stone-700 bg-stone-50 dark:bg-stone-950"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-stone-500 mb-1">
              Description (optional)
            </label>
            <input
              type="text"
              value={newListDesc}
              onChange={(e) => setNewListDesc(e.target.value)}
              placeholder="Notes or source context..."
              className="w-full px-3.5 py-2 text-sm rounded-lg border border-stone-300 dark:border-stone-700 bg-stone-50 dark:bg-stone-950"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-stone-500 mb-1">
              Words (comma or newline separated)
            </label>
            <textarea
              rows={5}
              value={newListWordsRaw}
              onChange={(e) => setNewListWordsRaw(e.target.value)}
              placeholder="callow, perspicacious, ephemeral..."
              className="w-full px-3.5 py-2 text-sm font-mono rounded-lg border border-stone-300 dark:border-stone-700 bg-stone-50 dark:bg-stone-950"
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="px-5 py-2.5 rounded-lg bg-emerald-700 hover:bg-emerald-800 text-white text-xs font-medium"
          >
            {loading ? "Saving List..." : "Create Vocabulary List"}
          </button>
        </form>
      )}

      {/* TAB 4: INSTALIST VOCABGRABBER */}
      {activeTab === "instalist" && (
        <div className="space-y-6">
          <form
            onSubmit={handleGrabVocabulary}
            className="rounded-xl border border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 p-6 space-y-4"
          >
            <h2 className="text-base font-semibold text-stone-900 dark:text-stone-100">
              InstaList · Extract Learnable Vocabulary from Any Text
            </h2>
            <textarea
              rows={5}
              value={grabText}
              onChange={(e) => setGrabText(e.target.value)}
              placeholder="Paste an article, chapter, or essay here to extract its most relevant vocabulary words..."
              className="w-full px-3.5 py-2.5 text-sm rounded-lg border border-stone-300 dark:border-stone-700 bg-stone-50 dark:bg-stone-950"
            />
            <div className="flex items-center gap-3">
              <button
                type="submit"
                disabled={loading || !grabText.trim()}
                className="px-4 py-2 rounded-lg bg-emerald-700 hover:bg-emerald-800 text-white text-xs font-medium"
              >
                {loading ? "Analyzing Text..." : "Extract Vocabulary"}
              </button>
              {grabbedWords.length > 0 && (
                <button
                  type="button"
                  onClick={() => {
                    setNewListWordsRaw(grabbedWords.map((w) => w.word).join(", "));
                    setActiveTab("create");
                  }}
                  className="px-4 py-2 rounded-lg border border-stone-200 dark:border-stone-800 text-xs font-medium"
                >
                  Send {grabbedWords.length} Words to List Creator →
                </button>
              )}
            </div>
          </form>

          {grabbedWords.length > 0 && (
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3">
              {grabbedWords.map((gw) => (
                <button
                  key={gw.word}
                  type="button"
                  onClick={() => onOpenWord(gw.word)}
                  className="p-3.5 rounded-xl border border-stone-200 dark:border-stone-800 bg-white dark:bg-stone-900 text-left hover:border-emerald-500 transition-colors"
                >
                  <div className="flex items-center justify-between">
                    <span className="font-editorial font-semibold text-stone-900 dark:text-stone-100">
                      {gw.word}
                    </span>
                    {gw.diff && (
                      <span className="text-[10px] font-mono text-stone-400">
                        Diff {gw.diff}
                      </span>
                    )}
                  </div>
                  <p className="text-xs text-stone-500 dark:text-stone-400 mt-1 line-clamp-2">
                    {gw.def}
                  </p>
                </button>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
};
