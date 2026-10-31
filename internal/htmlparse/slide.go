package htmlparse

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var bgImageURLRegex = regexp.MustCompile(`url\(['"]?([^'")]+)['"]?\)`)

// ChoiceOption represents one of the multiple-choice answers in a challenge/quiz slide.
type ChoiceOption struct {
	Index     int    `json:"index"`
	Nonce     string `json:"nonce"`
	AccessKey string `json:"access_key,omitempty"`
	Text      string `json:"text,omitempty"`
	HTML      string `json:"html,omitempty"`
	ImageURL  string `json:"image_url,omitempty"`
}

// SlideData represents the parsed DOM structure of a base64 `question.code` slide
// or a raw `qs` quiz slide across all 16 Vocabulary.com question types:
// A, D, P, F, I, G, L, H, T, S, Y, R, C, M, E, U.
type SlideData struct {
	RawHTML          string         `json:"raw_html"`
	QuestionType     string         `json:"question_type,omitempty"`
	Template         string         `json:"template,omitempty"`
	Mode             string         `json:"mode,omitempty"`
	PointValue       int            `json:"point_value,omitempty"`
	AudioID          string         `json:"audio_id,omitempty"`
	TargetWord       string         `json:"target_word,omitempty"`
	InstructionsText string         `json:"instructions_text,omitempty"`
	InstructionsHTML string         `json:"instructions_html,omitempty"`
	SentenceText     string         `json:"sentence_text,omitempty"`
	SentenceHTML     string         `json:"sentence_html,omitempty"`
	BlankedWord      string         `json:"blanked_word,omitempty"`
	SourceText       string         `json:"source_text,omitempty"`
	PromptImageURL   string         `json:"prompt_image_url,omitempty"`
	IsSpelling       bool           `json:"is_spelling"`
	IsResultsSlide   bool           `json:"is_results_slide,omitempty"`
	Choices          []ChoiceOption `json:"choices,omitempty"`
}

// DecodeAndParseSlide base64-decodes `code` and parses the resulting HTML slide.
func DecodeAndParseSlide(base64Code, fallbackType string) (*SlideData, error) {
	trimmed := strings.TrimSpace(base64Code)
	if trimmed == "" {
		return nil, fmt.Errorf("htmlparse: empty question code")
	}
	raw, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		// Fallback to RawStdEncoding if unpadded.
		raw, err = base64.RawStdEncoding.DecodeString(trimmed)
		if err != nil {
			return nil, fmt.Errorf("htmlparse: base64 decode question code: %w", err)
		}
	}
	return ParseSlideHTML(string(raw), fallbackType)
}

