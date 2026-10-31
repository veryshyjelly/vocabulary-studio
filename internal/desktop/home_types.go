package desktop

import (
	vocab "github.com/veryshyjelly/vocabulary-studio/sdk"
)

// HomeDashboardDTO aggregates real data for the Home screen.
type HomeDashboardDTO struct {
	Account              AccountStatusDTO            `json:"account"`
	WordOfTheDay         *vocab.WordOfTheDayEntry    `json:"word_of_the_day,omitempty"`
	Effort               *vocab.EffortStats          `json:"effort,omitempty"`
	InProgressActivities []vocab.ActivityItem        `json:"in_progress_activities,omitempty"`
	LearningWords        []vocab.AccountProgressWord `json:"learning_words,omitempty"`
	TroubleWords         []vocab.AccountProgressWord `json:"trouble_words,omitempty"`
	RecentLists          []vocab.ProfileListSummary  `json:"recent_lists,omitempty"`
	SearchHistory        []string                    `json:"search_history"`
}
