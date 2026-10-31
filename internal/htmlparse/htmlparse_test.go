package htmlparse

import (
	"encoding/base64"
	"fmt"
	"testing"
)

func TestAll16QuestionTypesParsing(t *testing.T) {
	multipleChoiceTypes := []struct {
		code       string
		modeClass  string
		wantMode   string
		points     int
		hasPassage bool
	}{
		{"A", "assessment", "ASSESSMENT", 50, false},
		{"D", "progress", "PROGRESS", 100, false},
		{"P", "review", "REVIEW", 75, true},
		{"F", "mastery", "BRUSH-UP", 120, true},
		{"G", "progress", "PROGRESS", 100, false},
		{"L", "assessment", "ASSESSMENT", 100, true},
		{"H", "progress", "PROGRESS", 100, true},
		{"S", "review", "REVIEW", 80, false},
		{"Y", "progress", "PROGRESS", 90, false},
		{"R", "mastery", "BRUSH-UP", 110, false},
		{"C", "assessment", "ASSESSMENT", 60, false},
		{"M", "progress", "PROGRESS", 100, true},
		{"E", "review", "REVIEW", 85, true},
		{"U", "progress", "PROGRESS", 95, true},
	}

	for _, tc := range multipleChoiceTypes {
		t.Run("Type_"+tc.code, func(t *testing.T) {
			contentBlock := ""
			if tc.hasPassage {
				contentBlock = `<div class="questionContent"><p>Passage context for <strong>callow</strong> in type ` + tc.code + `.</p><div class="source">The New Yorker</div></div>`
			} else if tc.code == "G" {
				contentBlock = `<div class="questionContent"><div class="image" style="background-image: url('https://cdn.vocabulary.com/images/single.jpg')"></div></div>`
			}
			rawHTML := fmt.Sprintf(`<div class="challenge-slide" data-audio="C/1XB5QC9VRN0HU">
				<div class="question type%s" data-template="multiple-choice">
					<div class="mode"><div class="%s"></div><div class="pointValue">%d</div></div>
					%s
					<div class="instructions">Question instructions for <strong>callow</strong>:</div>
					<div class="choices">
						<a href="javascript:void(0)" data-nonce="n0_%s" accesskey="1A">Choice A</a>
						<a href="javascript:void(0)" data-nonce="n1_%s" accesskey="2B">Choice B</a>
						<a href="javascript:void(0)" data-nonce="n2_%s" accesskey="3C">Choice C</a>
						<a href="javascript:void(0)" data-nonce="n3_%s" accesskey="4D">Choice D</a>
					</div>
				</div>
			</div>`, tc.code, tc.modeClass, tc.points, contentBlock, tc.code, tc.code, tc.code, tc.code)

			b64 := base64.StdEncoding.EncodeToString([]byte(rawHTML))
			slide, err := DecodeAndParseSlide(b64, "")
			if err != nil {
				t.Fatalf("DecodeAndParseSlide(%s) failed: %v", tc.code, err)
			}
			if slide.QuestionType != tc.code {
				t.Errorf("QuestionType = %q, want %q", slide.QuestionType, tc.code)
			}
			if slide.Mode != tc.wantMode || slide.PointValue != tc.points {
				t.Errorf("Mode=%q PointValue=%d, want %q / %d", slide.Mode, slide.PointValue, tc.wantMode, tc.points)
			}
			if slide.TargetWord != "callow" {
				t.Errorf("TargetWord = %q, want callow", slide.TargetWord)
			}
			if tc.hasPassage {
				if slide.SentenceText == "" || slide.SourceText != "The New Yorker" {
					t.Errorf("SentenceText=%q SourceText=%q", slide.SentenceText, slide.SourceText)
				}
			}
			if tc.code == "G" && slide.PromptImageURL != "https://cdn.vocabulary.com/images/single.jpg" {
				t.Errorf("PromptImageURL = %q", slide.PromptImageURL)
			}
			if len(slide.Choices) != 4 || slide.Choices[2].Nonce != "n2_"+tc.code {
				t.Errorf("unexpected choices: %+v", slide.Choices)
			}
		})
	}

	// Type I: 4-Image Choices (with both strong wrapper and direct wrapper text)
	t.Run("Type_I_FourImages", func(t *testing.T) {
		imgHTML := `<div class="challenge-slide">
			<div class="question typeI" data-template="image-choice">
				<div class="mode"><div class="interrogate">CUSTOM MODE</div><div class="pointValue">50</div></div>
				<div class="word"><div class="wrapper">kumquat</div></div>
				<div class="choices">
					<a data-nonce="img_1" accesskey="1" style="background-image: url(&quot;https://cdn.vocabulary.com/images/q1.jpg&quot;)"></a>
					<a data-nonce="img_2" accesskey="2" style="background-image: url('https://cdn.vocabulary.com/images/q2.jpg')"></a>
					<a data-nonce="img_3" accesskey="3" style="background-image: url(https://cdn.vocabulary.com/images/q3.jpg)"></a>
					<a data-nonce="img_4" accesskey="4"><img src="https://cdn.vocabulary.com/images/q4.jpg"/></a>
				</div>
			</div>
		</div>`
		b64Raw := base64.RawStdEncoding.EncodeToString([]byte(imgHTML)) // test unpadded base64 too
		slide, err := DecodeAndParseSlide(b64Raw, "")
		if err != nil {
			t.Fatalf("DecodeAndParseSlide(I) failed: %v", err)
		}
		if slide.QuestionType != "I" || slide.TargetWord != "kumquat" || slide.Mode != "CUSTOM MODE" {
			t.Errorf("unexpected slide: %+v", slide)
		}
		if len(slide.Choices) != 4 || slide.Choices[0].ImageURL != "https://cdn.vocabulary.com/images/q1.jpg" || slide.Choices[3].ImageURL != "https://cdn.vocabulary.com/images/q4.jpg" {
			t.Errorf("unexpected image choices: %+v", slide.Choices)
		}
	})

	// Type T: Spelling Question
	t.Run("Type_T_Spelling", func(t *testing.T) {
		spellHTML := `<div class="challenge-slide" data-audio="K/1A2B3C">
			<div class="question typeT">
				<div class="mode"><div class="mastery"></div><div class="pointValue">150</div></div>
				<div class="questionContent"><div class="sentence">She ate a <span class="blanked">_______</span> from the tree.</div></div>
				<div class="spelltheword">
					<input type="text" class="wordspelling" />
					<button class="playword"></button>
					<button class="spellit">Spell it</button>
				</div>
			</div>
		</div>`
		slide, err := ParseSlideHTML(spellHTML, "T")
		if err != nil {
			t.Fatalf("ParseSlideHTML(T) failed: %v", err)
		}
		if !slide.IsSpelling || slide.QuestionType != "T" || slide.Mode != "BRUSH-UP" || slide.TargetWord != "" {
			t.Errorf("unexpected spelling slide: %+v", slide)
		}
	})
}