// ParseSlideHTML parses a decoded HTML slide from Challenge/VocabTrainer (`question.code`) or Quiz (`qs`).
func ParseSlideHTML(rawHTML, fallbackType string) (*SlideData, error) {
	if strings.TrimSpace(rawHTML) == "" {
		return nil, fmt.Errorf("htmlparse: empty slide HTML")
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader([]byte(rawHTML)))
	if err != nil {
		return nil, fmt.Errorf("htmlparse: parse slide HTML: %w", err)
	}

	out := &SlideData{
		RawHTML:      rawHTML,
		QuestionType: strings.TrimSpace(fallbackType),
	}

	if doc.Find(".results, div.results").Length() > 0 {
		out.IsResultsSlide = true
	}

	// Audio ID can sit on root .challenge-slide, .question, or [data-audio]
	if audioID, ok := doc.Find("[data-audio]").First().Attr("data-audio"); ok {
		out.AudioID = strings.TrimSpace(audioID)
	}

	qSel := doc.Find(".question").First()
	if qSel.Length() == 0 {
		qSel = doc.Selection
	}

	if tpl, ok := qSel.Attr("data-template"); ok {
		out.Template = strings.TrimSpace(tpl)
	}
	if slideType, ok := qSel.Attr("data-type"); ok && strings.EqualFold(strings.TrimSpace(slideType), "spelling") {
		out.IsSpelling = true
		if out.QuestionType == "" {
			out.QuestionType = "T"
		}
	}

	// Extract class type{X} e.g. typeA, typeD, typeP, typeF, typeI, typeG, typeL, typeH, typeT, typeS, typeY, typeR, typeC, typeM, typeE, typeU
	if classAttr, ok := qSel.Attr("class"); ok {
		for _, tok := range strings.Fields(classAttr) {
			if strings.HasPrefix(tok, "type") && len(tok) == 5 {
				if out.QuestionType == "" {
					out.QuestionType = tok[4:]
				}
			}
		}
	}

	if out.QuestionType == "T" || doc.Find(".spelltheword, input.wordspelling").Length() > 0 {
		out.IsSpelling = true
		if out.QuestionType == "" {
			out.QuestionType = "T"
		}
	}

	// Mode & PointValue
	modeSel := doc.Find(".mode").First()
	if modeSel.Length() > 0 {
		switch {
		case modeSel.Find(".assessment").Length() > 0:
			out.Mode = "ASSESSMENT"
		case modeSel.Find(".progress").Length() > 0:
			out.Mode = "PROGRESS"
		case modeSel.Find(".review").Length() > 0:
			out.Mode = "REVIEW"
		case modeSel.Find(".mastery").Length() > 0:
			out.Mode = "BRUSH-UP"
		default:
			if txt := normalizeSpace(modeSel.Find(".interrogate").Text()); txt != "" {
				out.Mode = txt
			}
		}
		if pvText := strings.TrimSpace(modeSel.Find(".pointValue").Text()); pvText != "" {
			if pv, err := strconv.Atoi(pvText); err == nil {
				out.PointValue = pv
			}
		}
	}

	// Instructions
	instSel := doc.Find(".instructions").First()
	if instSel.Length() > 0 {
		out.InstructionsText = normalizeSpace(instSel.Text())
		if h, err := instSel.Html(); err == nil {
			out.InstructionsHTML = strings.TrimSpace(h)
		}
		if strongWord := normalizeSpace(instSel.Find("strong, b, .word").First().Text()); strongWord != "" {
			out.TargetWord = strongWord
		}
	}

	// Question content / sentence / quote / passage / image
	qcSel := doc.Find(".questionContent").First()
	if qcSel.Length() > 0 {
		if srcSel := qcSel.Find(".source").First(); srcSel.Length() > 0 {
			out.SourceText = normalizeSpace(srcSel.Text())
		}
		if imgSel := qcSel.Find("img").First(); imgSel.Length() > 0 {
			if src, ok := imgSel.Attr("src"); ok {
				out.PromptImageURL = strings.TrimSpace(src)
			}
		}
		if out.PromptImageURL == "" {
			qcSel.Find("[style]").EachWithBreak(func(_ int, s *goquery.Selection) bool {
				if style, ok := s.Attr("style"); ok {
					if m := bgImageURLRegex.FindStringSubmatch(style); len(m) > 1 {
						out.PromptImageURL = strings.TrimSpace(m[1])
						return false
					}
				}
				return true
			})
		}

		sentSel := qcSel.Find(".sentence").First()
		if sentSel.Length() == 0 {
			// Clone questionContent and strip .source so passage/custom questions (typeP, typeE, etc.)
			// retain their passage text even when a .source citation is present.
			clone := qcSel.Clone()
			clone.Find(".source").Remove()
			sentSel = clone
		}
		if sentSel.Length() > 0 {
			out.SentenceText = normalizeSpace(sentSel.Text())
			if h, err := sentSel.Html(); err == nil {
				out.SentenceHTML = strings.TrimSpace(h)
			}
			if bw := normalizeSpace(sentSel.Find(".blanked, strong, b, .word").First().Text()); bw != "" {
				out.BlankedWord = bw
				if out.TargetWord == "" && !isBlankPlaceholder(bw) {
					out.TargetWord = bw
				}
			}
		}
	}

	// For 4-image questions (typeI), target word can sit inside .word .wrapper (either inside strong/.word or as direct text)
	if out.TargetWord == "" {
		if w := normalizeSpace(doc.Find(".word .wrapper strong, .word .wrapper .word, .word strong").First().Text()); w != "" {
			out.TargetWord = w
		} else if w := normalizeSpace(doc.Find(".word .wrapper").First().Text()); w != "" {
			out.TargetWord = w
		}
	}

	// Choices
	doc.Find(".choices a").Each(func(idx int, a *goquery.Selection) {
		nonce, _ := a.Attr("data-nonce")
		accessKey, _ := a.Attr("accesskey")
		htmlInner, _ := a.Html()
		choice := ChoiceOption{
			Index:     idx,
			Nonce:     strings.TrimSpace(nonce),
			AccessKey: strings.TrimSpace(accessKey),
			Text:      normalizeSpace(a.Text()),
			HTML:      strings.TrimSpace(htmlInner),
		}
		if style, ok := a.Attr("style"); ok {
			if m := bgImageURLRegex.FindStringSubmatch(style); len(m) > 1 {
				choice.ImageURL = strings.TrimSpace(m[1])
			}
		}
		if choice.ImageURL == "" {
			if imgSrc, ok := a.Find("img").First().Attr("src"); ok {
				choice.ImageURL = strings.TrimSpace(imgSrc)
			}
		}
		out.Choices = append(out.Choices, choice)
	})

	// Validate that the slide contains recognizable question or results structure.
	if !out.IsResultsSlide && !out.IsSpelling && len(out.Choices) == 0 {
		return nil, fmt.Errorf("htmlparse: slide contains neither multiple-choice options (.choices a), spelling input (.spelltheword), nor results (.results)")
	}

	return out, nil
}

