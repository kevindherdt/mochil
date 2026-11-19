package web

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"mochil/internal/appconfig"
	"mochil/internal/audio"
)

// When you ask the assistant to play something ("play Lilac by IU"), the server looks in your own audio libraries,
// answers straight away with a Play card for what it finds. A small chat model cannot be trusted to call a search tool
// and format a card, and the slow fallback model would keep you waiting for a sentence the server already knows.

// hermesCard is one thing from the library that the chat offers to play.
type hermesCard struct {
	Kind       string `json:"kind"`
	Library    string `json:"library"`
	ID         string `json:"id"`
	Title      string `json:"title"`
	Subtitle   string `json:"subtitle,omitempty"`
	StreamURL  string `json:"stream_url"`
	ArtworkURL string `json:"artwork_url,omitempty"`
}

const maximumHermesCards = 4

var (
	hermesPlayIntent = regexp.MustCompile(`\b(play|put on|listen to|start playing|speel|luister)\b`)
	// Asking for a speaker, a TV or a radio station is a Home Assistant request, not a library search.
	hermesDeviceWords = regexp.MustCompile(`\b(radio|speakers?|jbl|tv|television|living room|bedroom|kitchen|home group|everywhere|station|stream)\b`)
	hermesWordPattern = regexp.MustCompile(`[\p{L}\p{N}']+`)
	hermesStopWords   = map[string]bool{}
)

func init() {
	for _, word := range strings.Fields(`the a an some any my me i we you u your please can could would will hey hi hello ok okay
		song songs track tracks music audiobook audiobooks book podcast podcasts episode episodes album by from of on in at to for with
		app here phone now just local playback play put start playing listen something anything something want like it that this one
		library libraries collection collections local locally file files offline downloaded saved own mine
		again then and or is are be do ik wil graag een de het van door ze op in naar speel luister maar eens`) {
		hermesStopWords[word] = true
	}
}

// hermesPlayWords returns the words to look for when the message asks to play something from the library, or nil
// when it is not such a request.
func hermesPlayWords(message, assistantName string) []string {
	text := strings.ToLower(strings.TrimSpace(message))
	if len(text) > 160 || !hermesPlayIntent.MatchString(text) || hermesDeviceWords.MatchString(text) {
		return nil
	}
	name := strings.ToLower(strings.TrimSpace(assistantName))
	var words []string
	for _, word := range hermesWordPattern.FindAllString(text, -1) {
		word = strings.Trim(word, "'")
		if len(word) < 2 || hermesStopWords[word] || word == name {
			continue
		}
		words = append(words, word)
	}
	return words
}

// searchTokens lowercases text and splits it into words, dropping punctuation.
func searchTokens(text string) []string {
	return hermesWordPattern.FindAllString(strings.ToLower(text), -1)
}

func editDistance(a, b string) int {
	x, y := []rune(a), []rune(b)
	previous := make([]int, len(y)+1)
	for j := range previous {
		previous[j] = j
	}
	for i := 1; i <= len(x); i++ {
		current := make([]int, len(y)+1)
		current[0] = i
		for j := 1; j <= len(y); j++ {
			cost := 1
			if x[i-1] == y[j-1] {
				cost = 0
			}
			current[j] = min(previous[j]+1, current[j-1]+1, previous[j-1]+cost)
		}
		previous = current
	}
	return previous[len(y)]
}

// wordFits says whether a word is in the text, allowing a typo in longer words when fuzzy is set.
func wordFits(word, text string, tokens []string, fuzzy bool) bool {
	if strings.Contains(text, word) {
		return true
	}
	if !fuzzy || len(word) < 4 {
		return false
	}
	allowed := 1
	if len(word) >= 7 {
		allowed = 2
	}
	for _, token := range tokens {
		if len(token) >= 3 && editDistance(word, token) <= allowed {
			return true
		}
	}
	return false
}