func TestSlideAndQuizParseErrorsAndEdgeCases(t *testing.T) {
	if _, err := DecodeAndParseSlide("", "D"); err == nil {
		t.Error("expected error for empty base64 code")
	}
	if _, err := DecodeAndParseSlide("!!!not_base64!!!", "D"); err == nil {
		t.Error("expected error for invalid base64 code")
	}
	if _, err := ParseSlideHTML("   ", "D"); err == nil {
		t.Error("expected error for empty slide HTML")
	}
	if _, err := ParseSlideHTML("<div><p>Random unrelated markup</p></div>", "D"); err == nil {
		t.Error("expected error for slide missing choices, spelling input, and results")
	}

	// Quiz results slide detection
	resSlide, err := ParseSlideHTML(`<div class="results"><h3>Quiz Complete!</h3></div>`, "")
	if err != nil || !resSlide.IsResultsSlide {
		t.Fatalf("expected results slide, got %+v, err=%v", resSlide, err)
	}

	// Quiz results breakdown with data-stats
	resultsHTML := `<ol class="questions">
		<li data-choice="0" data-ao="2143" data-pc="120" data-cc="95" data-stats="[95, 10, 10, 5]">
			<div class="q-text">What does <strong>callow</strong> mean?</div>
		</li>
	</ol>`
	qResults, err := ParseQuizResultsHTML(resultsHTML)
	if err != nil || len(qResults) != 1 {
		t.Fatalf("ParseQuizResultsHTML failed: %+v, err=%v", qResults, err)
	}
	if qResults[0].AnswerOrder != "2143" || qResults[0].PeopleCount != 120 || len(qResults[0].Stats) != 4 || qResults[0].Stats[0] != 95 {
		t.Errorf("unexpected QuizQuestionResult: %+v", qResults[0])
	}
	if _, err := ParseQuizResultsHTML(""); err == nil {
		t.Error("expected error for empty quiz results HTML")
	}
}

