import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { App } from "./App";
import {
  getBridge,
  setBridgeForTesting,
  SessionStoreMode,
  type AccountStatusDTO,
  type TrainerSessionStateDTO,
  type WordDetailDTO,
} from "./lib/bridge";
import { TrainerSessionState } from "./bindings/github.com/veryshyjelly/vocabulary_go/models.js";

function createMockAccount(authenticated = true): AccountStatusDTO {
  return {
    authenticated,
    valid_user: authenticated,
    uid: "u_1",
    nickname: "Ayush",
    points: 12450,
    level: { id: "L04", name: "Wordsmith", milestone: 2, progress: 0.6 },
    paid: true,
    full_vocab_trainer: true,
    session_store_mode: SessionStoreMode.SessionStoreKeyring,
  };
}

function createMockQuestionSlide(isSpelling = false): TrainerSessionStateDTO {
  return {
    active: true,
    session_state: TrainerSessionState.SessionStateActiveQuestion,
    activity_type: "c",
    tries: 0,
    round: { number: 1, played_count: 0, streak: 3 },
    question: {
      turn: 1,
      type: isSpelling ? "T" : "D",
      type_label: isSpelling ? "Spell the Word" : "Definition",
      mode: "PROGRESS",
      point_value: 100,
      target_word: "callow",
      instructions_text: "In this sentence, callow means:",
      instructions_spans: [
        { text: "In this sentence, " },
        { text: "callow", bold: true },
        { text: " means:" },
      ],
      sentence_text: "What transformed them from callow copyists into titans was experience.",
      source_text: "The Guardian",
      is_spelling: isSpelling,
      is_image_grid: false,
      available_hints: ["F", "D", "E"],
      choices: isSpelling
        ? []
        : [
            { index: 0, nonce: "n0", text: "wealthy and influential" },
            { index: 1, nonce: "n1", text: "young and inexperienced" },
            { index: 2, nonce: "n2", text: "secretive and cautious" },
            { index: 3, nonce: "n3", text: "exhausted and weary" },
          ],
    },
  };
}

function createMockWordDetail(word = "callow"): WordDetailDTO {
  return {
    entry: {
      word,
      learnable: true,
      word_forms: ["callower", "callowest"],
      short_blurb: "Someone who is callow is young and inexperienced.",
      long_blurb: "Callow comes from the Old English word calu meaning bald.",
      pronunciations: [
        {
          region: "US",
          ipa: "/ˈkæloʊ/",
          audio_url: "https://audio.vocabulary.com/1.0/us/C/1XB5QC9VRN0HU.mp3",
        },
      ],
      senses: [
        {
          synset_id: 101016,
          part_of_speech: "adjective",
          definition: "young and inexperienced",
          examples: ['"a callow youth of seventeen"'],
          related: [{ relation: "synonyms", words: ["fledgling", "unfledged"] }],
        },
      ],
      word_family: [
        { word: "callow", hw: true, freq: 0.51, ffreq: 0.52, type: 0 },
        { word: "callowness", hw: true, parent: "callow", freq: 0.01, ffreq: 0.01, type: 2 },
      ],
    },
    examples: [
      {
        sentence: "What transformed Rush from callow copyists into titans was experience.",
        offsets: [27, 33],
        source_title: "The Guardian",
        source_url: "https://www.theguardian.com/example",
      },
    ],
  };
}

