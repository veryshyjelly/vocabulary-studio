package htmlparse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var posCodeToFullName = map[string]string{
	"n":      "noun",
	"v":      "verb",
	"a":      "adjective",
	"adj":    "adjective",
	"r":      "adverb",
	"adv":    "adverb",
	"pron":   "pronoun",
	"prep":   "preposition",
	"conj":   "conjunction",
	"interj": "interjection",
	"art":    "article",
	"idm":    "idiom",
	"abbr":   "abbreviation",
}

// PronunciationVariant holds regional IPA and audio/video pronunciation links from a dictionary page.
type PronunciationVariant struct {
	Region   string `json:"region"`
	IPA      string `json:"ipa,omitempty"`
	AudioID  string `json:"audio_id,omitempty"`
	AudioURL string `json:"audio_url,omitempty"`
	VideoURL string `json:"video_url,omitempty"`
}

// RelatedWordGroup holds a group of synonyms, antonyms, types, or hypernyms for a sense.
type RelatedWordGroup struct {
	Relation string            `json:"relation"`
	Words    []string          `json:"words,omitempty"`
	Glossed  map[string]string `json:"glossed,omitempty"`
}

// DictionarySense represents a single sense entry (`li.sense`) on `/dictionary/{word}`.
type DictionarySense struct {
	SynsetID     int                `json:"synset_id,omitempty"`
	PartOfSpeech string             `json:"part_of_speech"`
	POSCode      string             `json:"pos_code,omitempty"`
	Definition   string             `json:"definition"`
	Examples     []string           `json:"examples,omitempty"`
	Related      []RelatedWordGroup `json:"related,omitempty"`
}

// WordFamilyMember represents an entry in the embedded `<vcom:wordfamily data="[...]">` tree.
type WordFamilyMember struct {
	Word       string  `json:"word"`
	Headword   bool    `json:"hw,omitempty"`
	Parent     string  `json:"parent,omitempty"`
	Frequency  float64 `json:"freq"`
	FamilyFreq float64 `json:"ffreq"`
	Type       int     `json:"type"`
}

