# Practice

## Purpose

Support adaptive trainer and list practice, Spelling Bee, and quizzes while
preserving the upstream distinction between saved answers and local retries.

Implementation: `sdk/trainer_session.go`, `sdk/question.go`, `sdk/bee.go`, `sdk/quiz.go`,
`internal/desktop/trainer_service.go`, `internal/desktop/bee_service.go`,
`internal/desktop/quiz_service.go`, `frontend/src/features/practice/PracticeView.tsx`.
Existing coverage: `sdk/trainer_test.go`, `sdk/integration_test.go`,
`internal/desktop/desktop_test.go`, `frontend/src/app/App.test.tsx`.

## Requirements

### Requirement: Explicit trainer lifecycle
Trainer sessions SHALL expose idle, active-question, question-completed, and trial-over states and SHALL reject answers incompatible with the current state or question type.

#### Scenario: Answer a completed question
- WHEN a caller submits another answer after the question is completed
- THEN the SDK returns an invalid-state-transition error and requires the next question to be loaded

#### Scenario: Trial has ended
- WHEN a caller requests the next question from a trial-over session
- THEN the SDK returns `ErrTrialOver`

### Requirement: Saved first attempts and local retries
Trainer sessions SHALL submit the first answer attempt upstream and evaluate subsequent attempts locally from the returned answer data, marking local outcomes and awarding no additional points for them.

#### Scenario: Retry an incorrect spelling answer
- GIVEN a first spelling attempt returned accepted answers from the server
- WHEN a later normalized spelling guess matches an accepted answer
- THEN the session completes the question with `EvaluatedLocally` set and zero additional points

#### Scenario: Answer is still pending
- WHEN a user selects an answer and its SDK submission has not completed
- THEN the practice view marks the selected answer as pending without claiming success

### Requirement: Active-question hints
Trainer sessions SHALL allow hints only on an active question before its first answer attempt and SHALL reflect eliminated choices returned by hints.

#### Scenario: Use a fifty-fifty hint
- WHEN the user requests the fifty-fifty hint before answering an active multiple-choice question
- THEN the returned eliminated choices are unavailable for selection in the practice view

#### Scenario: Hint requested after answering
- WHEN a caller requests a hint after the first answer attempt
- THEN the SDK returns an invalid-state-transition error

### Requirement: Spelling Bee sessions
The desktop SHALL support starting or resuming a list's Spelling Bee, submitting guesses, surrendering eligible words, advancing completed words, and resetting progress.

#### Scenario: Surrender is not yet available
- WHEN the Bee session reports that surrender is unavailable
- THEN the practice view does not offer its surrender action

#### Scenario: Advance after finishing a word
- WHEN the current Bee word has been completed or surrendered and the caller advances
- THEN the desktop provides the next available word state

### Requirement: Quiz sessions and results
The desktop SHALL support starting or resuming a quiz, submitting answers through SDK pacing, and retrieving a completed quiz's per-question results.

#### Scenario: Complete a quiz
- WHEN upstream quiz state reports completion after an answer submission
- THEN the application exposes the completed state and can retrieve the activity's question results
