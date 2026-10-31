// Package vocab preserves the original Vocabulary.com SDK import path.
//
// Deprecated: import github.com/veryshyjelly/vocabulary_go/sdk for new code.
package vocab

import (
	"log/slog"
	"net/http"

	sdk "github.com/veryshyjelly/vocabulary_go/sdk"
)

// Legacy type aliases retain the SDK's concrete types and methods.
type AuthService = sdk.AuthService
type LoginData = sdk.LoginData
type UserLevel = sdk.UserLevel
type AuthDetails = sdk.AuthDetails
type SchoolInfo = sdk.SchoolInfo
type UserPerms = sdk.UserPerms
type UserClasses = sdk.UserClasses
type UserPlan = sdk.UserPlan
type BowlSeason = sdk.BowlSeason
type BeeService = sdk.BeeService
type BeeScore = sdk.BeeScore
type BeeDef = sdk.BeeDef
type BeeStateResponse = sdk.BeeStateResponse
type CorpusDomain = sdk.CorpusDomain
type DictionaryEntry = sdk.DictionaryEntry
type AutocompleteSuggestion = sdk.AutocompleteSuggestion
type WordOfTheDayEntry = sdk.WordOfTheDayEntry
type DictionaryService = sdk.DictionaryService
type WordFinderOptions = sdk.WordFinderOptions
type WordFinderResponse = sdk.WordFinderResponse
type WordFinderWord = sdk.WordFinderWord
type WordFinderSense = sdk.WordFinderSense
type CorpusExamplesResponse = sdk.CorpusExamplesResponse
type CorpusSentence = sdk.CorpusSentence
type CorpusVolume = sdk.CorpusVolume
type WODArchiveResponse = sdk.WODArchiveResponse
type WODArchiveMonth = sdk.WODArchiveMonth
type WODArchiveItem = sdk.WODArchiveItem
type PreviewQuestionResponse = sdk.PreviewQuestionResponse
type APIError = sdk.APIError
type ListsService = sdk.ListsService
type VocabularyList = sdk.VocabularyList
type ListWord = sdk.ListWord
type ListWordExample = sdk.ListWordExample
type FlashcardListResponse = sdk.FlashcardListResponse
type FlashcardListWord = sdk.FlashcardListWord
type FlashcardSense = sdk.FlashcardSense
type FlashcardIPA = sdk.FlashcardIPA
type ProfileListSummary = sdk.ProfileListSummary
type SmartSearchResponse = sdk.SmartSearchResponse
type SmartSearchItem = sdk.SmartSearchItem
type Unit = sdk.Unit
type UnitItem = sdk.UnitItem
type UnitsResponse = sdk.UnitsResponse
type SearchUnitsOptions = sdk.SearchUnitsOptions
type LearnableSense = sdk.LearnableSense
type ValidatedWord = sdk.ValidatedWord
type VocabGrabberResponse = sdk.VocabGrabberResponse
type GrabbedWord = sdk.GrabbedWord
type CreateListWordInput = sdk.CreateListWordInput
type CreateListRequest = sdk.CreateListRequest
type Clock = sdk.Clock
type Option = sdk.Option
type PacingGuard = sdk.PacingGuard
type WordPriority = sdk.WordPriority
type WordFilter = sdk.WordFilter
type ProgressService = sdk.ProgressService
type WordProgressDetail = sdk.WordProgressDetail
type ListProgressStatus = sdk.ListProgressStatus
type AccountProgressWord = sdk.AccountProgressWord
type EffortStats = sdk.EffortStats
type ActivityItem = sdk.ActivityItem
type ActivitiesResponse = sdk.ActivitiesResponse
type ActivitiesQuery = sdk.ActivitiesQuery
type AchievementSummary = sdk.AchievementSummary
type ProfileSummary = sdk.ProfileSummary
type LeaderboardResponse = sdk.LeaderboardResponse
type LeaderboardRow = sdk.LeaderboardRow
type QuestionType = sdk.QuestionType
type ActivityType = sdk.ActivityType
type ClientVersion = sdk.ClientVersion
type HintType = sdk.HintType
type ChoiceOption = sdk.ChoiceOption
type ParsedQuestion = sdk.ParsedQuestion
type AnswerStats = sdk.AnswerStats
type RawQuestion = sdk.RawQuestion
type QuizService = sdk.QuizService
type StartQuizRequest = sdk.StartQuizRequest
type QuizStateResponse = sdk.QuizStateResponse
type SerializableCookie = sdk.SerializableCookie
type TokenInfo = sdk.TokenInfo
type SessionSnapshot = sdk.SessionSnapshot
type SessionStore = sdk.SessionStore
type MemorySessionStore = sdk.MemorySessionStore
type FileSessionStore = sdk.FileSessionStore
type KeyringSessionStore = sdk.KeyringSessionStore
type TrainerSessionState = sdk.TrainerSessionState
type TrainerService = sdk.TrainerService
type StartChallengeRequest = sdk.StartChallengeRequest
type ChallengePlayerData = sdk.ChallengePlayerData
type ChallengeListInfo = sdk.ChallengeListInfo
type ChallengeRound = sdk.ChallengeRound
type ChallengeGame = sdk.ChallengeGame
type ChallengeStateResponse = sdk.ChallengeStateResponse
type AnswerOutcome = sdk.AnswerOutcome
type WordFrequency = sdk.WordFrequency
type EarnedAchievement = sdk.EarnedAchievement
type AnswerWordProgress = sdk.AnswerWordProgress
type WordBlurb = sdk.WordBlurb
type AnswerWordSense = sdk.AnswerWordSense
type AnswerPronunciation = sdk.AnswerPronunciation
type SaveAnswerResponse = sdk.SaveAnswerResponse
type HintResponse = sdk.HintResponse
type TrainerSession = sdk.TrainerSession
type Client = sdk.Client