func isBlankPlaceholder(s string) bool {
	trimmed := strings.Trim(s, "_- \t\n\r")
	return trimmed == ""
}

// QuizQuestionResult represents one question row from GET /quiz/results/{aid}.
type QuizQuestionResult struct {
	QuestionNumber int    `json:"question_number"`
	UserChoice     int    `json:"user_choice"`
	AnswerOrder    string `json:"answer_order,omitempty"`
	PeopleCount    int    `json:"people_count,omitempty"`
	CorrectCount   int    `json:"correct_count,omitempty"`
	Stats          []int  `json:"stats,omitempty"`
	QuestionHTML   string `json:"question_html,omitempty"`
	QuestionText   string `json:"question_text,omitempty"`
}

// ParseQuizResultsHTML parses the HTML returned by GET /quiz/results/{aid}.
func ParseQuizResultsHTML(rawHTML string) ([]QuizQuestionResult, error) {
	if strings.TrimSpace(rawHTML) == "" {
		return nil, fmt.Errorf("htmlparse: empty quiz results HTML")
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader([]byte(rawHTML)))
	if err != nil {
		return nil, fmt.Errorf("htmlparse: parse quiz results HTML: %w", err)
	}
	var results []QuizQuestionResult
	doc.Find("ol.questions > li").Each(func(i int, li *goquery.Selection) {
		r := QuizQuestionResult{
			QuestionNumber: i + 1,
		}
		if cStr, ok := li.Attr("data-choice"); ok {
			r.UserChoice, _ = strconv.Atoi(strings.TrimSpace(cStr))
		}
		if ao, ok := li.Attr("data-ao"); ok {
			r.AnswerOrder = strings.TrimSpace(ao)
		}
		if pc, ok := li.Attr("data-pc"); ok {
			r.PeopleCount, _ = strconv.Atoi(strings.TrimSpace(pc))
		}
		if cc, ok := li.Attr("data-cc"); ok {
			r.CorrectCount, _ = strconv.Atoi(strings.TrimSpace(cc))
		}
		if statsRaw, ok := li.Attr("data-stats"); ok && strings.TrimSpace(statsRaw) != "" {
			var stats []int
			if err := json.Unmarshal([]byte(statsRaw), &stats); err == nil {
				r.Stats = stats
			}
		}
		if h, err := li.Html(); err == nil {
			r.QuestionHTML = strings.TrimSpace(h)
		}
		r.QuestionText = normalizeSpace(li.Text())
		results = append(results, r)
	})
	return results, nil
}

func normalizeSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
