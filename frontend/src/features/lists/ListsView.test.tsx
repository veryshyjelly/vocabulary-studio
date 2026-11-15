import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ListsView } from "./ListsView";
import { SessionStoreMode, type AccountStatusDTO, type ListProgressStatus, type VocabularyList } from "../../lib/bridge";

const bridge = vi.hoisted(() => ({
  GetListDetail: vi.fn(), GetListProgress: vi.fn(), GetUserLists: vi.fn(),
  SearchLists: vi.fn(), GetFlashcards: vi.fn(),
}));

vi.mock("../../lib/bridge", async (original) => {
  const actual = await original<typeof import("../../lib/bridge")>();
  return { ...actual, getBridge: () => ({
    ...actual.getBridge(),
    lists: { ...actual.getBridge().lists, ...bridge },
    progress: { ...actual.getBridge().progress, GetListProgress: bridge.GetListProgress },
  }) };
});

const account = (uid = "account-a"): AccountStatusDTO => ({
  authenticated: true, valid_user: true, uid, nickname: uid, points: 0,
  level: { id: "L01", name: "Novice", milestone: 0, progress: 0 }, paid: false, full_vocab_trainer: false,
  session_store_mode: SessionStoreMode.SessionStoreMemory,
});
const list = (id: number): VocabularyList => ({
  wordlistid: id, name: `List ${id}`, shared: true,
  words: [{ word: "callow", definition: "young and inexperienced", learnable: true }],
});
const progress = (value: number): ListProgressStatus => ({
  learnable_word_count: 1, progress: value, learning: true, starred: false,
});
function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: Error) => void;
  const promise = new Promise<T>((res, rej) => { resolve = res; reject = rej; });
  return { promise, resolve, reject };
}
const props = {
  account: account(), initialListId: 1,
  onStartPractice: vi.fn(), onStartBee: vi.fn(), onOpenWord: vi.fn(), onRequestLogin: vi.fn(),
};