const CorpusDomainAll = sdk.CorpusDomainAll
const CorpusDomainLiterature = sdk.CorpusDomainLiterature
const CorpusDomainArts = sdk.CorpusDomainArts
const CorpusDomainNews = sdk.CorpusDomainNews
const CorpusDomainBusiness = sdk.CorpusDomainBusiness
const CorpusDomainSports = sdk.CorpusDomainSports
const CorpusDomainScienceMed = sdk.CorpusDomainScienceMed
const CorpusDomainTechnology = sdk.CorpusDomainTechnology
const UnitCategoryLiterature = sdk.UnitCategoryLiterature
const UnitCategoryNonFiction = sdk.UnitCategoryNonFiction
const UnitCategoryTextbooks = sdk.UnitCategoryTextbooks
const UnitCategoryTestPrep = sdk.UnitCategoryTestPrep
const UnitCategoryCurrentEvents = sdk.UnitCategoryCurrentEvents
const UnitCategoryRootsAndAffixes = sdk.UnitCategoryRootsAndAffixes
const UnitCategoryJustForFun = sdk.UnitCategoryJustForFun
const UnitCategorySpeeches = sdk.UnitCategorySpeeches
const UnitCategoryHistoricalDocuments = sdk.UnitCategoryHistoricalDocuments
const DefaultWebBaseURL = sdk.DefaultWebBaseURL
const DefaultAPIBaseURL = sdk.DefaultAPIBaseURL
const DefaultCorpusBaseURL = sdk.DefaultCorpusBaseURL
const DefaultAudioBaseURL = sdk.DefaultAudioBaseURL
const DefaultVideoBaseURL = sdk.DefaultVideoBaseURL
const DefaultUserAgent = sdk.DefaultUserAgent
const DefaultTimezone = sdk.DefaultTimezone
const DefaultMaxResponseBytes = sdk.DefaultMaxResponseBytes
const MinChallengeFirstTryMS = sdk.MinChallengeFirstTryMS
const MinChallengeRetryMS = sdk.MinChallengeRetryMS
const MinQuizSubmitMS = sdk.MinQuizSubmitMS
const MinBeeSubmitMS = sdk.MinBeeSubmitMS
const TrainerRollingWindow = sdk.TrainerRollingWindow
const TrainerMaxAnswersInWindow = sdk.TrainerMaxAnswersInWindow
const PriorityLow = sdk.PriorityLow
const PriorityNormal = sdk.PriorityNormal
const PriorityHigh = sdk.PriorityHigh
const WordFilterLearning = sdk.WordFilterLearning
const WordFilterTrouble = sdk.WordFilterTrouble
const WordFilterMastered = sdk.WordFilterMastered
const QuestionTypeAntonym = sdk.QuestionTypeAntonym
const QuestionTypeDefinition = sdk.QuestionTypeDefinition
const QuestionTypeCustom = sdk.QuestionTypeCustom
const QuestionTypeFillIn = sdk.QuestionTypeFillIn
const QuestionTypeImage4 = sdk.QuestionTypeImage4
const QuestionTypeSingleImage = sdk.QuestionTypeSingleImage
const QuestionTypeSentenceDefinition = sdk.QuestionTypeSentenceDefinition
const QuestionTypeSentenceSynonym = sdk.QuestionTypeSentenceSynonym
const QuestionTypeSpelling = sdk.QuestionTypeSpelling
const QuestionTypeSynonym = sdk.QuestionTypeSynonym
const QuestionTypeHyponymTypeOf = sdk.QuestionTypeHyponymTypeOf
const QuestionTypeMeronymPartOf = sdk.QuestionTypeMeronymPartOf
const QuestionTypeMatchWordToDefinition = sdk.QuestionTypeMatchWordToDefinition
const QuestionTypeMultipleMeaning = sdk.QuestionTypeMultipleMeaning
const QuestionTypeRelatedWord = sdk.QuestionTypeRelatedWord
const QuestionTypeWordUsage = sdk.QuestionTypeWordUsage
const ActivityVocabTrainer = sdk.ActivityVocabTrainer
const ActivityListPractice = sdk.ActivityListPractice
const ClientVersionMobile = sdk.ClientVersionMobile
const ClientVersionClassicWeb = sdk.ClientVersionClassicWeb
const ClientVersionVocabTrainer = sdk.ClientVersionVocabTrainer
const HintFiftyFifty = sdk.HintFiftyFifty
const HintDefinition = sdk.HintDefinition
const HintExample = sdk.HintExample
const SessionStateIdle = sdk.SessionStateIdle
const SessionStateActiveQuestion = sdk.SessionStateActiveQuestion
const SessionStateQuestionCompleted = sdk.SessionStateQuestionCompleted
const SessionStateTrialOver = sdk.SessionStateTrialOver