func TestParseDictionaryV1AndV2AndErrors(t *testing.T) {
	// v2 layout
	dictV2HTML := `<div class="wordPage content-wrapper" data-word="callow" data-lang="en" data-learnable="true" data-hasimage="false" data-prev="callosity" data-next="callowness">
		<div class="word-area">
			<h1 id="hdr-word-area">callow</h1>
			<div class="ipa-section">
				<div class="ipa-with-audio">
					<span class="us-flag-icon"></span>
					<span class="span-replace-h3">/ˈkæloʊ/</span>
					<a class="audio" data-audio="C/1XB5QC9VRN0HU"></a>
				</div>
				<div class="ipa-with-audio">
					<span class="uk-flag-icon"></span>
					<span class="span-replace-h3">/ˈkæləʊ/</span>
					<audio class="pron-audio" src="https://sd-pronunciation-processed-videos.sdcdns.com/desktop/uk_callow.mp4"></audio>
				</div>
			</div>
			<p class="word-forms">Other forms: <b>callower; callowest</b></p>
			<p class="short">Someone who is callow is young and inexperienced.</p>
			<p class="long">Callow comes from the Old English word calu meaning bald.</p>
		</div>
		<div class="word-definitions">
			<ol>
				<li id="s101016" class="sense pos_a ord1">
					<div class="definition"><div class="pos-icon">adjective</div>young and inexperienced</div>
					<div class="defContent">
						<div class="example">"a callow youth of seventeen"</div>
						<div class="div-replace-dl instances">
							<span class="detail">synonyms:</span>
							<span><a class="word">fledgling</a>, <a class="word">unfledged</a></span>
							<div class="div-replace-dd">
								<a class="word">inexperienced</a>
								<div class="definition">lacking practical knowledge</div>
							</div>
						</div>
					</div>
				</li>
			</ol>
		</div>
		<div class="section family">
			<vcom:wordfamily word="callow" lang="en" data='[{"word":"callow","hw":true,"freq":0.508,"ffreq":0.524,"type":0},{"word":"callowness","hw":true,"parent":"callow","freq":0.013,"ffreq":0.013,"type":2}]'></vcom:wordfamily>
		</div>
		<div class="section related-lists">
			<div class="box-vocab-item">
				<a class="vlist-title" href="/lists/12345">SAT High Frequency</a>
				<div class="vlist-author">Vocabulary.com</div>
				<p class="vlist-txt">Essential SAT adjectives.</p>
			</div>
		</div>
	</div>`

	entry, err := ParseDictionaryHTML(dictV2HTML, "https://audio.vocabulary.com/1.0/us")
	if err != nil {
		t.Fatalf("ParseDictionaryHTML v2 failed: %v", err)
	}
	if entry.Word != "callow" || !entry.Learnable || len(entry.WordForms) != 2 {
		t.Errorf("unexpected entry header: %+v", entry)
	}
	if len(entry.Pronunciations) != 2 || entry.Pronunciations[0].AudioURL != "https://audio.vocabulary.com/1.0/us/C/1XB5QC9VRN0HU.mp3" {
		t.Errorf("unexpected pronunciations: %+v", entry.Pronunciations)
	}
	if len(entry.Senses) != 1 || entry.Senses[0].SynsetID != 101016 || entry.Senses[0].Definition != "young and inexperienced" {
		t.Errorf("unexpected senses: %+v", entry.Senses)
	}
	if len(entry.Senses[0].Related) != 1 || entry.Senses[0].Related[0].Glossed["inexperienced"] != "lacking practical knowledge" {
		t.Errorf("unexpected related words: %+v", entry.Senses[0].Related)
	}
	if len(entry.WordFamily) != 2 || entry.WordFamily[1].Word != "callowness" {
		t.Errorf("unexpected word family: %+v", entry.WordFamily)
	}

	// v1 fallback layout + JSON-LD + POS code expansion
	dictV1HTML := `<html><head>
		<script type="application/ld+json">{"name":"present","description":"Present can mean a gift or right now."}</script>
	</head><body>
		<div class="header-container"><a class="audio" data-audio="P/PRESENT123"></a></div>
		<div class="sense pos_v" id="s200">
			<div class="definition"><div class="anchor">v</div>give an exhibition of to an interested audience</div>
		</div>
	</body></html>`
	v1Entry, err := ParseDictionaryHTML(dictV1HTML, "https://audio.vocabulary.com/1.0/us")
	if err != nil {
		t.Fatalf("ParseDictionaryHTML v1 failed: %v", err)
	}
	if v1Entry.Word != "present" || v1Entry.LongBlurb != "Present can mean a gift or right now." {
		t.Errorf("unexpected v1 entry: %+v", v1Entry)
	}
	if len(v1Entry.Pronunciations) != 1 || v1Entry.Pronunciations[0].AudioID != "P/PRESENT123" {
		t.Errorf("unexpected v1 pronunciations: %+v", v1Entry.Pronunciations)
	}
	if len(v1Entry.Senses) != 1 || v1Entry.Senses[0].PartOfSpeech != "verb" {
		t.Errorf("unexpected v1 sense: %+v", v1Entry.Senses)
	}

	// Dictionary error cases
	if _, err := ParseDictionaryHTML("", "https://audio.vocabulary.com/1.0/us"); err == nil {
		t.Error("expected error for empty dictionary HTML")
	}
	if _, err := ParseDictionaryHTML("<html><body><div>404 Not Found</div></body></html>", ""); err == nil {
		t.Error("expected error for HTML missing headword and senses")
	}

	// Autocomplete & WOD
	acHTML := `<div class="suggestions word-table">
		<div class="word-result suggestion learnable">
			<a class="word-result-link" href="/dictionary/callow">
				<span class="word" data-ipa="/ˈkæloʊ/">callow</span>
				<span class="definition">young and inexperienced</span>
			</a>
		</div>
	</div>`
	suggestions, err := ParseAutocompleteHTML(acHTML)
	if err != nil || len(suggestions) != 1 || suggestions[0].Word != "callow" || !suggestions[0].Learnable {
		t.Fatalf("ParseAutocompleteHTML failed: %+v, err=%v", suggestions, err)
	}

	wodHTML := `<div>
		<div class="box-yesterday-today-tomorrow">
			<a class="lk-yesterday" href="/word-of-the-day/2026-10-30"></a>
			October 31, 2026
			<a class="lk-tomorrow" href="/word-of-the-day/2026-11-01"></a>
		</div>
		<div class="box-lks-audio-wod"><a class="word-of-the-day" href="/dictionary/enchantment">enchantment</a></div>
		<button class="btn-wod-audio" data-code="E/WCWZUN54K8ZD" data-wod="enchantment"></button>
		<p class="txt-wod-usage">Short usage text.</p>
		<p class="txt-wod-desc">Long description text.</p>
	</div>`
	wod, err := ParseWordOfTheDayHTML(wodHTML, "https://audio.vocabulary.com/1.0/us")
	if err != nil {
		t.Fatalf("ParseWordOfTheDayHTML failed: %v", err)
	}
	if wod.Word != "enchantment" || wod.Date != "October 31, 2026" || wod.AudioURL != "https://audio.vocabulary.com/1.0/us/E/WCWZUN54K8ZD.mp3" {
		t.Errorf("unexpected WOD: %+v", wod)
	}
	if _, err := ParseWordOfTheDayHTML("<html><body>No WOD here</body></html>", ""); err == nil {
		t.Error("expected error when WOD HTML is missing word")
	}
}