// RelatedListSummary represents a vocabulary list linked on a dictionary or WOD page.
type RelatedListSummary struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Author      string `json:"author,omitempty"`
	Description string `json:"description,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
}

// DictionaryEntry holds the parsed contents of `GET /dictionary/{word}`.
type DictionaryEntry struct {
	Word           string                 `json:"word"`
	Language       string                 `json:"language,omitempty"`
	Learnable      bool                   `json:"learnable"`
	HasImage       bool                   `json:"has_image,omitempty"`
	PrevWord       string                 `json:"prev_word,omitempty"`
	NextWord       string                 `json:"next_word,omitempty"`
	WordForms      []string               `json:"word_forms,omitempty"`
	ShortBlurb     string                 `json:"short_blurb,omitempty"`
	LongBlurb      string                 `json:"long_blurb,omitempty"`
	Pronunciations []PronunciationVariant `json:"pronunciations,omitempty"`
	Senses         []DictionarySense      `json:"senses,omitempty"`
	WordFamily     []WordFamilyMember     `json:"word_family,omitempty"`
	RelatedLists   []RelatedListSummary   `json:"related_lists,omitempty"`
}

// AutocompleteSuggestion represents a single suggestion from `GET /dictionary/autocomplete-ss`.
type AutocompleteSuggestion struct {
	Word       string `json:"word"`
	Definition string `json:"definition"`
	IPA        string `json:"ipa,omitempty"`
	ImageURL   string `json:"image_url,omitempty"`
	Learnable  bool   `json:"learnable"`
	URL        string `json:"url"`
}

// WordOfTheDayEntry holds the parsed contents of `/word-of-the-day/` or `/word-of-the-day/{YYYY-MM-DD}`.
type WordOfTheDayEntry struct {
	Date            string               `json:"date,omitempty"`
	Word            string               `json:"word"`
	AudioCode       string               `json:"audio_code,omitempty"`
	AudioURL        string               `json:"audio_url,omitempty"`
	ShortUsage      string               `json:"short_usage,omitempty"`
	LongDescription string               `json:"long_description,omitempty"`
	YesterdayURL    string               `json:"yesterday_url,omitempty"`
	TomorrowURL     string               `json:"tomorrow_url,omitempty"`
	RelatedLists    []RelatedListSummary `json:"related_lists,omitempty"`
}

// ParseDictionaryHTML parses `GET /dictionary/{word}` HTML (supporting both v1 and v2 dictionary layouts).
func ParseDictionaryHTML(rawHTML, audioBaseURL string) (*DictionaryEntry, error) {
	if strings.TrimSpace(rawHTML) == "" {
		return nil, fmt.Errorf("htmlparse: empty dictionary HTML")
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader([]byte(rawHTML)))
	if err != nil {
		return nil, fmt.Errorf("htmlparse: parse dictionary HTML: %w", err)
	}

	root := doc.Find(".wordPage.content-wrapper, .wordPage").First()
	entry := &DictionaryEntry{
		Language: "en",
	}

	if root.Length() > 0 {
		if w, ok := root.Attr("data-word"); ok {
			entry.Word = strings.TrimSpace(w)
		}
		if lang, ok := root.Attr("data-lang"); ok && lang != "" {
			entry.Language = strings.TrimSpace(lang)
		}
		if learnable, ok := root.Attr("data-learnable"); ok {
			entry.Learnable = learnable == "true"
		}
		if hasImg, ok := root.Attr("data-hasimage"); ok {
			entry.HasImage = hasImg == "true"
		}
		if prev, ok := root.Attr("data-prev"); ok {
			entry.PrevWord = strings.TrimSpace(prev)
		}
		if next, ok := root.Attr("data-next"); ok {
			entry.NextWord = strings.TrimSpace(next)
		}
	}

	if entry.Word == "" {
		entry.Word = normalizeSpace(doc.Find("h1#hdr-word-area, .word-area h1, h1.dynamictext").First().Text())
	}

	// Word forms (e.g. "callower; callowest")
	if wfText := strings.TrimSpace(doc.Find("p.word-forms > b, .word-forms b").First().Text()); wfText != "" {
		for _, part := range strings.Split(wfText, ";") {
			if p := strings.TrimSpace(part); p != "" {
				entry.WordForms = append(entry.WordForms, p)
			}
		}
	}

	// Short & long blurbs
	entry.ShortBlurb = normalizeSpace(doc.Find(".word-area p.short, p.short").First().Text())
	entry.LongBlurb = normalizeSpace(doc.Find(".word-area p.long, p.long").First().Text())

	// JSON-LD fallback if headword or long blurb is missing
	if entry.Word == "" || entry.LongBlurb == "" {
		doc.Find(`script[type="application/ld+json"]`).Each(func(_ int, s *goquery.Selection) {
			var ld struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			}
			if json.Unmarshal([]byte(s.Text()), &ld) == nil {
				if entry.Word == "" && strings.TrimSpace(ld.Name) != "" {
					entry.Word = strings.TrimSpace(ld.Name)
				}
				if entry.LongBlurb == "" && strings.TrimSpace(ld.Description) != "" {
					entry.LongBlurb = normalizeSpace(ld.Description)
				}
			}
		})
	}

	// Pronunciations (v2 layout: .ipa-section .ipa-with-audio)
	doc.Find(".ipa-section .ipa-with-audio").Each(func(_ int, sel *goquery.Selection) {
		pv := PronunciationVariant{Region: "US"}
		if sel.Find(".uk-flag-icon").Length() > 0 {
			pv.Region = "UK"
		}
		pv.IPA = strings.TrimSpace(sel.Find(".span-replace-h3, h3").First().Text())
		if audioID, ok := sel.Find("a.audio[data-audio]").First().Attr("data-audio"); ok {
			pv.AudioID = strings.TrimSpace(audioID)
			if pv.AudioID != "" && audioBaseURL != "" {
				pv.AudioURL = strings.TrimRight(audioBaseURL, "/") + "/" + pv.AudioID + ".mp3"
			}
		}
		if videoSrc, ok := sel.Find("audio.pron-audio[src], source[src]").First().Attr("src"); ok {
			pv.VideoURL = strings.TrimSpace(videoSrc)
		}
		if pv.IPA != "" || pv.AudioID != "" || pv.VideoURL != "" {
			entry.Pronunciations = append(entry.Pronunciations, pv)
		}
	})

	// Fallback v1 audio links if v2 .ipa-section was not present
	if len(entry.Pronunciations) == 0 {
		doc.Find(".word-area a.audio[data-audio], .header-container a.audio[data-audio], h1 a.audio[data-audio]").Each(func(_ int, a *goquery.Selection) {
			if audioID, ok := a.Attr("data-audio"); ok {
				audioID = strings.TrimSpace(audioID)
				if audioID != "" {
					pv := PronunciationVariant{
						Region:  "US",
						AudioID: audioID,
					}
					if audioBaseURL != "" {
						pv.AudioURL = strings.TrimRight(audioBaseURL, "/") + "/" + audioID + ".mp3"
					}
					entry.Pronunciations = append(entry.Pronunciations, pv)
				}
			}
		})
	}

	// Senses (.word-definitions > ol > li.sense, li.sense, or div.sense in v1)
	doc.Find(".word-definitions li.sense, li.sense, div.sense").Each(func(_ int, li *goquery.Selection) {
		sense := DictionarySense{}
		if idAttr, ok := li.Attr("id"); ok && strings.HasPrefix(idAttr, "s") {
			if sid, err := strconv.Atoi(strings.TrimPrefix(idAttr, "s")); err == nil {
				sense.SynsetID = sid
			}
		}
		if classAttr, ok := li.Attr("class"); ok {
			for _, tok := range strings.Fields(classAttr) {
				if strings.HasPrefix(tok, "pos_") {
					sense.POSCode = strings.TrimPrefix(tok, "pos_")
				}
			}
		}
		defSel := li.Find(".definition").First()
		if defSel.Length() > 0 {
			posText := normalizeSpace(defSel.Find(".pos-icon, .anchor").First().Text())
			sense.PartOfSpeech = posText
			clone := defSel.Clone()
			clone.Find(".pos-icon, .anchor").Remove()
			sense.Definition = normalizeSpace(clone.Text())
		}
		if sense.PartOfSpeech == "" && sense.POSCode != "" {
			if full, ok := posCodeToFullName[sense.POSCode]; ok {
				sense.PartOfSpeech = full
			} else {
				sense.PartOfSpeech = sense.POSCode
			}
		} else if full, ok := posCodeToFullName[strings.ToLower(sense.PartOfSpeech)]; ok {
			sense.PartOfSpeech = full
		}

		li.Find(".defContent > .example, > .example").Each(func(_ int, ex *goquery.Selection) {
			if txt := normalizeSpace(ex.Text()); txt != "" {
				sense.Examples = append(sense.Examples, txt)
			}
		})

		li.Find(".div-replace-dl.instances, dl.instances").Each(func(_ int, dl *goquery.Selection) {
			relLabel := strings.TrimSuffix(strings.ToLower(normalizeSpace(dl.Find("span.detail, dt").First().Text())), ":")
			if relLabel == "" {
				relLabel = "similar"
			}
			group := RelatedWordGroup{
				Relation: relLabel,
			}
			dl.Find("a.word").Each(func(_ int, w *goquery.Selection) {
				wordTxt := normalizeSpace(w.Text())
				if wordTxt != "" {
					group.Words = append(group.Words, wordTxt)
				}
			})
			dl.Find(".div-replace-dd, dd").Each(func(_ int, dd *goquery.Selection) {
				glossTxt := normalizeSpace(dd.Find(".definition").First().Text())
				if glossTxt != "" {
					if group.Glossed == nil {
						group.Glossed = make(map[string]string)
					}
					dd.Find("a.word").Each(func(_ int, w *goquery.Selection) {
						if wt := normalizeSpace(w.Text()); wt != "" {
							group.Glossed[wt] = glossTxt
						}
					})
				}
			})
			if len(group.Words) > 0 {
				sense.Related = append(sense.Related, group)
			}
		})

		if sense.Definition != "" {
			entry.Senses = append(entry.Senses, sense)
		}
	})

	// Word family (<vcom:wordfamily data="[...]">)
	wfNode := doc.Find("vcom\\:wordfamily, [data]").FilterFunction(func(_ int, s *goquery.Selection) bool {
		if len(s.Nodes) > 0 && strings.EqualFold(s.Nodes[0].Data, "vcom:wordfamily") {
			return true
		}
		return s.Parent().HasClass("family")
	}).First()
	if wfNode.Length() > 0 {
		if rawData, ok := wfNode.Attr("data"); ok && strings.TrimSpace(rawData) != "" {
			var members []WordFamilyMember
			if err := json.Unmarshal([]byte(rawData), &members); err == nil {
				entry.WordFamily = members
			}
		}
	}

	// Related vocabulary lists
	doc.Find(".section.related-lists .box-vocab-item, .box-vocab-list .box-vocab-item").Each(func(_ int, item *goquery.Selection) {
		titleSel := item.Find("a.vlist-title").First()
		href, _ := titleSel.Attr("href")
		imgSrc, _ := item.Find("img.img-vocab-list").First().Attr("src")
		rl := RelatedListSummary{
			Title:       normalizeSpace(titleSel.Text()),
			URL:         strings.TrimSpace(href),
			Author:      normalizeSpace(item.Find(".vlist-author").First().Text()),
			Description: normalizeSpace(item.Find("p.vlist-txt").First().Text()),
			ImageURL:    strings.TrimSpace(imgSrc),
		}
		if rl.Title != "" || rl.URL != "" {
			entry.RelatedLists = append(entry.RelatedLists, rl)
		}
	})

	if entry.Word == "" && len(entry.Senses) == 0 && entry.ShortBlurb == "" {
		return nil, fmt.Errorf("htmlparse: dictionary HTML missing headword and senses")
	}

	return entry, nil
}

// ParseAutocompleteHTML parses the HTML snippet returned by `GET /dictionary/autocomplete-ss`.
func ParseAutocompleteHTML(rawHTML string) ([]AutocompleteSuggestion, error) {
	if strings.TrimSpace(rawHTML) == "" {
		return nil, nil
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader([]byte(rawHTML)))
	if err != nil {
		return nil, fmt.Errorf("htmlparse: parse autocomplete HTML: %w", err)
	}
	var out []AutocompleteSuggestion
	doc.Find(".word-result.suggestion, li[word], div.suggestion").Each(func(_ int, s *goquery.Selection) {
		wordSpan := s.Find("span.word").First()
		wordText := normalizeSpace(wordSpan.Text())
		if wordText == "" {
			if wAttr, ok := s.Attr("word"); ok {
				wordText = strings.TrimSpace(wAttr)
			}
		}
		if wordText == "" {
			return
		}
		ipa, _ := wordSpan.Attr("data-ipa")
		img, _ := wordSpan.Attr("data-img")
		href, _ := s.Find("a.word-result-link, a").First().Attr("href")
		if href == "" {
			href = "/dictionary/" + wordText
		}
		out = append(out, AutocompleteSuggestion{
			Word:       wordText,
			Definition: normalizeSpace(s.Find("span.definition, .definition").First().Text()),
			IPA:        strings.TrimSpace(ipa),
			ImageURL:   strings.TrimSpace(img),
			Learnable:  s.HasClass("learnable"),
			URL:        strings.TrimSpace(href),
		})
	})
	return out, nil
}

// ParseWordOfTheDayHTML parses `/word-of-the-day/` or `/word-of-the-day/{YYYY-MM-DD}`.
func ParseWordOfTheDayHTML(rawHTML, audioBaseURL string) (*WordOfTheDayEntry, error) {
	if strings.TrimSpace(rawHTML) == "" {
		return nil, fmt.Errorf("htmlparse: empty Word of the Day HTML")
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader([]byte(rawHTML)))
	if err != nil {
		return nil, fmt.Errorf("htmlparse: parse WOD HTML: %w", err)
	}

	wod := &WordOfTheDayEntry{}
	wod.Word = normalizeSpace(doc.Find(".box-lks-audio-wod a.word-of-the-day, a.word-of-the-day").First().Text())

	dateSel := doc.Find(".box-yesterday-today-tomorrow").First().Clone()
	dateSel.Find("a").Remove()
	wod.Date = normalizeSpace(dateSel.Text())

	if yHref, ok := doc.Find("a.lk-yesterday").First().Attr("href"); ok {
		wod.YesterdayURL = strings.TrimSpace(yHref)
	}
	if tHref, ok := doc.Find("a.lk-tomorrow").First().Attr("href"); ok {
		wod.TomorrowURL = strings.TrimSpace(tHref)
	}

	btnAudio := doc.Find("button.btn-wod-audio, [data-code]").First()
	if code, ok := btnAudio.Attr("data-code"); ok {
		wod.AudioCode = strings.TrimSpace(code)
		if wod.AudioCode != "" && audioBaseURL != "" {
			wod.AudioURL = strings.TrimRight(audioBaseURL, "/") + "/" + wod.AudioCode + ".mp3"
		}
	}
	if wod.Word == "" {
		if dw, ok := btnAudio.Attr("data-wod"); ok {
			wod.Word = strings.TrimSpace(dw)
		}
	}

	wod.ShortUsage = normalizeSpace(doc.Find("p.txt-wod-usage").First().Text())
	wod.LongDescription = normalizeSpace(doc.Find("p.txt-wod-desc").First().Text())

	doc.Find(".box-vocab-list .box-vocab-item").Each(func(_ int, item *goquery.Selection) {
		titleSel := item.Find("a.vlist-title").First()
		href, _ := titleSel.Attr("href")
		imgSrc, _ := item.Find("img.img-vocab-list").First().Attr("src")
		rl := RelatedListSummary{
			Title:       normalizeSpace(titleSel.Text()),
			URL:         strings.TrimSpace(href),
			Author:      normalizeSpace(item.Find(".vlist-author").First().Text()),
			Description: normalizeSpace(item.Find("p.vlist-txt").First().Text()),
			ImageURL:    strings.TrimSpace(imgSrc),
		}
		if rl.Title != "" || rl.URL != "" {
			wod.RelatedLists = append(wod.RelatedLists, rl)
		}
	})

	if wod.Word == "" {
		return nil, fmt.Errorf("htmlparse: Word of the Day HTML missing target word")
	}

	return wod, nil
}
