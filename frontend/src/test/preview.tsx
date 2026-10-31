/** Development-only visual fixtures. This entry is never imported by the app. */
import { createRoot } from "react-dom/client";
import { App } from "../App";
import { setBridgeForTesting, SessionStoreMode } from "../lib/bridge";
import "../index.css";

const params = new URLSearchParams(location.search);
const scenario = params.get("scenario") || "choices";
const word = scenario === "long" ? "pneumonoultramicroscopicsilicovolcanoconiosis" : "callow";
const account = {
  authenticated: scenario !== "auth", valid_user: true, uid: "fixture-learner", nickname: "Sample learner",
  points: 1250, level: { name: "Wordsmith", id: "L04", milestone: 1, progress: 0.4 },
  full_vocab_trainer: true, session_store_mode: SessionStoreMode.SessionStoreMemory,
};
let preferences = {
  theme: params.get("theme") || "light", pronunciation_accent: "US", default_view: params.get("screen") || "practice",
  session_store_mode: SessionStoreMode.SessionStoreMemory, search_history: [], app_version: "1.0.0",
};
const lists = scenario === "empty" ? [] : Array.from({ length: 24 }, (_, i) => ({
  wordlistid: i + 1, name: i === 0 ? "Words for a curious mind" : `Reading collection ${i + 1}`,
  wordcount: 20 + i, description: "Words gathered from books, conversations, and everyday discoveries.",
}));
const entry = {
  word, learnable: true, short_blurb: "Someone who is callow is young and inexperienced. With a little curiosity and practice, unfamiliar words become part of your everyday language.",
  long_blurb: "Callow comes from the Old English word calu, meaning bald or featherless. " .repeat(12),
  pronunciations: [{ region: "US", ipa: "/ˈkæloʊ/" }, { region: "UK", ipa: "/ˈkæləʊ/" }],
  senses: Array.from({ length: 8 }, (_, i) => ({ synset_id: i + 1, part_of_speech: "adjective", definition: i ? "Lacking experience in a particular activity or field of knowledge." : "young and inexperienced", examples: ["The callow apprentice was eager to learn."], related: [{ relation: "synonyms", words: ["fledgling", "unfledged"] }] })),
  word_family: [{ word, freq: 0.51 }, { word: "callowness", parent: "callow", freq: 0.01 }],
};
const choices = ["wealthy and influential", "young and inexperienced", "secretive and cautious", "exhausted and weary"];
let session: any;
function newSession() {
  return {
    active: true, session_state: "active_question", tries: 0, round: { number: 1, played_count: 2, streak: 2 },
    question: {
      turn: 1, type: scenario === "spelling" ? "T" : scenario === "images" ? "I" : "D",
      type_label: scenario === "spelling" ? "Spell the word" : scenario === "images" ? "Visual match" : "Meaning in context",
      target_word: scenario === "spelling" ? "" : word, is_spelling: scenario === "spelling", is_image_grid: scenario === "images",
      instructions_text: scenario === "spelling" ? "Spell the word you hear." : `In this sentence, ${word} means:`,
      sentence_text: scenario === "spelling" ? "Experience transformed the ____ apprentice into a confident craftsperson." : "Experience transformed the callow apprentice into a confident craftsperson.",
      sentence_spans: scenario === "spelling" ? [{ text: "Experience transformed the ____ apprentice into a confident craftsperson." }] : [{ text: "Experience transformed the " }, { text: "callow", bold: true }, { text: " apprentice into a confident craftsperson." }],
      source_text: "UI test fixture", point_value: 100, available_hints: ["F", "D"],
      choices: choices.map((text, index) => ({ index, nonce: `n${index}`, text,
        ...(scenario === "images" ? { image_url: `data:image/svg+xml,${encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" width="400" height="240" viewBox="0 0 400 240"><rect width="400" height="240" fill="${["#e7e4d8", "#d4e5d9", "#eadccf", "#d9e2e9"][index]}"/><circle cx="200" cy="110" r="48" fill="none" stroke="#4e6559" stroke-width="3"/><text x="200" y="195" fill="#334a40" text-anchor="middle" font-size="18">Test image ${index + 1}</text></svg>`)}` } : {}),
      })),
    },
  };
}
session = newSession();
let quiz: any;
let bee: any;
const delay = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));
const getExamples = async () => scenario === "empty" ? [] : [{ sentence: "The callow apprentice was eager to learn.", source_title: "UI test fixture", offsets: [4, 10] }];
setBridgeForTesting({
  settings: { GetPreferences: async () => preferences, UpdatePreferences: async (p: any) => (preferences = p) } as any,
  auth: { RestoreSession: async () => account, GetStatus: async () => account, Login: async () => { throw new Error("Sign-in is disabled in the visual test fixture."); }, Logout: async () => ({ ...account, authenticated: false }) } as any,
  trainer: {
    StartSession: async () => { if (scenario === "loading") await delay(60000); if (scenario === "error") throw new Error("Unable to reach Vocabulary.com. Check your connection and try again."); return (session = newSession()); },
    GetSessionState: async () => session,
    StartQuiz: async () => (quiz = { active: true, is_complete: false, question_number: 1, question_total: 3, activity_id: "fixture-quiz", question: newSession().question }),
    SubmitQuizAnswer: async () => { await delay(500); quiz = { ...quiz, question_number: quiz.question_number + 1, is_complete: quiz.question_number === 3 }; return quiz; },
    GetQuizResults: async () => [],
    StartBee: async () => (bee = { active: true, completed_word: false, guesses: 0, formatted_example: "The ____ apprentice was eager to learn.", definitions: [{ pos: "adjective", text: "Young and inexperienced." }], can_surrender: true }),
    GuessBee: async (guess: string) => { const correct = guess.toLowerCase() === "callow"; return (bee = { ...bee, guesses: bee.guesses + 1, correct, completed_word: correct, revealed_word: correct ? "callow" : "" }); },
    SurrenderBee: async () => (bee = { ...bee, completed_word: true, revealed_word: "callow" }),
    SubmitChoice: async (index: number) => {
      await delay(900);
      session = { ...session, tries: session.tries + 1, session_state: index === 1 ? "question_completed" : "active_question", last_feedback: { correct: index === 1, attempt: session.tries + 1, points_earned: index === 1 ? 100 : 0, word, evaluated_locally: session.tries > 0, sense_definition: "young and inexperienced", short_blurb_text: entry.short_blurb } };
      return session;
    },
    SubmitSpelling: async (guess: string) => { await delay(500); const correct = guess.toLowerCase() === "callow"; return (session = { ...session, session_state: correct ? "question_completed" : "active_question", last_feedback: { correct, attempt: 1, points_earned: correct ? 100 : 0, word: "callow" } }); },
    NextQuestion: async () => (session = newSession()),
    UseHint: async () => ({ hint_type: "D", definition_text: "Young and inexperienced." }),
  } as any,
  dictionary: {
    Lookup: async () => { if (scenario === "error") throw new Error("Dictionary lookup timed out. Please try again."); return { entry, examples: await getExamples() }; },
    Autocomplete: async () => [{ word, definition: "young and inexperienced" }],
    GetSearchHistory: async () => ["callow", "serendipity", "perspicacious"], ClearSearchHistory: async () => {},
    GetExamples: getExamples, GetRandomWord: async () => ({ entry }),
    ResolveAudioURL: async () => "",
  } as any,
  lists: {
    GetUserLists: async () => lists, SearchLists: async () => ({ results: lists, totalResults: lists.length }),
    GetListDetail: async () => ({ ...lists[0], words: [{ word: "callow", definition: "young and inexperienced" }] }),
    GetFlashcards: async () => ({ wordlist: { words: [{ word: "callow", sense: { definition: "young and inexperienced" } }] } }),
  } as any,
  progress: {
    GetHomeDashboard: async () => ({ account, effort: { words_mastered: 8, questions_answered: 42, points: 500, time_minutes: 12 }, word_of_the_day: { word: "serendipity", date: "UI test fixture", short_usage: "The pleasure of finding something good when you were not looking for it." }, recent_lists: lists.slice(0, 3) }),
    ListLearningWords: async () => scenario === "empty" ? [] : [{ word, definition: "young and inexperienced", progress: 0.65 }],
    StartLearningWord: async () => {}, GetWordProgress: async () => null, SetWordPriority: async () => {},
    GetListProgress: async () => ({ progress: 0.4 }), GetAchievements: async () => [], GetActivities: async () => ({ activities: [] }),
    GetLeaderboard: async () => ({ rows: [] }),
  } as any,
  discovery: { GetRootCategories: async () => [], SearchUnits: async () => ({ units: [] }), WordFinder: async () => ({ words: [], totalResults: 0 }) } as any,
});
createRoot(document.getElementById("root")!).render(<App />);