var ErrInvalidCredentials = sdk.ErrInvalidCredentials
var ErrMaxLoginAttempts = sdk.ErrMaxLoginAttempts
var ErrSessionExpired = sdk.ErrSessionExpired
var ErrTrialOver = sdk.ErrTrialOver
var ErrRestartChallenge = sdk.ErrRestartChallenge
var ErrConcurrentPlay = sdk.ErrConcurrentPlay
var ErrCooldownActive = sdk.ErrCooldownActive
var ErrNotFound = sdk.ErrNotFound
var ErrRateLimited = sdk.ErrRateLimited
var ErrResponseTooLarge = sdk.ErrResponseTooLarge
var ErrUnexpectedResponse = sdk.ErrUnexpectedResponse
var ErrCorruptedSession = sdk.ErrCorruptedSession
var ErrInvalidStateTransition = sdk.ErrInvalidStateTransition

func ParseJWTClaims(accessToken string) (map[string]any, error) {
	return sdk.ParseJWTClaims(accessToken)
}
func LoadDotEnvFile(path string) error           { return sdk.LoadDotEnvFile(path) }
func WithHTTPClient(hc *http.Client) Option      { return sdk.WithHTTPClient(hc) }
func WithTransport(rt http.RoundTripper) Option  { return sdk.WithTransport(rt) }
func WithSessionStore(store SessionStore) Option { return sdk.WithSessionStore(store) }
func WithBaseURLs(webURL, apiURL, corpusURL, audioURL string) Option {
	return sdk.WithBaseURLs(webURL, apiURL, corpusURL, audioURL)
}
func WithUserAgent(ua string) Option              { return sdk.WithUserAgent(ua) }
func WithTimezone(tz string) Option               { return sdk.WithTimezone(tz) }
func WithRateLimit(rps float64, burst int) Option { return sdk.WithRateLimit(rps, burst) }
func WithPacingGuard(enabled bool) Option         { return sdk.WithPacingGuard(enabled) }
func WithClock(clock Clock) Option                { return sdk.WithClock(clock) }
func WithMaxResponseBytes(maxBytes int64) Option  { return sdk.WithMaxResponseBytes(maxBytes) }
func WithLogger(logger *slog.Logger) Option       { return sdk.WithLogger(logger) }
func NormalizeSpellingGuess(guess string) string  { return sdk.NormalizeSpellingGuess(guess) }
func ProficiencyScoreFromAbility(ability float64) int {
	return sdk.ProficiencyScoreFromAbility(ability)
}
func NewMemorySessionStore() *MemorySessionStore        { return sdk.NewMemorySessionStore() }
func NewFileSessionStore(path string) *FileSessionStore { return sdk.NewFileSessionStore(path) }
func NewKeyringSessionStore(service, account string) *KeyringSessionStore {
	return sdk.NewKeyringSessionStore(service, account)
}
func NewClient(opts ...Option) (*Client, error) { return sdk.NewClient(opts...) }
func ParseLinkHeader(linkHeader, apiBaseURL string) map[string]string {
	return sdk.ParseLinkHeader(linkHeader, apiBaseURL)
}