describe("Vocabulary Studio Desktop Frontend", () => {
  beforeEach(() => {
    setBridgeForTesting({
      settings: {
        GetPreferences: vi.fn().mockResolvedValue({
          theme: "light",
          pronunciation_accent: "US",
          default_view: "home",
          session_store_mode: SessionStoreMode.SessionStoreKeyring,
          search_history: ["callow", "ephemeral"],
          app_version: "1.0.0",
        }),
        UpdatePreferences: vi.fn().mockImplementation(async (p) => p),
      } as any,
      auth: {
        RestoreSession: vi.fn().mockResolvedValue(createMockAccount(false)),
        GetStatus: vi.fn().mockResolvedValue(createMockAccount(false)),
        Login: vi.fn().mockResolvedValue(createMockAccount(true)),
        Logout: vi.fn().mockResolvedValue(createMockAccount(false)),
      } as any,
      progress: {
        GetHomeDashboard: vi.fn().mockResolvedValue({
          account: createMockAccount(false),
          word_of_the_day: {
            date: "October 31, 2026",
            word: "enchantment",
            short_usage: "A sense of delight and escape from everyday reality.",
          },
          search_history: ["callow", "ephemeral"],
        }),
        GetLeaderboard: vi.fn().mockResolvedValue({
          domain: "individuals",
          period: "alltime",
          metric: "points",
          in_list: false,
          realtime: true,
          rows: [{ rank: 1, points: 95000, profile: { uid: "u9", nickname: "TopScholar", points: 95000, level: { id: "L10", name: "Verbivore", milestone: 5, progress: 0.9 } } }],
        }),
      } as any,
      dictionary: {
        Lookup: vi.fn().mockImplementation(async (w: string) => createMockWordDetail(w)),
        Autocomplete: vi.fn().mockResolvedValue([
          { word: "callow", definition: "young and inexperienced", ipa: "/ˈkæloʊ/", learnable: true, url: "/dictionary/callow" },
        ]),
        GetSearchHistory: vi.fn().mockResolvedValue(["callow", "ephemeral"]),
        ClearSearchHistory: vi.fn().mockResolvedValue(undefined),
        GetExamples: vi.fn().mockResolvedValue([]),
      } as any,
      trainer: {
        StartSession: vi.fn().mockResolvedValue(createMockQuestionSlide(false)),
        GetSessionState: vi.fn().mockResolvedValue(createMockQuestionSlide(false)),
        SubmitChoice: vi.fn().mockImplementation(async (idx: number) => {
          const isCorrect = idx === 1;
          const base = createMockQuestionSlide(false);
          return {
            ...base,
            tries: idx === 0 ? 1 : 2,
            session_state: isCorrect
              ? TrainerSessionState.SessionStateQuestionCompleted
              : TrainerSessionState.SessionStateActiveQuestion,
            last_feedback: {
              correct: isCorrect,
              correct_choice: 1,
              user_choice: idx,
              word: "callow",
              points_earned: isCorrect && idx === 1 ? 100 : 0,
              attempt: idx === 0 ? 1 : 2,
              evaluated_locally: idx !== 0,
              short_blurb_text: "Someone who is callow is young and inexperienced.",
              part_of_speech: "adjective",
              sense_definition: "young and inexperienced",
            },
          };
        }),
        NextQuestion: vi.fn().mockResolvedValue(createMockQuestionSlide(true)),
        SubmitSpelling: vi.fn().mockResolvedValue({
          ...createMockQuestionSlide(true),
          session_state: TrainerSessionState.SessionStateQuestionCompleted,
          last_feedback: {
            correct: true,
            correct_choice: 0,
            user_choice: 0,
            word: "callow",
            points_earned: 150,
            attempt: 1,
            evaluated_locally: false,
          },
        }),
      } as any,
      lists: {
        SearchLists: vi.fn().mockResolvedValue({
          totalResults: 1,
          hasMore: false,
          results: [{ wordlistid: 285407, name: "30 GRE Words", wordcount: 30, description: "Essential GRE vocabulary." }],
        }),
        GetListDetail: vi.fn().mockResolvedValue({
          wordlistid: 285407,
          name: "30 GRE Words",
          description: "Essential GRE vocabulary.",
          shared: true,
          words: [{ word: "callow", definition: "young and inexperienced", learnable: true }],
        }),
        GetFlashcards: vi.fn().mockResolvedValue({
          wordlist: {
            words: [
              {
                word: "callow",
                learnable: true,
                sense: { id: 1, definition: "young and inexperienced", part_of_speech: "adjective" },
              },
            ],
          },
        }),
      } as any,
      discovery: {
        GetRootCategories: vi.fn().mockResolvedValue([
          { unit_id: "gfpldfb3", tag: "LIT", title: "Literature", description: "Classic novels." },
        ]),
        SearchUnits: vi.fn().mockResolvedValue({
          units: [{ id: "fssljomy", title: "Brave New World", type: "work", author: "Aldous Huxley" }],
        }),
      } as any,
    });
  });

  it("renders the Home dashboard with Word of the Day and supports signing in via AuthModal", async () => {
    const user = userEvent.setup();
    render(<App />);

    expect(await screen.findByText("enchantment")).toBeInTheDocument();
    expect(screen.getByText(/What would you like to study today/i)).toBeInTheDocument();

    // Open Sign-In Modal
    const signInButtons = screen.getAllByRole("button", { name: /Sign In/i });
    await user.click(signInButtons[0]);

    const dialog = await screen.findByRole("dialog");
    expect(dialog).toBeInTheDocument();
    await user.type(within(dialog).getByLabelText(/Email or Username/i), "ayush@example.com");
    await user.type(within(dialog).getByLabelText(/^Password$/i), "secret123");
    await user.click(within(dialog).getByRole("button", { name: /^Sign In$/i }));

    await waitFor(() => {
      expect(screen.getByText("Ayush")).toBeInTheDocument();
    });
  });

  it("supports interactive Practice with keyboard shortcuts, local retry feedback, and advancing to spelling", async () => {
    const user = userEvent.setup();
    render(<App />);

    // Navigate to Practice
    await user.click(await screen.findByRole("button", { name: /^Practice$/i }));

    expect(await screen.findByText(/wealthy and influential/i)).toBeInTheDocument();

    // 1st attempt: press key "1" (wrong choice index 0) -> Server-recorded feedback
    await user.keyboard("1");
    expect(await screen.findByText(/Attempt 1: Keep trying!/i)).toBeInTheDocument();
    expect(screen.getByText(/Answer saved to your session/i)).toBeInTheDocument();

    // 2nd attempt: press key "2" (correct choice index 1) -> Locally evaluated retry
    await user.keyboard("2");
    expect(await screen.findByText(/Practice retry · keep going/i)).toBeInTheDocument();

    // Press Enter to advance to next question (which is a Spelling question)
    await user.keyboard("{Enter}");
    const spellInput = await screen.findByLabelText(/Word spelling input/i);
    await user.type(spellInput, "callow{Enter}");

    expect(await screen.findByText(/Correct on first try!/i)).toBeInTheDocument();
    await user.keyboard("{Enter}");
    await waitFor(() => expect(screen.getByLabelText(/Word spelling input/i)).toBeEnabled());
  });

  it("supports Dictionary lookup, Word Family navigation, and Lists Flashcards", async () => {
    const user = userEvent.setup();
    render(<App />);

    // Navigate to Dictionary
    await user.click(await screen.findByRole("button", { name: /^Dictionary$/i }));

    expect(await screen.findByText(/Someone who is callow is young and inexperienced/i)).toBeInTheDocument();
    expect(screen.getByText("callowness")).toBeInTheDocument();

    // Click related word in Word Family
    await user.click(screen.getByRole("button", { name: /callowness/i }));
    await waitFor(() => {
      expect(screen.getByRole("heading", { name: "callowness" })).toBeInTheDocument();
    });

    // Navigate to My Lists, open list detail, and flip flashcard
    await user.click(screen.getByRole("button", { name: /^My Lists$/i }));
    const inspectBtn = await screen.findByRole("button", { name: /Inspect Words/i });
    await user.click(inspectBtn);

    const flashcardsBtn = await screen.findByRole("button", { name: /Flashcards/i });
    await user.click(flashcardsBtn);

    expect(await screen.findByText(/Click or press Space to reveal definition/i)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Flip Card/i }));
    const defs = await screen.findAllByText(/young and inexperienced/i);
    expect(defs.length).toBeGreaterThanOrEqual(2);

    // Navigate to Discover and Settings
    await user.click(screen.getByRole("button", { name: /^Discover$/i }));
    expect(await screen.findByText(/Brave New World/i)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /^Settings$/i }));
    expect(await screen.findByRole("heading", { name: /Session Storage Security/i })).toBeInTheDocument();
  });

  it("handles lifeline hints, 4-image visual questions, error retry states, and Progress/Settings flows", async () => {
    const user = userEvent.setup();

    let lookupCalls = 0;
    const useHintMock = vi.fn().mockResolvedValue({
      hint_type: "F",
      eliminated_nonces: ["n0", "n3"],
    });
    const updatePrefsMock = vi.fn().mockImplementation(async (p) => p);

    setBridgeForTesting({
      dictionary: {
        Lookup: vi.fn().mockImplementation(async (w: string) => {
          lookupCalls++;
          if (lookupCalls === 1) {
            throw new Error("Simulated network timeout");
          }
          return createMockWordDetail(w);
        }),
        Autocomplete: vi.fn().mockResolvedValue([]),
        GetSearchHistory: vi.fn().mockResolvedValue(["callow"]),
        ClearSearchHistory: vi.fn().mockResolvedValue(undefined),
        GetExamples: vi.fn().mockResolvedValue([]),
      } as any,
      trainer: {
        StartSession: vi.fn().mockResolvedValue({
          ...createMockQuestionSlide(false),
          question: {
            ...createMockQuestionSlide(false).question!,
            type: "I",
            type_label: "Visual Match (4 Images)",
            is_image_grid: true,
            choices: [
              { index: 0, nonce: "n0", image_url: "https://cdn.vocabulary.com/q1.jpg" },
              { index: 1, nonce: "n1", image_url: "https://cdn.vocabulary.com/q2.jpg" },
              { index: 2, nonce: "n2", image_url: "https://cdn.vocabulary.com/q3.jpg" },
              { index: 3, nonce: "n3", image_url: "https://cdn.vocabulary.com/q4.jpg" },
            ],
          },
        }),
        UseHint: useHintMock,
        GetSessionState: vi.fn().mockResolvedValue({
          ...createMockQuestionSlide(false),
          question: {
            ...createMockQuestionSlide(false).question!,
            type: "I",
            type_label: "Visual Match (4 Images)",
            is_image_grid: true,
            choices: [
              { index: 0, nonce: "n0", image_url: "https://cdn.vocabulary.com/q1.jpg", eliminated: true },
              { index: 1, nonce: "n1", image_url: "https://cdn.vocabulary.com/q2.jpg", eliminated: false },
              { index: 2, nonce: "n2", image_url: "https://cdn.vocabulary.com/q3.jpg", eliminated: false },
              { index: 3, nonce: "n3", image_url: "https://cdn.vocabulary.com/q4.jpg", eliminated: true },
            ],
          },
        }),
      } as any,
      settings: {
        GetPreferences: vi.fn().mockResolvedValue({
          theme: "light",
          pronunciation_accent: "US",
          default_view: "home",
          session_store_mode: SessionStoreMode.SessionStoreKeyring,
          search_history: ["callow"],
          app_version: "1.0.0",
        }),
        UpdatePreferences: updatePrefsMock,
      } as any,
    });

    render(<App />);

    // 1. Dictionary error state & Retry recovery
    await user.click(await screen.findByRole("button", { name: /^Dictionary$/i }));
    expect(await screen.findByText(/Simulated network timeout/i)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /^Retry$/i }));
    expect(await screen.findByText(/Someone who is callow is young and inexperienced/i)).toBeInTheDocument();

    // 2. Practice 4-image grid & 50/50 lifeline hint
    await user.click(screen.getByRole("button", { name: /^Practice$/i }));
    expect(await screen.findByText(/Visual Match \(4 Images\)/i)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "50/50" }));
    expect(useHintMock).toHaveBeenCalledWith("F");
    expect(await screen.findByText(/Two incorrect choices have been eliminated/i)).toBeInTheDocument();

    // 3. Progress Leaderboards view
    await user.click(screen.getByRole("button", { name: /^Progress$/i }));
    expect(await screen.findByText("TopScholar")).toBeInTheDocument();

    // 4. Settings Dark theme update
    await user.click(screen.getByRole("button", { name: /^Settings$/i }));
    await user.click(await screen.findByRole("button", { name: /Dark Slate/i }));
    expect(updatePrefsMock).toHaveBeenCalled();
  });
  it("keeps sign-in focus stable while editing and restores the opener on Escape", async () => {
    const user = userEvent.setup();
    render(<App />);
    const opener = (await screen.findAllByRole("button", { name: /^Sign In$/i }))[0];
    await user.click(opener);
    const dialog = screen.getByRole("dialog");
    const password = within(dialog).getByLabelText(/^Password$/i);
    await user.type(password, "a secret with spaces");
    expect(password).toHaveFocus();
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(opener).toHaveFocus();
    await user.click(opener);
    expect(within(screen.getByRole("dialog")).getByLabelText(/^Password$/i)).toHaveValue("");
  });

  it("shows the pending choice without claiming success before the SDK responds", async () => {
    const user = userEvent.setup();
    let resolveAnswer!: (value: any) => void;
    const answer = new Promise<any>((resolve) => { resolveAnswer = resolve; });
    vi.mocked(getBridge().trainer.SubmitChoice).mockReturnValueOnce(answer as any);
    render(<App />);
    await user.click(await screen.findByRole("button", { name: /^Practice$/i }));
    const choice = await screen.findByRole("button", { name: /young and inexperienced/i });
    await user.click(choice);
    expect(choice).toHaveAttribute("aria-pressed", "true");
    expect(choice).toBeDisabled();
    expect(screen.getByTestId("pacing-indicator")).toHaveTextContent("Submitting answer");
    expect(screen.queryByRole("region", { name: "Answer feedback" })).not.toBeInTheDocument();
    await act(async () => resolveAnswer({
      ...createMockQuestionSlide(false),
      session_state: TrainerSessionState.SessionStateQuestionCompleted,
      last_feedback: { correct: true, attempt: 1, points_earned: 100, word: "callow" },
    }));
    expect(await screen.findByText(/Correct on first try!/i)).toBeInTheDocument();
  });

  it("uses arrow keys to switch tool tabs and leaves the selected tab in the tab order", async () => {
    const user = userEvent.setup();
    render(<App />);
    await user.click(await screen.findByRole("button", { name: /^Discover$/i }));
    const collections = screen.getByRole("tab", { name: /Curated Collections/i });
    await user.click(collections);
    await user.keyboard("{End}");
    const finder = screen.getByRole("tab", { name: "WordFinder" });
    expect(finder).toHaveFocus();
    expect(finder).toHaveAttribute("aria-selected", "true");
    expect(collections).toHaveAttribute("tabindex", "-1");
    expect(screen.getByLabelText("Match Rule")).toBeInTheDocument();
  });

  it("flips a focused flashcard once and does not intercept typing in search", async () => {
    const user = userEvent.setup();
    render(<App />);
    await user.click(await screen.findByRole("button", { name: /^My Lists$/i }));
    await user.click(await screen.findByRole("button", { name: /Inspect Words/i }));
    await user.click(await screen.findByRole("button", { name: /Flashcards/i }));
    const card = screen.getByRole("button", { name: "Flip flashcard for callow" });
    card.focus();
    await user.keyboard(" ");
    expect(card).toHaveAttribute("aria-pressed", "true");
    await user.keyboard(" ");
    expect(card).toHaveAttribute("aria-pressed", "false");
    await user.type(screen.getByLabelText("Quick word lookup"), "a word");
    expect(card).toHaveAttribute("aria-pressed", "false");
  });

  it("filters personal lists and offers recovery when a filter has no matches", async () => {
    const user = userEvent.setup();
    vi.mocked(getBridge().auth.RestoreSession).mockResolvedValue(createMockAccount(true) as any);
    setBridgeForTesting({ lists: { ...getBridge().lists, GetUserLists: vi.fn().mockResolvedValue([
      { wordlistid: 1, name: "Biology words", wordcount: 12 },
      { wordlistid: 2, name: "Literature words", wordcount: 8 },
    ]) } as any });
    render(<App />);
    await screen.findByText("Ayush");
    await user.click(screen.getByRole("button", { name: /^My Lists$/i }));
    const filter = await screen.findByLabelText("Filter personal lists");
    await screen.findByText("Biology words");
    await user.type(filter, "biology");
    expect(screen.queryByText("Literature words")).not.toBeInTheDocument();
    await user.type(filter, "unmatched");
    expect(screen.getByText("No lists match your search")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear filter" }));
    expect(screen.getByText("Literature words")).toBeInTheDocument();
  });

  it("keeps the latest example-source result when requests finish out of order", async () => {
    const user = userEvent.setup();
    let resolveLiterature!: (value: any) => void;
    let resolveNews!: (value: any) => void;
    const literature = new Promise<any>((resolve) => { resolveLiterature = resolve; });
    const news = new Promise<any>((resolve) => { resolveNews = resolve; });
    vi.mocked(getBridge().dictionary.GetExamples).mockReturnValueOnce(literature as any).mockReturnValueOnce(news as any);
    render(<App />);
    await user.click(await screen.findByRole("button", { name: /^Dictionary$/i }));
    await screen.findByRole("heading", { name: "callow" });
    await user.click(screen.getByRole("button", { name: "Literature" }));
    await user.click(screen.getByRole("button", { name: "News" }));
    await act(async () => resolveNews([{ sentence: "The latest news example.", source_name: "News" }]));
    expect(await screen.findByText("The latest news example.")).toBeInTheDocument();
    await act(async () => resolveLiterature([{ sentence: "The outdated literature example.", source_name: "Book" }]));
    expect(screen.queryByText("The outdated literature example.")).not.toBeInTheDocument();
    expect(screen.getByText("The latest news example.")).toBeInTheDocument();
  });

});
