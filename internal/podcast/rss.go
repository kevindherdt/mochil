package podcast

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"html"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const maximumDescriptionRunes = 8_000

var podcastHTMLTag = regexp.MustCompile(`(?s)<[^>]*>`)

type rss struct {
	Channel struct {
		Title       string `xml:"title"`
		Description string `xml:"description"`
		Image       struct {
			URL  string `xml:"url"`
			Href string `xml:"href,attr"`
		} `xml:"image"`
		Items []struct {
			GUID        string `xml:"guid"`
			Title       string `xml:"title"`
			Description string `xml:"description"`
			Summary     string `xml:"summary"`
			Content     string `xml:"encoded"`
			Link        string `xml:"link"`
			PubDate     string `xml:"pubDate"`
			Duration    string `xml:"duration"`
			Image       struct {
				Href string `xml:"href,attr"`
			} `xml:"image"`
			Enclosure struct {
				URL    string `xml:"url,attr"`
				Type   string `xml:"type,attr"`
				Length int64  `xml:"length,attr"`
			} `xml:"enclosure"`
		} `xml:"item"`
	} `xml:"channel"`
}

func parseRSS(body []byte, feedID string) (Feed, error) {
	var document rss
	if err := xml.Unmarshal(body, &document); err != nil {
		return Feed{}, err
	}
	title := strings.TrimSpace(document.Channel.Title)
	if title == "" {
		return Feed{}, errors.New("feed has no title")
	}
	art := document.Channel.Image.Href
	if art == "" {
		art = document.Channel.Image.URL
	}
	feed := Feed{Title: title, Description: podcastPlainText(document.Channel.Description), ArtworkURL: art}
	for _, item := range document.Channel.Items {
		if strings.TrimSpace(item.Enclosure.URL) == "" {
			continue
		}
		stable := strings.TrimSpace(item.GUID)
		if stable == "" {
			stable = item.Enclosure.URL
		}
		sum := sha256.Sum256([]byte(feedID + "\x00" + stable))
		episodeArt := item.Image.Href
		if episodeArt == "" {
			episodeArt = art
		}
		description := item.Description
		if strings.TrimSpace(description) == "" {
			description = item.Summary
		}
		if strings.TrimSpace(description) == "" {
			description = item.Content
		}
		feed.Episodes = append(feed.Episodes, Episode{
			ID: hex.EncodeToString(sum[:16]), FeedID: feedID, FeedTitle: title,
			Title: strings.TrimSpace(item.Title), Description: podcastPlainText(description),
			MediaURL: item.Enclosure.URL, Link: strings.TrimSpace(item.Link),
			Published: parsePodcastTime(item.PubDate), Duration: strings.TrimSpace(item.Duration),
			MediaType: item.Enclosure.Type, Size: item.Enclosure.Length, ArtworkURL: episodeArt,
		})
	}
	return feed, nil
}

func parsePodcastTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	if parsed, err := http.ParseTime(value); err == nil {
		return crediblePodcastTime(parsed)
	}
	for _, layout := range []string{time.RFC1123Z, time.RFC822, time.RFC822Z, time.RFC3339, "Mon, 2 Jan 2006 15:04:05 -0700", "2 Jan 2006 15:04:05 -0700"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return crediblePodcastTime(parsed)
		}
	}
	return time.Time{}
}

// crediblePodcastTime prevents syntactically valid placeholder dates (most
// notably year 1) from becoming misleading relative ages or affecting episode
// ordering. Unknown publication dates remain represented by time.Time{}.
func crediblePodcastTime(value time.Time) time.Time {
	if value.Year() < 1900 {
		return time.Time{}
	}
	return value
}

func podcastPlainText(value string) string {
	value = strings.NewReplacer("<br>", " ", "<br/>", " ", "<br />", " ", "</p>", " ", "</div>", " ").Replace(value)
	value = html.UnescapeString(podcastHTMLTag.ReplaceAllString(value, " "))
	value = strings.Join(strings.Fields(value), " ")
	value = strings.NewReplacer(" .", ".", " ,", ",", " ;", ";", " :", ":", " !", "!", " ?", "?").Replace(value)
	runes := []rune(value)
	if len(runes) > maximumDescriptionRunes {
		value = strings.TrimSpace(string(runes[:maximumDescriptionRunes])) + "…"
	}
	return value
}
