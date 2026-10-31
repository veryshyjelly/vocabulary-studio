// Package vocab provides a pure-Go (CGO_ENABLED=0) client SDK for Vocabulary.com,
// designed for desktop, mobile, and CLI applications.
//
// It implements Vocabulary.com's dual-tier authentication (web session cookies +
// singleflight Bearer JWT refresh), VocabTrainer (v=4) and List Practice (v=3)
// gameplay with built-in anti-cheat pacing guards, Vocabulary Lists and Curated
// Collections (Units), Dictionary word lookups, Corpus example sentences ("Word
// in the Wild"), audio/video pronunciations, Word of the Day, User Progress,
// Spelling Bee, and Quizzes.
package vocab
