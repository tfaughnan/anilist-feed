package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

type GQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type UnixTime time.Time

func (ut *UnixTime) UnmarshalJSON(b []byte) error {
	var ts int64
	if err := json.Unmarshal(b, &ts); err != nil {
		return err
	}
	*ut = UnixTime(time.Unix(ts, 0))
	return nil
}

type Activity struct {
	CreatedAt UnixTime `json:"createdAt"`
	ID        int      `json:"id"`
	Progress  string   `json:"progress"`
	SiteURL   string   `json:"siteURL"`
	Status    string   `json:"status"`
	Media     struct {
		BannerImage string `json:"bannerImage"`
		CoverImage  struct {
			Medium string `json:"medium"`
		} `json:"coverImage"`
		SiteURL string `json:"siteURL"`
		Title   struct {
			English string `json:"english"`
			Romaji  string `json:"romaji"`
		} `json:"title"`
	} `json:"media"`
}

func main() {
	var username, output, feedURL, tagger string
	var timeout int
	flag.StringVar(&username, "n", "", "anilist username")
	flag.StringVar(&output, "o", "-", "output file")
	flag.StringVar(&feedURL, "u", "", `URL for feed's rel="self" link`)
	flag.StringVar(&tagger, "e", "example.com,1970-01-01", "Tag URI taggingEntity per RFC 4151")
	flag.IntVar(&timeout, "t", 30, "http request timeout in seconds")
	flag.Parse()

	if username == "" {
		fmt.Fprintf(os.Stderr, "must supply username with -n flag\n")
		os.Exit(1)
	}

	client := http.Client{Timeout: time.Duration(timeout) * time.Second}
	userID, err := fetchUserID(&client, username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetchUserID: %v\n", err)
		os.Exit(1)
	}

	activities, err := fetchActivities(&client, userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetchActivities: %v\n", err)
		os.Exit(1)
	}

	feed := mkFeed(tagger, feedURL, username, activities)

	var file = os.Stdout
	if output != "-" {
		var err error
		file, err = os.Create(output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		defer file.Close()
	}

	if err := feed.WriteXML(file); err != nil {
		fmt.Fprintf(os.Stderr, "feed.WriteXML: %v\n", err)
		os.Exit(1)
	}
}

func postJSON(client *http.Client, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, "https://graphql.anilist.co", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "anilist-feed (+https://github.com/tfaughnan/anilist-feed)")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf(`received status "%s"`, resp.Status)
		return nil, err
	}

	return resp, nil
}

func fetchUserID(client *http.Client, username string) (int, error) {
	const query = `
query UserID($name: String) {
	User(name: $name) {
		id
	}
}
`
	body, err := json.Marshal(GQLRequest{
		Query:     query,
		Variables: map[string]any{"name": username},
	})
	if err != nil {
		return 0, err
	}

	resp, err := postJSON(client, body)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			User struct {
				ID int `json:"id"`
			} `json:"User"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	if len(result.Errors) > 0 {
		return 0, fmt.Errorf("anilist: %s", result.Errors[0].Message)
	}

	return result.Data.User.ID, nil
}

func fetchActivities(client *http.Client, userID int) ([]Activity, error) {
	const query = `
query ListActivity($page: Int, $perPage: Int, $userId: Int, $sort: [ActivitySort], $type: ActivityType) {
  Page(page: $page, perPage: $perPage) {
    activities(userId: $userId, sort: $sort, type: $type) {
      ... on ListActivity {
        createdAt
        id
        progress
        siteUrl
        status
        media {
          bannerImage
          coverImage {
            medium
          }
          siteUrl
          title {
            english
            romaji
          }
        }
      }
    }
  }
}
`
	body, err := json.Marshal(GQLRequest{
		Query: query,
		Variables: map[string]any{
			"page":    1,  // TODO: do pagination?
			"perPage": 50, // TODO: cli flag for this? api max is 50
			"userId":  userID,
			"sort":    []string{"ID_DESC"},
			"type":    "ANIME_LIST", // TODO: manga? text posts?
		},
	})
	if err != nil {
		return nil, err
	}

	resp, err := postJSON(client, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Page struct {
				Activities []Activity `json:"activities"`
			} `json:"Page"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("anilist: %s", result.Errors[0].Message)
	}

	return result.Data.Page.Activities, nil
}

func mkFeed(tagger, feedURL, username string, activities []Activity) Feed {
	var entries []Entry
	for _, activity := range activities {
		var title string
		mediaTitle := activity.Media.Title.English
		if mediaTitle == "" {
			mediaTitle = activity.Media.Title.Romaji
		}
		if activity.Progress != "" {
			title = fmt.Sprintf("%s %s of %s", activity.Status, activity.Progress, mediaTitle)
		} else {
			title = fmt.Sprintf("%s %s", activity.Status, mediaTitle)
		}

		content := Content{
			Type: "html",
			Body: fmt.Sprintf(`<a href="%s"><img src="%s" alt="%s"></a>`, activity.Media.SiteURL, activity.Media.CoverImage.Medium, mediaTitle),
		}

		entry := Entry{
			Title:     title,
			ID:        TagURI(tagger, "anilist-feed:"+activity.SiteURL),
			Links:     []Link{{Rel: "alternate", Type: "text/html", Href: activity.SiteURL}},
			Published: AtomTime(activity.CreatedAt),
			Updated:   AtomTime(activity.CreatedAt),
			Content:   &content,
			// TODO: if we later support manga, may want to use Categories
		}
		entries = append(entries, entry)
	}

	links := []Link{{Rel: "alternate", Type: "text/html", Href: "https://anilist.co/user/" + username}}
	if feedURL != "" { // feeds SHOULD contain a self link, per rfc
		links = append(links, Link{Rel: "self", Type: "application/atom+xml", Href: feedURL})
	}
	feed := Feed{
		Generator: &Generator{
			Name: "anilist-feed",
			URI:  "https://github.com/tfaughnan/anilist-feed",
		},
		Title:   username + " on AniList",
		ID:      TagURI(tagger, "anilist-feed"),
		Links:   links,
		Updated: AtomTime(time.Now()),
		Entries: entries,
	}
	return feed
}
