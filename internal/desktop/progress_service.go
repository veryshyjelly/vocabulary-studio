package desktop

import (
	"context"
	"strings"
	"sync"
	"time"

	vocab "github.com/veryshyjelly/vocabulary_go/sdk"
)

// ProgressService exposes the Home dashboard aggregator, learning/trouble/mastered word lists,
// priority controls, effort metrics, activities, achievements, and leaderboards.
type ProgressService struct {
	hub *Hub
}

// GetHomeDashboard aggregates real data for the Home screen, gracefully degrading for unauthenticated users.
func (p *ProgressService) GetHomeDashboard(ctx context.Context) (*HomeDashboardDTO, error) {
	ctx, cancel := p.hub.BeginOperation(ctx, "progress:dashboard", 25*time.Second)
	defer cancel()

	client := p.hub.Client()
	out := &HomeDashboardDTO{
		SearchHistory: p.hub.Dictionary.GetSearchHistory(),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	// 1. Check auth status
	me, authErr := client.Auth.Me(ctx)
	if authErr == nil && me != nil {
		out.Account = *p.hub.Auth.mapLoginData(me)
	} else {
		out.Account = *p.hub.Auth.unauthenticatedStatus()
	}

	// 2. Fetch Word of the Day (works unauthenticated)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if wod, err := client.Dictionary.WordOfTheDay(ctx, ""); err == nil {
			mu.Lock()
			out.WordOfTheDay = wod
			mu.Unlock()
		}
	}()

	// 3. If authenticated, fetch real learner effort, activities, learning/trouble words, and recent lists in parallel
	if out.Account.Authenticated {
		now := time.Now()
		startDate := now.AddDate(0, 0, -7).Format("2006-01-02")
		endDate := now.Format("2006-01-02")

		wg.Add(5)
		go func() {
			defer wg.Done()
			if eff, err := client.Progress.GetEffort(ctx, startDate, endDate); err == nil {
				mu.Lock()
				out.Effort = eff
				mu.Unlock()
			}
		}()
		go func() {
			defer wg.Done()
			complete := false
			if acts, err := client.Progress.GetActivities(ctx, vocab.ActivitiesQuery{Complete: &complete, Limit: 5, Include: "wordlists"}); err == nil && acts != nil {
				mu.Lock()
				out.InProgressActivities = acts.Activities
				mu.Unlock()
			}
		}()
		go func() {
			defer wg.Done()
			if words, err := client.Progress.ListLearningWords(ctx, vocab.WordFilterLearning, 0, "", ""); err == nil {
				if len(words) > 8 {
					words = words[:8]
				}
				mu.Lock()
				out.LearningWords = words
				mu.Unlock()
			}
		}()
		go func() {
			defer wg.Done()
			if trouble, err := client.Progress.ListLearningWords(ctx, vocab.WordFilterTrouble, 0, "", ""); err == nil {
				if len(trouble) > 8 {
					trouble = trouble[:8]
				}
				mu.Lock()
				out.TroubleWords = trouble
				mu.Unlock()
			}
		}()
		go func() {
			defer wg.Done()
			if lists, err := client.Lists.ListUserLists(ctx, "", "activitydate", 6); err == nil {
				mu.Lock()
				out.RecentLists = lists
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return out, nil
}

// GetWordProgress fetches sense and mastery progress for a single word.
func (p *ProgressService) GetWordProgress(ctx context.Context, word string) (*vocab.WordProgressDetail, error) {
	return p.hub.Client().Progress.GetWordProgress(ctx, word, true)
}

// SetWordPriority updates a word's learning priority (-1 Low, 0 Normal, 1 High).
func (p *ProgressService) SetWordPriority(ctx context.Context, word string, priority int) (*vocab.WordProgressDetail, error) {
	return p.hub.Client().Progress.SetWordPriority(ctx, word, vocab.WordPriority(priority))
}

// StartLearningWord adds `word` to the user's active learning program.
func (p *ProgressService) StartLearningWord(ctx context.Context, word string) error {
	return p.hub.Client().Progress.StartLearningWord(ctx, word)
}

// StopLearningWord removes `word` from the user's active learning program.
func (p *ProgressService) StopLearningWord(ctx context.Context, word string) error {
	return p.hub.Client().Progress.StopLearningWord(ctx, word)
}

// ListLearningWords fetches paginated words from `/account/progress/words.json`
// (`filter` = `"learning"`, `"trouble"`, or `"mastered"`).
func (p *ProgressService) ListLearningWords(ctx context.Context, filter string, skip int, sortBy, sortDir string) ([]vocab.AccountProgressWord, error) {
	ctx, cancel := p.hub.BeginOperation(ctx, "progress:words", 20*time.Second)
	defer cancel()
	return p.hub.Client().Progress.ListLearningWords(ctx, vocab.WordFilter(strings.TrimSpace(filter)), skip, sortBy, sortDir)
}

// GetListProgress fetches the user's mastery status on a vocabulary list.
func (p *ProgressService) GetListProgress(ctx context.Context, listID int64) (*vocab.ListProgressStatus, error) {
	return p.hub.Client().Progress.GetListProgress(ctx, listID)
}

// GetEffort fetches learner effort stats between `startDate` and `endDate` (`YYYY-MM-DD`).
func (p *ProgressService) GetEffort(ctx context.Context, startDate, endDate string) (*vocab.EffortStats, error) {
	return p.hub.Client().Progress.GetEffort(ctx, startDate, endDate)
}

// GetActivities fetches the user's activities (`complete` = `"true"`, `"false"`, or `""` for all).
func (p *ProgressService) GetActivities(ctx context.Context, completeFilter string, limit int, nextPageURI string) (*vocab.ActivitiesResponse, error) {
	ctx, cancel := p.hub.BeginOperation(ctx, "progress:activities", 20*time.Second)
	defer cancel()
	if strings.TrimSpace(nextPageURI) != "" {
		return p.hub.Client().Progress.GetActivitiesByURI(ctx, nextPageURI)
	}
	var compPtr *bool
	switch strings.ToLower(strings.TrimSpace(completeFilter)) {
	case "true":
		t := true
		compPtr = &t
	case "false":
		f := false
		compPtr = &f
	}
	if limit <= 0 {
		limit = 20
	}
	return p.hub.Client().Progress.GetActivities(ctx, vocab.ActivitiesQuery{
		Complete: compPtr,
		Limit:    limit,
		Include:  "wordlists",
	})
}

// GetAchievements fetches the user's earned achievements and milestones.
func (p *ProgressService) GetAchievements(ctx context.Context) ([]vocab.AchievementSummary, error) {
	ctx, cancel := p.hub.BeginOperation(ctx, "progress:achievements", 20*time.Second)
	defer cancel()
	return p.hub.Client().Progress.GetAchievements(ctx, 500)
}

// GetLeaderboard fetches an individual leaderboard (`metric` = `"points"` or `"mastered"`,
// `period` = `""` (all-time), `"daily"`, `"weekly"`, `"monthly"`, or `"season"`, `yyyymmdd` = `"20261031"`).
func (p *ProgressService) GetLeaderboard(ctx context.Context, metric, period, yyyymmdd string, top int) (*vocab.LeaderboardResponse, error) {
	ctx, cancel := p.hub.BeginOperation(ctx, "progress:leaderboard", 20*time.Second)
	defer cancel()
	return p.hub.Client().Progress.GetIndividualLeaderboard(ctx, metric, period, yyyymmdd, top, "")
}