describe("list detail request ownership", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    bridge.GetListDetail.mockImplementation(async (id: number) => list(id));
    bridge.GetListProgress.mockResolvedValue(progress(0.25));
    bridge.GetUserLists.mockResolvedValue([]);
    bridge.SearchLists.mockResolvedValue({ results: [] });
    bridge.GetFlashcards.mockResolvedValue({ wordlist: { words: [
      { word: "callow", learnable: true, sense: { id: 1, definition: "young and inexperienced" } },
    ] } });
  });

  it("preserves details and back navigation when optional progress fails", async () => {
    bridge.GetListProgress.mockRejectedValue(new Error("progress unavailable"));
    const user = userEvent.setup();
    render(<ListsView {...props} />);
    expect(await screen.findByRole("heading", { name: "List 1" })).toBeInTheDocument();
    expect(screen.getByText("young and inexperienced")).toBeInTheDocument();
    expect(screen.queryByText(/progress unavailable/)).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Back to Vocabulary Lists" }));
    expect(screen.getByRole("heading", { name: "Vocabulary Lists" })).toBeInTheDocument();
  });

  it("preserves keyboard flashcard flipping", async () => {
    const user = userEvent.setup();
    render(<ListsView {...props} />);
    await user.click(await screen.findByRole("button", { name: /Flashcards/ }));
    const card = await screen.findByRole("button", { name: /Flip flashcard for callow/ });
    card.focus();
    await user.keyboard(" ");
    expect(card).toHaveTextContent("young and inexperienced");
  });

  it("keeps the selected list's progress when responses finish in reverse order", async () => {
    const old = deferred<ListProgressStatus>();
    bridge.GetListProgress.mockReturnValueOnce(old.promise).mockResolvedValueOnce(progress(0.8));
    const view = render(<ListsView {...props} />);
    await screen.findByRole("heading", { name: "List 1" });
    view.rerender(<ListsView {...props} initialListId={2} />);
    await screen.findByText(/Mastery 80%/);
    await act(async () => old.resolve(progress(0.1)));
    expect(screen.getByText(/Mastery 80%/)).toBeInTheDocument();
    expect(screen.queryByText(/Mastery 10%/)).not.toBeInTheDocument();
  });

  it("clears previous progress and ignores an obsolete progress failure", async () => {
    const old = deferred<ListProgressStatus>();
    bridge.GetListProgress.mockReturnValueOnce(old.promise).mockResolvedValueOnce(progress(0.8));
    const view = render(<ListsView {...props} />);
    await screen.findByRole("heading", { name: "List 1" });
    view.rerender(<ListsView {...props} initialListId={2} />);
    await screen.findByText(/Mastery 80%/);
    await act(async () => old.reject(new Error("old progress failed")));
    expect(screen.getByText(/Mastery 80%/)).toBeInTheDocument();
  });

  it("clears already displayed progress while the next list is loading", async () => {
    const current = deferred<ListProgressStatus>();
    bridge.GetListProgress.mockResolvedValueOnce(progress(0.25)).mockReturnValueOnce(current.promise);
    const view = render(<ListsView {...props} />);
    await screen.findByText(/Mastery 25%/);
    view.rerender(<ListsView {...props} initialListId={2} />);
    await screen.findByRole("heading", { name: "List 2" });
    expect(screen.queryByText(/Mastery/)).not.toBeInTheDocument();
    await act(async () => current.resolve(progress(0.8)));
    expect(screen.getByText(/Mastery 80%/)).toBeInTheDocument();
  });

  it("ignores obsolete detail success after the latest list has loaded", async () => {
    const old = deferred<VocabularyList>();
    bridge.GetListDetail.mockReturnValueOnce(old.promise).mockResolvedValueOnce(list(2));
    const view = render(<ListsView {...props} />);
    view.rerender(<ListsView {...props} initialListId={2} />);
    await screen.findByRole("heading", { name: "List 2" });
    await act(async () => old.resolve(list(1)));
    expect(screen.getByRole("heading", { name: "List 2" })).toBeInTheDocument();
    expect(bridge.GetListProgress).toHaveBeenCalledTimes(1);
    expect(bridge.GetListProgress).toHaveBeenCalledWith(2);
  });

  it("does not restore pending details after leaving the list tools", async () => {
    const detail = deferred<VocabularyList>();
    bridge.GetListDetail.mockReturnValueOnce(detail.promise);
    const user = userEvent.setup();
    render(<ListsView {...props} />);
    await user.click(screen.getByRole("tab", { name: "Create List" }));
    await act(async () => detail.resolve(list(1)));
    expect(screen.queryByRole("heading", { name: "List 1" })).not.toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Create List" })).toHaveAttribute("aria-selected", "true");
  });

  it("ignores obsolete detail errors without ending current loading", async () => {
    const old = deferred<VocabularyList>();
    const current = deferred<VocabularyList>();
    bridge.GetListDetail.mockReturnValueOnce(old.promise).mockReturnValueOnce(current.promise);
    const view = render(<ListsView {...props} />);
    view.rerender(<ListsView {...props} initialListId={2} />);
    await act(async () => old.reject(new Error("old detail failed")));
    expect(screen.queryByText("old detail failed")).not.toBeInTheDocument();
    expect(screen.getByText(/Loading your lists/)).toBeInTheDocument();
    await act(async () => current.reject(new Error("current detail failed")));
    expect(screen.getByText("current detail failed")).toBeInTheDocument();
  });

  it("ignores previous-account progress even when both accounts are authenticated", async () => {
    const old = deferred<ListProgressStatus>();
    bridge.GetListProgress.mockReturnValueOnce(old.promise).mockResolvedValueOnce(progress(0.6));
    const view = render(<ListsView {...props} />);
    await screen.findByRole("heading", { name: "List 1" });
    view.rerender(<ListsView {...props} account={account("account-b")} />);
    await screen.findByText(/Mastery 60%/);
    await act(async () => old.resolve(progress(0.1)));
    expect(screen.getByText(/Mastery 60%/)).toBeInTheDocument();
  });

  it("does not request enrichment after pending details finish on an unmounted view", async () => {
    const detail = deferred<VocabularyList>();
    bridge.GetListDetail.mockReturnValueOnce(detail.promise);
    const view = render(<ListsView {...props} />);
    view.unmount();
    await act(async () => detail.resolve(list(1)));
    expect(bridge.GetListProgress).not.toHaveBeenCalled();
  });

  it("does not restore pending flashcards after back navigation", async () => {
    const cards = deferred<unknown>();
    bridge.GetFlashcards.mockReturnValueOnce(cards.promise);
    const user = userEvent.setup();
    render(<ListsView {...props} />);
    await user.click(await screen.findByRole("button", { name: /Flashcards/ }));
    await user.click(screen.getByRole("button", { name: "Back to Vocabulary Lists" }));
    await act(async () => cards.reject(new Error("obsolete flashcards failed")));
    expect(screen.getByRole("heading", { name: "Vocabulary Lists" })).toBeInTheDocument();
    expect(screen.queryByText("obsolete flashcards failed")).not.toBeInTheDocument();
  });

  it("does not show authenticated progress after sign-out", async () => {
    const old = deferred<ListProgressStatus>();
    bridge.GetListProgress.mockReturnValueOnce(old.promise);
    const view = render(<ListsView {...props} />);
    await screen.findByRole("heading", { name: "List 1" });
    view.rerender(<ListsView {...props} account={null} />);
    await screen.findByRole("heading", { name: "List 1" });
    await act(async () => old.resolve(progress(0.9)));
    expect(screen.queryByText(/Mastery/)).not.toBeInTheDocument();
    expect(bridge.GetListProgress).toHaveBeenCalledTimes(1);
  });
});
