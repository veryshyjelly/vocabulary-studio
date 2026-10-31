import React, { useState, useEffect, useCallback } from "react";
import { ListChecks, Search, Plus, Sparkles, Play, SpellCheck, Layers, Trash2, Star, ArrowLeft, ArrowRight, RotateCw, BookOpen } from "lucide-react";
import type { AccountStatusDTO, ProfileListSummary, SmartSearchResponse, VocabularyList, FlashcardListResponse, ListProgressStatus, GrabbedWord } from "../../lib/bridge";
import { getBridge, formatError } from "../../lib/bridge";
import { Button, Dialog, EmptyState, LoadingBlock, PageHeader, Tabs } from "../../components/ui";
import { AudioButton, ErrorBanner, Kbd } from "../../components/shared";
import { acceptsPlainShortcut } from "../../lib/keys";

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
  const [listFilter, setListFilter] = useState("");
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
      if (!acceptsPlainShortcut(e) || words.length === 0) return;
      if ((e.key === " " || e.key === "Enter") && (e.target as HTMLElement)?.closest("button")) return;
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
        <div className="flex flex-wrap items-center justify-between gap-3">
          <button
            type="button"
            onClick={() => {
              setSelectedList(null);
              setShowFlashcards(false);
            }}
            className="inline-flex items-center gap-1.5 text-xs font-medium text-ink-muted hover:text-ink"
          >
            <ArrowLeft className="w-4 h-4" />
            <span>Back to Vocabulary Lists</span>
          </button>

          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => handleToggleFavorite(true)}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-line bg-surface text-xs font-medium text-ink-muted hover:bg-sunken"
            >
              <Star className="w-3.5 h-3.5 text-warning-ink" />
              <span>Bookmark</span>
            </button>
            {account?.authenticated && (
              <button
                type="button"
                onClick={() => setConfirmDelete(true)}
                className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-danger-line text-danger-ink text-xs font-medium hover:bg-danger-soft"
              >
                <Trash2 className="w-3.5 h-3.5" />
                <span>Delete</span>
              </button>
            )}
          </div>
        </div>

        {error && <ErrorBanner message={error} onDismiss={() => setError(null)} />}
        {notice && (
          <div className="p-3 rounded-lg bg-accent-soft border border-accent-line text-xs text-accent-ink">
            {notice}
          </div>
        )}

        {/* Destructive Delete Confirmation Dialog */}
        <Dialog open={confirmDelete} onClose={() => { if (!loading) setConfirmDelete(false); }} labelledBy="delete-list-title">
          <div className="p-6 space-y-4">
            <h2 id="delete-list-title" className="font-nav text-ui font-semibold">Delete this list?</h2>
            <p className="reading-copy text-ink-muted">“{selectedList.name}” will be permanently deleted from your account.</p>
            <div className="flex justify-end gap-2">
              <Button onClick={() => setConfirmDelete(false)} disabled={loading}>Cancel</Button>
              <Button variant="danger" onClick={handleDeleteCurrentList} loading={loading}>Confirm Delete</Button>
            </div>
          </div>
        </Dialog>

        {/* List Header Card */}
        <div className="rounded-xl border border-line/90 bg-surface p-6 md:p-8 space-y-5">
          <div className="flex flex-col md:flex-row md:items-start justify-between gap-4">
            <div className="space-y-2 max-w-2xl">
              <span className="text-xs font-data text-ink-subtle">
                List #{selectedList.wordlistid} · {selectedList.words?.length || 0} words
                {listProgress
                  ? ` · Mastery ${Math.round((listProgress.progress || 0) * 100)}%`
                  : ""}
              </span>
              <h1 className="font-word text-title font-semibold text-ink break-words">
                {selectedList.name}
              </h1>
              {selectedList.description && (
                <p className="reading-copy text-ink-muted">
                  {selectedList.description}
                </p>
              )}
            </div>

            <div className="flex flex-wrap items-center gap-2 shrink-0">
              <button
                type="button"
                onClick={() => onStartPractice(selectedList.wordlistid)}
                className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-accent hover:bg-accent-hover text-on-accent text-xs font-medium shadow-xs"
              >
                <Play className="w-3.5 h-3.5 fill-current" />
                <span>Practice List</span>
              </button>
              <button
                type="button"
                onClick={() => onStartBee(selectedList.wordlistid)}
                className="inline-flex items-center gap-2 px-3.5 py-2 rounded-lg border border-line hover:bg-sunken text-xs font-medium text-ink-muted"
              >
                <SpellCheck className="w-3.5 h-3.5 text-accent-ink" />
                <span>Spelling Bee</span>
              </button>
              <button
                type="button"
                onClick={() =>
                  showFlashcards ? setShowFlashcards(false) : handleOpenFlashcards()
                }
                className="inline-flex items-center gap-2 px-3.5 py-2 rounded-lg border border-line hover:bg-sunken text-xs font-medium text-ink-muted"
              >
                <Layers className="w-3.5 h-3.5 text-accent-ink" />
                <span>{showFlashcards ? "Hide Flashcards" : "Flashcards"}</span>
              </button>
            </div>
          </div>

          {/* Add Words Form */}
          {account?.authenticated && (
            <form onSubmit={handleAddWordsToCurrentList} className="flex items-center gap-2 pt-2">
              <input
                type="text"
                aria-label="Words to add to this list" value={addWordsInput}
                onChange={(e) => setAddWordsInput(e.target.value)}
                placeholder="Add words to this list (comma-separated)..."
                className="flex-1 px-3 py-1.5 text-xs rounded-lg border border-line bg-sunken"
              />
              <button
                type="submit"
                disabled={!addWordsInput.trim() || loading}
                className="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg bg-accent text-on-accent text-xs font-medium disabled:opacity-40"
              >
                <Plus className="w-3.5 h-3.5" />
                <span>Add Words</span>
              </button>
            </form>
          )}
        </div>

        {/* Interactive Flashcards Deck */}
        {showFlashcards && currentCard && (
          <div className="rounded-xl border border-accent-line bg-surface p-8 shadow-sm space-y-6">
            <div className="flex items-center justify-between text-xs text-ink-subtle">
              <span>
                Flashcard {cardIndex + 1} of {fcWords.length}
              </span>
              <span className="flex items-center gap-1.5">
                <Kbd>Space</Kbd> Flip · <Kbd>←</Kbd> <Kbd>→</Kbd> Navigate
              </span>
            </div>

            <div className="study-sheet overflow-hidden">
              <button type="button" onClick={() => setCardFlipped((f) => !f)}
                aria-label={`Flip flashcard for ${currentCard.word}`}
                aria-pressed={cardFlipped}
                className="w-full min-h-60 flex flex-col items-center justify-center text-center p-8 hover:bg-sunken/50 transition-colors space-y-4">
                {!cardFlipped ? (
                  <>
                    <span className="word-display text-ink">{currentCard.word}</span>
                    {currentCard.sense?.ipas?.[0]?.formatted_ipastring && <span className="font-ipa text-ui text-ink-muted">{currentCard.sense.ipas[0].formatted_ipastring}</span>}
                    <span className="text-meta text-ink-subtle pt-2">Click or press Space to reveal definition</span>
                  </>
                ) : (
                  <>
                    {currentCard.sense?.part_of_speech && <span className="eyebrow text-accent-ink">{currentCard.sense.part_of_speech}</span>}
                    <span className="font-read text-read-lg text-ink max-w-xl">{currentCard.sense?.definition || "Definition unavailable"}</span>
                    {currentCard.example?.text && <span className="font-read text-read italic text-ink-muted max-w-xl">“{currentCard.example.text}”</span>}
                  </>
                )}
              </button>
              <div className="flex justify-center pb-4"><AudioButton word={currentCard.word} size="sm" label="Listen" /></div>
            </div>

            <div className="flex flex-wrap items-center justify-between gap-3">
              <button
                type="button"
                onClick={() => {
                  setCardFlipped(false);
                  setCardIndex((i) => (i > 0 ? i - 1 : fcWords.length - 1));
                }}
                className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg border border-line text-xs font-medium"
              >
                <ArrowLeft className="w-3.5 h-3.5" />
                <span>Previous</span>
              </button>
              <button
                type="button"
                onClick={() => setCardFlipped((f) => !f)}
                className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg border border-line text-xs font-medium"
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
                className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg border border-line text-xs font-medium"
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
              className="p-4 rounded-xl border border-line/80 bg-surface flex flex-col justify-between gap-2"
            >
              <div className="space-y-1">
                <div className="flex items-center justify-between gap-2">
                  <button
                    type="button"
                    onClick={() => onOpenWord(w.word)}
                    className="font-word text-[1.125rem] font-semibold text-ink hover:text-accent-ink break-words text-left"
                  >
                    {w.word}
                  </button>
                  <AudioButton word={w.word} size="sm" />
                </div>
                <p className="font-read text-read text-ink-muted">
                  {w.definition || w.shortdefinition}
                </p>
                {w.example?.text && (
                  <p className="text-xs font-word italic text-ink-subtle pt-1">
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
      <PageHeader title="Vocabulary Lists" description="Your word collections, ready for the next study session.">
        <Tabs<ListsTab>
          label="List tools"
          value={activeTab}
          onChange={(id) => {
            setActiveTab(id);
            setError(null);
            if (id === "my-lists") loadUserLists();
            if (id === "search" && !searchResults) handleSearchLists();
          }}
          items={[
            { id: "my-lists", label: "My Lists", icon: ListChecks },
            { id: "search", label: "Search Lists", icon: Search },
            { id: "create", label: "Create List", icon: Plus },
            { id: "instalist", label: "InstaList Extractor", icon: Sparkles },
          ]}
        />
      </PageHeader>

      {error && <ErrorBanner message={error} onDismiss={() => setError(null)} />}

      {/* TAB 1: MY LISTS */}
      {activeTab === "my-lists" && (
        <div className="space-y-4">
          {account?.authenticated && (
            <div className="flex flex-wrap items-center justify-between gap-3">
              <p className="font-nav text-meta text-ink-muted">{userLists.length.toLocaleString()} personal lists</p>
              <input aria-label="Filter personal lists" placeholder="Filter your lists…" value={listFilter}
                onChange={(e) => setListFilter(e.target.value)}
                className="w-full sm:w-64 px-3 py-2 rounded-control border border-line bg-surface text-ui" />
            </div>
          )}
          {!account?.authenticated ? (
            <div className="p-8 rounded-xl border border-line bg-surface text-center space-y-3">
              <p className="text-sm text-ink-muted">
                Sign in to view and manage your personal Vocabulary.com lists.
              </p>
              <button
                type="button"
                onClick={onRequestLogin}
                className="px-4 py-2 rounded-lg bg-accent text-on-accent text-xs font-medium"
              >
                Sign In
              </button>
            </div>
          ) : loading ? (
            <LoadingBlock label="Loading your lists…" />
          ) : userLists.filter((list) => list.name.toLowerCase().includes(listFilter.toLowerCase())).length > 0 ? (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {userLists.filter((list) => list.name.toLowerCase().includes(listFilter.toLowerCase())).map((ul) => (
                <div
                  key={ul.wordlistid}
                  className="library-item p-5 flex flex-col justify-between gap-4"
                >
                  <div>
                    <div className="flex items-center justify-between gap-2">
                      <button
                        type="button"
                        onClick={() => openListDetail(ul.wordlistid)}
                        className="min-w-0 flex-1 font-nav text-[0.9375rem] font-semibold text-ink hover:text-accent-ink text-left break-words"
                      >
                        {ul.name}
                      </button>
                      <span className="text-xs font-data text-ink-subtle shrink-0">
                        {ul.wordcount} words
                      </span>
                    </div>
                    {ul.description && (
                      <p className="font-read text-[1rem] leading-relaxed text-ink-muted mt-2 line-clamp-2">
                        {ul.description}
                      </p>
                    )}
                  </div>
                  <div className="flex items-center gap-2 pt-2 border-t border-line">
                    <button
                      type="button"
                      onClick={() => openListDetail(ul.wordlistid)}
                      className="px-3 py-1.5 text-xs font-medium rounded-md border border-line hover:bg-sunken"
                    >
                      Open List
                    </button>
                    <button
                      type="button"
                      onClick={() => onStartPractice(ul.wordlistid)}
                      className="px-3 py-1.5 text-xs font-medium rounded-md bg-accent hover:bg-accent-hover text-on-accent"
                    >
                      Practice
                    </button>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <EmptyState icon={ListChecks} title={listFilter ? "No lists match your search" : "A collection starts with one word"}
              description={listFilter ? "Try a different name or clear your filter." : "Create a list for the words you want to remember."}
              action={<Button onClick={() => listFilter ? setListFilter("") : setActiveTab("create")}>{listFilter ? "Clear filter" : "Create List"}</Button>} />
          )}
        </div>
      )}

      {/* TAB 2: SEARCH PUBLIC LISTS */}
      {activeTab === "search" && (
        <div className="space-y-4">
          <form onSubmit={handleSearchLists} className="flex items-center gap-2">
            <input
              aria-label="Search public vocabulary lists"
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search 70,000+ vocabulary lists (e.g. SAT, Shakespeare, Biology)..."
              className="flex-1 px-4 py-2 text-sm rounded-xl border border-line-strong bg-surface"
            />
            <button
              type="submit"
              disabled={loading}
              className="px-4 py-2 rounded-xl bg-accent hover:bg-accent-hover text-on-accent text-xs font-medium"
            >
              Search
            </button>
          </form>

          {loading && <LoadingBlock label="Searching vocabulary lists…" />}
          {!loading && searchResults?.results?.length === 0 && !error && <EmptyState icon={Search} title="No lists found" description="Try a different subject, exam, or author." />}
          {!loading && searchResults?.results && searchResults.results.length > 0 && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {searchResults.results.map((item) => (
                <div
                  key={item.wordlistid}
                  className="library-item p-5 flex flex-col justify-between gap-3"
                >
                  <div>
                    <div className="flex items-center justify-between gap-2">
                      <button
                        type="button"
                        onClick={() => openListDetail(item.wordlistid)}
                        className="min-w-0 flex-1 font-nav text-[0.9375rem] font-semibold text-ink hover:text-accent-ink text-left break-words"
                      >
                        {item.name}
                      </button>
                      <span className="text-xs font-data text-ink-subtle shrink-0">
                        {item.wordcount} words
                      </span>
                    </div>
                    {item.description && (
                      <p className="font-read text-[1rem] leading-relaxed text-ink-muted mt-2 line-clamp-2">
                        {item.description}
                      </p>
                    )}
                  </div>
                  <div className="flex items-center gap-2 pt-2 border-t border-line">
                    <button
                      type="button"
                      onClick={() => openListDetail(item.wordlistid)}
                      className="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium rounded-md border border-line hover:bg-sunken"
                    >
                      <BookOpen className="w-3.5 h-3.5" />
                      <span>Inspect Words</span>
                    </button>
                    <button
                      type="button"
                      onClick={() => onStartPractice(item.wordlistid)}
                      className="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium rounded-md bg-accent hover:bg-accent-hover text-on-accent"
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
          className="rounded-xl border border-line bg-surface p-6 space-y-4 max-w-2xl"
        >
          <h2 className="text-base font-semibold text-ink">
            Create a New Vocabulary List
          </h2>
          <div>
            <label htmlFor="list-title" className="block font-nav text-meta font-medium text-ink-muted mb-1">List Title</label>
            <input
              type="text"
              id="list-title" value={newListName}
              onChange={(e) => setNewListName(e.target.value)}
              placeholder="e.g. Economist Reading Words"
              className="w-full px-3.5 py-2 text-sm rounded-lg border border-line-strong bg-sunken"
            />
          </div>
          <div>
            <label htmlFor="list-description" className="block font-nav text-meta font-medium text-ink-muted mb-1">
              Description (optional)
            </label>
            <input
              type="text"
              id="list-description" value={newListDesc}
              onChange={(e) => setNewListDesc(e.target.value)}
              placeholder="Notes or source context..."
              className="w-full px-3.5 py-2 text-sm rounded-lg border border-line-strong bg-sunken"
            />
          </div>
          <div>
            <label htmlFor="list-words" className="block font-nav text-meta font-medium text-ink-muted mb-1">
              Words (comma or newline separated)
            </label>
            <textarea
              rows={5}
              id="list-words" value={newListWordsRaw}
              onChange={(e) => setNewListWordsRaw(e.target.value)}
              placeholder="callow, perspicacious, ephemeral..."
              className="w-full px-3.5 py-2 text-sm font-data rounded-lg border border-line-strong bg-sunken"
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="px-5 py-2.5 rounded-lg bg-accent hover:bg-accent-hover text-on-accent text-xs font-medium"
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
            className="rounded-xl border border-line bg-surface p-6 space-y-4"
          >
            <h2 className="text-base font-semibold text-ink">
              InstaList · Extract Learnable Vocabulary from Any Text
            </h2>
            <textarea
              rows={5}
              aria-label="Text to extract vocabulary from" value={grabText}
              onChange={(e) => setGrabText(e.target.value)}
              placeholder="Paste an article, chapter, or essay here to extract its most relevant vocabulary words..."
              className="w-full px-3.5 py-2.5 text-sm rounded-lg border border-line-strong bg-sunken"
            />
            <div className="flex flex-wrap items-center gap-3">
              <button
                type="submit"
                disabled={loading || !grabText.trim()}
                className="px-4 py-2 rounded-lg bg-accent hover:bg-accent-hover text-on-accent text-xs font-medium"
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
                  className="px-4 py-2 rounded-lg border border-line text-xs font-medium"
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
                  className="p-3.5 rounded-xl border border-line bg-surface text-left hover:border-accent-line transition-colors"
                >
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <span className="font-word font-semibold text-ink">
                      {gw.word}
                    </span>
                    {gw.diff && (
                      <span className="text-[10px] font-data text-ink-subtle">
                        Diff {gw.diff}
                      </span>
                    )}
                  </div>
                  <p className="font-read text-read text-ink-muted mt-2 line-clamp-2">
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
