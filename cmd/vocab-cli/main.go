package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	vocab "github.com/veryshyjelly/vocabulary-studio/sdk"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr, nil))
}

func run(ctx context.Context, rawArgs []string, stdin io.Reader, stdout, stderr io.Writer, extraOpts []vocab.Option) int {
	_ = vocab.LoadDotEnvFile(".env.local")
	_ = vocab.LoadDotEnvFile(".env")

	defaultSession := os.Getenv("VOCAB_SESSION_FILE")
	if defaultSession == "" {
		defaultSession = ".scratch/session.json"
	}
	defaultTZ := os.Getenv("VOCAB_TIMEZONE")
	if defaultTZ == "" {
		defaultTZ = "UTC"
	}

	fs := flag.NewFlagSet("vocab-cli", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var flagSession string
	var flagJSON bool
	var flagTimeout time.Duration
	fs.StringVar(&flagSession, "session", defaultSession, "Path to persistent session file")
	fs.BoolVar(&flagJSON, "json", false, "Print raw JSON output")
	fs.DurationVar(&flagTimeout, "timeout", 2*time.Minute, "Command timeout")

	if err := fs.Parse(rawArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(stdout)
			return 0
		}
		return 2
	}

	args := fs.Args()
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	opts := []vocab.Option{
		vocab.WithSessionStore(vocab.NewFileSessionStore(flagSession)),
		vocab.WithTimezone(defaultTZ),
	}
	opts = append(opts, extraOpts...)

	client, err := vocab.NewClient(opts...)
	if err != nil {
		fmt.Fprintf(stderr, "error: create client: %v\n", err)
		return 1
	}

	if flagTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, flagTimeout)
		defer cancel()
	}

	cmd := args[0]
	switch cmd {
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0

	case "login":
		username := os.Getenv("VOCAB_USERNAME")
		password := os.Getenv("VOCAB_PASSWORD")
		if len(args) >= 2 {
			username = args[1]
		}
		if len(args) >= 3 {
			password = args[2]
		}
		if strings.TrimSpace(username) == "" || password == "" {
			fmt.Fprintln(stderr, "error: provide username/password via VOCAB_USERNAME and VOCAB_PASSWORD in .env.local or CLI args")
			return 2
		}
		me, err := client.Auth.Login(ctx, username, password)
		if err != nil {
			fmt.Fprintf(stderr, "error: login failed: %v\n", err)
			return 1
		}
		if flagJSON {
			printJSON(stdout, me)
			return 0
		}
		fmt.Fprintf(stdout, "Logged in as %s (uid=%s, level=%s, points=%d, fullVocabTrainer=%v)\n",
			me.Auth.Nickname, me.Auth.UID, me.Level.Name, me.Points, me.Perms.FullVocabTrainer)
		fmt.Fprintf(stdout, "Session saved to %s\n", flagSession)
		return 0

	case "logout":
		if err := client.Auth.Logout(ctx); err != nil {
			fmt.Fprintf(stderr, "error: logout failed: %v\n", err)
			return 1
		}
		if flagJSON {
			printJSON(stdout, map[string]bool{"logged_out": true})
			return 0
		}
		fmt.Fprintf(stdout, "Logged out and cleared session %s\n", flagSession)
		return 0

	case "me":
		me, err := client.Auth.Me(ctx)
		if err != nil {
			fmt.Fprintf(stderr, "error: me failed: %v\n", err)
			return 1
		}
		if flagJSON {
			printJSON(stdout, me)
			return 0
		}
		fmt.Fprintf(stdout, "ValidUser: %v | LoggedIn: %v | UID: %s | Nickname: %s | Points: %d | Level: %s (%s)\n",
			me.ValidUser, me.Auth.LoggedIn, me.Auth.UID, me.Auth.Nickname, me.Points, me.Level.Name, me.Level.ID)
		return 0

	case "lookup":
		if len(args) < 2 {
			fmt.Fprintln(stderr, "error: usage: vocab-cli lookup <word>")
			return 2
		}
		word := args[1]
		entry, err := client.Dictionary.Lookup(ctx, word)
		if err != nil {
			fmt.Fprintf(stderr, "error: lookup %q: %v\n", word, err)
			return 1
		}
		if flagJSON {
			printJSON(stdout, entry)
			return 0
		}
		fmt.Fprintf(stdout, "=== %s (learnable=%v) ===\n", entry.Word, entry.Learnable)
		if entry.ShortBlurb != "" {
			fmt.Fprintf(stdout, "\nShort: %s\n", entry.ShortBlurb)
		}
		if entry.LongBlurb != "" {
			fmt.Fprintf(stdout, "\nLong:  %s\n", entry.LongBlurb)
		}
		for _, p := range entry.Pronunciations {
			fmt.Fprintf(stdout, "Pronunciation [%s]: IPA=%s Audio=%s\n", p.Region, p.IPA, p.AudioURL)
		}
		fmt.Fprintf(stdout, "\nSenses (%d):\n", len(entry.Senses))
		for i, s := range entry.Senses {
			fmt.Fprintf(stdout, "  %d. [%s] %s\n", i+1, s.PartOfSpeech, s.Definition)
		}
		return 0

	case "examples":
		if len(args) < 2 {
			fmt.Fprintln(stderr, "error: usage: vocab-cli examples <word>")
			return 2
		}
		ex, err := client.Dictionary.Examples(ctx, args[1], vocab.CorpusDomainAll, 0, 5)
		if err != nil {
			fmt.Fprintf(stderr, "error: examples: %v\n", err)
			return 1
		}
		if flagJSON {
			printJSON(stdout, ex)
			return 0
		}
		fmt.Fprintf(stdout, "Total Corpus Hits: %d\n", ex.TotalHits)
		for i, s := range ex.Sentences {
			src := ""
			if s.Volume != nil && s.Volume.Title != "" {
				src = s.Volume.Title
			}
			fmt.Fprintf(stdout, "  %d. %s (%s)\n", i+1, s.Sentence, src)
		}
		return 0

	case "wod":
		date := ""
		if len(args) >= 2 {
			date = args[1]
		}
		wod, err := client.Dictionary.WordOfTheDay(ctx, date)
		if err != nil {
			fmt.Fprintf(stderr, "error: wod: %v\n", err)
			return 1
		}
		if flagJSON {
			printJSON(stdout, wod)
			return 0
		}
		fmt.Fprintf(stdout, "Word of the Day (%s): %s\nAudio: %s\n\n%s\n\n%s\n",
			wod.Date, wod.Word, wod.AudioURL, wod.ShortUsage, wod.LongDescription)
		return 0

	case "list":
		if len(args) < 2 {
			fmt.Fprintln(stderr, "error: usage: vocab-cli list <listID>")
			return 2
		}
		listID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			fmt.Fprintf(stderr, "error: invalid listID %q: %v\n", args[1], err)
			return 2
		}
		wl, err := client.Lists.Get(ctx, listID)
		if err != nil {
			fmt.Fprintf(stderr, "error: get list %d: %v\n", listID, err)
			return 1
		}
		if flagJSON {
			printJSON(stdout, wl)
			return 0
		}
		fmt.Fprintf(stdout, "List #%d: %s (%d words)\n%s\n", wl.WordlistID, wl.Name, len(wl.Words), wl.Description)
		for i, w := range wl.Words {
			if i >= 15 {
				fmt.Fprintf(stdout, "  ... and %d more words\n", len(wl.Words)-15)
				break
			}
			fmt.Fprintf(stdout, "  - %-16s : %s\n", w.Word, w.Definition)
		}
		return 0

	case "search-lists":
		if len(args) < 2 {
			fmt.Fprintln(stderr, "error: usage: vocab-cli search-lists <query>")
			return 2
		}
		res, err := client.Lists.Search(ctx, strings.Join(args[1:], " "), 0, 10, false)
		if err != nil {
			fmt.Fprintf(stderr, "error: search-lists: %v\n", err)
			return 1
		}
		if flagJSON {
			printJSON(stdout, res)
			return 0
		}
		fmt.Fprintf(stdout, "Found %d lists:\n", res.TotalResults)
		for _, item := range res.Results {
			fmt.Fprintf(stdout, "  [%d] %s (%d words) — %s\n", item.WordlistID, item.Name, item.WordCount, item.Description)
		}
		return 0

	case "play":
		var listID int64
		actType := vocab.ActivityVocabTrainer
		if len(args) >= 2 {
			id, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil || id <= 0 {
				fmt.Fprintf(stderr, "error: invalid optional listID %q\n", args[1])
				return 2
			}
			listID = id
			actType = vocab.ActivityListPractice
		}
		session, state, err := client.Trainer.NewSession(ctx, vocab.StartChallengeRequest{
			ActivityType: actType,
			WordlistID:   listID,
		})
		if err != nil {
			fmt.Fprintf(stderr, "error: start trainer: %v\n", err)
			return 1
		}
		if flagJSON {
			printJSON(stdout, state)
			return 0
		}
		return runInteractivePlay(ctx, session, stdin, stdout, stderr)

	default:
		fmt.Fprintf(stderr, "error: unknown command %q\n\n", cmd)
		printUsage(stderr)
		return 2
	}
}

