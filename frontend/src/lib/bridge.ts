import {
  AuthService,
  DictionaryService,
  DiscoveryService,
  ListsService,
  ProgressService,
  SettingsService,
  TrainerService,
} from "../bindings/github.com/veryshyjelly/vocabulary_go/internal/desktop/index.js";

export type {
  AccountStatusDTO,
  AnswerFeedbackDTO,
  AppPreferences,
  BeeDefDTO,
  BeeSessionDTO,
  CategoryCardDTO,
  ChoiceDTO,
  CorpusSentenceDTO,
  HintResultDTO,
  HomeDashboardDTO,
  QuestionSlideDTO,
  QuizSessionDTO,
  RichSpan,
  TrainerSessionStateDTO,
  WordDetailDTO,
} from "../bindings/github.com/veryshyjelly/vocabulary_go/internal/desktop/models.js";

export { SessionStoreMode } from "../bindings/github.com/veryshyjelly/vocabulary_go/internal/desktop/models.js";

export type {
  AccountProgressWord,
  AchievementSummary,
  ActivitiesResponse,
  ActivityItem,
  AutocompleteSuggestion,
  CreateListWordInput,
  DictionaryEntry,
  EffortStats,
  FlashcardListResponse,
  GrabbedWord,
  LeaderboardResponse,
  LearnableSense,
  ListProgressStatus,
  ProfileListSummary,
  SmartSearchResponse,
  Unit,
  UnitsResponse,
  ValidatedWord,
  VocabGrabberResponse,
  VocabularyList,
  WODArchiveResponse,
  WordFinderResponse,
  WordOfTheDayEntry,
  WordProgressDetail,
} from "../bindings/github.com/veryshyjelly/vocabulary_go/models.js";

export type { QuizQuestionResult } from "../bindings/github.com/veryshyjelly/vocabulary_go/internal/htmlparse/models.js";

export interface DesktopBridge {
  auth: typeof AuthService;
  trainer: typeof TrainerService;
  dictionary: typeof DictionaryService;
  lists: typeof ListsService;
  discovery: typeof DiscoveryService;
  progress: typeof ProgressService;
  settings: typeof SettingsService;
}

const defaultBridge: DesktopBridge = {
  auth: AuthService,
  trainer: TrainerService,
  dictionary: DictionaryService,
  lists: ListsService,
  discovery: DiscoveryService,
  progress: ProgressService,
  settings: SettingsService,
};

let activeBridge: DesktopBridge = defaultBridge;

export function getBridge(): DesktopBridge {
  return activeBridge;
}

export function setBridgeForTesting(custom: Partial<DesktopBridge>) {
  activeBridge = {
    ...activeBridge,
    ...custom,
  };
}

export function resetBridge() {
  activeBridge = defaultBridge;
}

export function formatError(err: unknown): string {
  if (!err) return "An unexpected error occurred.";
  if (typeof err === "string") return err;
  if (err instanceof Error) return err.message;
  if (typeof err === "object" && "message" in err && typeof (err as { message: unknown }).message === "string") {
    return (err as { message: string }).message;
  }
  return String(err);
}
