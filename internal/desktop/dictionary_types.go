package desktop

import (
	vocab "github.com/veryshyjelly/vocabulary-studio/sdk"
)

// CorpusSentenceDTO wraps a CorpusSentence with its resolved canonical SourceURL.
type CorpusSentenceDTO struct {
	Sentence    string `json:"sentence"`
	Offsets     []int  `json:"offsets,omitempty"`
	SourceTitle string `json:"source_title,omitempty"`
	SourceName  string `json:"source_name,omitempty"`
	Author      string `json:"author,omitempty"`
	SourceURL   string `json:"source_url,omitempty"`
	CorpusID    string `json:"corpus_id,omitempty"`
}

// WordDetailDTO combines dictionary lookup, user progress (when available), and corpus examples.
type WordDetailDTO struct {
	Entry    *vocab.DictionaryEntry    `json:"entry"`
	Progress *vocab.WordProgressDetail `json:"progress,omitempty"`
	Examples []CorpusSentenceDTO       `json:"examples,omitempty"`
}