// matchAudioItems keeps the items that contain every word somewhere (title, artist, album, series), best first: a title
// that is exactly a word first, then the more of the words in the title, then the shorter title. When nothing matches
// exactly it tries again forgiving a typo ("lilak" finds "Lilac").
func matchAudioItems(items []audio.Item, words []string, limit int) []audio.Item {
	if len(words) == 0 || limit <= 0 {
		return nil
	}
	if found := matchAudioPass(items, words, limit, false); len(found) > 0 {
		return found
	}
	return matchAudioPass(items, words, limit, true)
}

func matchAudioPass(items []audio.Item, words []string, limit int, fuzzy bool) []audio.Item {
	type scored struct {
		item   audio.Item
		inName int
		exact  bool
	}
	var found []scored
	for _, item := range items {
		title := strings.ToLower(item.Title)
		all := title + " " + strings.ToLower(item.Artist+" "+item.AlbumArtist+" "+item.Album+" "+item.Series)
		var tokens, titleTokens []string
		if fuzzy {
			tokens, titleTokens = searchTokens(all), searchTokens(title)
		}
		inName, ok, exact := 0, true, false
		for _, word := range words {
			if !wordFits(word, all, tokens, fuzzy) {
				ok = false
				break
			}
			if wordFits(word, title, titleTokens, fuzzy) {
				inName++
			}
			if strings.TrimSpace(title) == word {
				exact = true
			}
		}
		if ok {
			found = append(found, scored{item, inName, exact})
		}
	}
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].exact != found[j].exact {
			return found[i].exact
		}
		if found[i].inName != found[j].inName {
			return found[i].inName > found[j].inName
		}
		return len(found[i].item.Title) < len(found[j].item.Title)
	})
	if len(found) > limit {
		found = found[:limit]
	}
	out := make([]audio.Item, len(found))
	for i := range found {
		out[i] = found[i].item
	}
	return out
}

// hermesPlayLookup returns the cards for a play request and the line to answer it with, or nothing.
func (m *multiServer) hermesPlayLookup(account, message string, settings appconfig.Hermes) ([]hermesCard, string) {
	if !settings.Feature("library_play") {
		return nil, ""
	}
	words := hermesPlayWords(message, settings.Name)
	if len(words) == 0 {
		return nil, ""
	}
	var cards []hermesCard
	for _, library := range m.libraries {
		if library.Audio == nil || len(cards) >= maximumHermesCards {
			continue
		}
		items, _ := library.Audio.Snapshot()
		for _, item := range matchAudioItems(items, words, maximumHermesCards-len(cards)) {
			view := m.audioView(account, library, item)
			subtitle := strings.TrimSpace(item.Artist)
			if subtitle == "" {
				subtitle = strings.TrimSpace(item.Album)
			}
			cards = append(cards, hermesCard{Kind: library.Kind, Library: library.ID, ID: item.ID, Title: item.Title, Subtitle: subtitle, StreamURL: view.StreamURL, ArtworkURL: view.ArtworkURL})
		}
	}
	if len(cards) == 0 {
		return nil, ""
	}
	return cards, hermesQuickReply(cards)
}

// hermesQuickReply is what is said for a play request. The server already knows what it found, so the model is not
// asked: the answer is instant even when only the slow fallback model is up.
func hermesQuickReply(cards []hermesCard) string {
	first := cards[0]
	name := fmt.Sprintf("\u201c%s\u201d", first.Title)
	if first.Subtitle != "" {
		name += " by " + first.Subtitle
	}
	if len(cards) == 1 {
		return "Found it: " + name + ". Tap Play."
	}
	return "Found " + name + ", and a few more that match. Tap Play on the one you want."
}

func hermesCardsEvent(cards []hermesCard) []byte {
	body, _ := json.Marshal(map[string]any{"cards": cards})
	return []byte("event: hermes.cards\ndata: " + string(body) + "\n\n")
}