func runInteractivePlay(ctx context.Context, session *vocab.TrainerSession, stdin io.Reader, stdout, stderr io.Writer) int {
	reader := bufio.NewReader(stdin)
	for step := 1; step <= 10; step++ {
		if err := ctx.Err(); err != nil {
			fmt.Fprintln(stderr, "\nsession interrupted")
			return 130
		}
		q := session.CurrentQuestion()
		if q == nil {
			fmt.Fprintln(stdout, "No active question.")
			return 0
		}
		fmt.Fprintf(stdout, "\n--- Question %d (Type=%s, Mode=%s, Points=%d) ---\n",
			step, q.Type, q.Mode, q.PointValue)
		if q.SentenceText != "" {
			fmt.Fprintf(stdout, "Context: %s\n", q.SentenceText)
		}
		if q.InstructionsText != "" {
			fmt.Fprintf(stdout, "Prompt:  %s\n", q.InstructionsText)
		}
		if q.IsSpelling {
			fmt.Fprintf(stdout, "Audio:   %s\n", q.AudioURL)
			for {
				fmt.Fprint(stdout, "Type spelling (or 'q' to quit): ")
				line, readErr := reader.ReadString('\n')
				line = strings.TrimSpace(line)
				if line == "q" || (readErr != nil && line == "") {
					return 0
				}
				outcome, ok, err := session.Spell(ctx, line)
				if err != nil {
					fmt.Fprintf(stderr, "error: spell: %v\n", err)
					return 1
				}
				if ok {
					fmt.Fprintf(stdout, "Correct! Word=%s (+%d pts)\n", outcome.Word, outcome.Points)
					break
				}
				fmt.Fprintln(stdout, "Incorrect — try spelling it again!")
				if errors.Is(readErr, io.EOF) {
					return 0
				}
			}
		} else {
			for i, ch := range q.Choices {
				label := ch.Text
				if label == "" && ch.ImageURL != "" {
					label = "[Image] " + ch.ImageURL
				}
				fmt.Fprintf(stdout, "  [%d] %s\n", i+1, label)
			}
			for {
				fmt.Fprint(stdout, "Choose 1-4, 'h' for 50/50 hint, or 'q' to quit: ")
				line, readErr := reader.ReadString('\n')
				line = strings.TrimSpace(line)
				if line == "q" || (readErr != nil && line == "") {
					return 0
				}
				if line == "h" {
					hResp, err := session.Hint(ctx, vocab.HintFiftyFifty)
					if err != nil {
						fmt.Fprintf(stdout, "Hint unavailable: %v\n", err)
					} else {
						fmt.Fprintf(stdout, "Eliminated nonces: %v\n", hResp.Nonces)
					}
					if errors.Is(readErr, io.EOF) {
						return 0
					}
					continue
				}
				idx, err := strconv.Atoi(line)
				if err != nil || idx < 1 || idx > len(q.Choices) {
					fmt.Fprintln(stdout, "Invalid selection.")
					if errors.Is(readErr, io.EOF) {
						return 0
					}
					continue
				}
				outcome, ok, err := session.ChooseIndex(ctx, idx-1)
				if err != nil {
					fmt.Fprintf(stderr, "error: choose: %v\n", err)
					return 1
				}
				if ok {
					fmt.Fprintf(stdout, "Correct! Word=%s (+%d pts)\n", outcome.Word, outcome.Points)
					if outcome.Blurb != nil && outcome.Blurb.Short != "" {
						fmt.Fprintf(stdout, "Blurb: %s\n", outcome.Blurb.Short)
					}
					break
				}
				fmt.Fprintln(stdout, "Incorrect — try another choice!")
				if errors.Is(readErr, io.EOF) {
					return 0
				}
			}
		}
		if step < 10 {
			if _, err := session.Next(ctx); err != nil {
				fmt.Fprintf(stderr, "error: next question: %v\n", err)
				return 1
			}
		}
	}
	return 0
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, `Usage: vocab-cli [--session .scratch/session.json] [--timeout 2m] [--json] <command> [args...]

Commands:
  login [username] [password]   Log in and cache session cookies + Bearer JWT
  logout                        Log out and clear cached session
  me                            Show current session, points, level, and subscription permissions
  lookup <word>                 Look up a word in the Vocabulary.com dictionary
  examples <word>               Fetch real-world example sentences ("Word in the Wild")
  wod [YYYY-MM-DD]              Fetch Word of the Day
  list <listID>                 Fetch a vocabulary list by ID
  search-lists <query>          Search public vocabulary lists
  play [optionalListID]         Play an interactive round of VocabTrainer or List Practice`)
}

func printJSON(w io.Writer, v any) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
