package hackernews

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

const baseURL = "https://hacker-news.firebaseio.com/v0"

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	httpClient HTTPClient
}

func NewClient(httpClient HTTPClient) *Client { return &Client{httpClient: httpClient} }

type item struct {
	ID    int    `json:"id"`
	By    string `json:"by"`
	Score int    `json:"score"`
	Time  int64  `json:"time"`
	Title string `json:"title"`
	URL   string `json:"url"`
	Type  string `json:"type"`
}

func (c *Client) Search(ctx context.Context, query string, limit int) ([]domain.BusinessStory, error) {
	if limit < 1 || limit > 8 {
		limit = 5
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/topstories.json", nil)
	if err != nil {
		return nil, fmt.Errorf("create top stories request: %w", err)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("load top stories: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("top stories returned %s", response.Status)
	}
	var ids []int
	if err := json.NewDecoder(response.Body).Decode(&ids); err != nil {
		return nil, fmt.Errorf("decode top stories: %w", err)
	}
	if len(ids) > 45 {
		ids = ids[:45]
	}

	items := make([]item, len(ids))
	valid := make([]bool, len(ids))
	semaphore := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for index, id := range ids {
		wg.Add(1)
		go func(index, id int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			loaded, loadErr := c.loadItem(ctx, id)
			if loadErr == nil && loaded.Type == "story" && loaded.Title != "" {
				items[index], valid[index] = loaded, true
			}
		}(index, id)
	}
	wg.Wait()

	terms := searchTerms(query)
	type ranked struct {
		item      item
		relevance int
	}
	rankedItems := make([]ranked, 0, len(items))
	for index, candidate := range items {
		if !valid[index] {
			continue
		}
		title := strings.ToLower(candidate.Title)
		relevance := 0
		for _, term := range terms {
			if strings.Contains(title, term) {
				relevance += 100
			}
		}
		rankedItems = append(rankedItems, ranked{item: candidate, relevance: relevance})
	}
	sort.SliceStable(rankedItems, func(i, j int) bool {
		if rankedItems[i].relevance != rankedItems[j].relevance {
			return rankedItems[i].relevance > rankedItems[j].relevance
		}
		return rankedItems[i].item.Score > rankedItems[j].item.Score
	})
	if len(rankedItems) > limit {
		rankedItems = rankedItems[:limit]
	}

	stories := make([]domain.BusinessStory, 0, len(rankedItems))
	for _, rankedItem := range rankedItems {
		candidate := rankedItem.item
		storyURL := candidate.URL
		if storyURL == "" {
			storyURL = fmt.Sprintf("https://news.ycombinator.com/item?id=%d", candidate.ID)
		}
		stories = append(stories, domain.BusinessStory{
			ID: fmt.Sprint(candidate.ID), Title: candidate.Title, URL: storyURL,
			DiscussionURL: fmt.Sprintf("https://news.ycombinator.com/item?id=%d", candidate.ID),
			Source:        "Hacker News", Author: candidate.By, Score: candidate.Score,
			PublishedAt: time.Unix(candidate.Time, 0).UTC(),
		})
	}
	return stories, nil
}

func (c *Client) loadItem(ctx context.Context, id int) (item, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/item/%d.json", baseURL, id), nil)
	if err != nil {
		return item{}, err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return item{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return item{}, fmt.Errorf("item returned %s", response.Status)
	}
	var result item
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return item{}, err
	}
	return result, nil
}

func searchTerms(query string) []string {
	ignored := map[string]bool{"найди": true, "новости": true, "новостей": true, "про": true, "для": true, "the": true, "and": true, "about": true, "business": true, "бизнес": true}
	terms := []string{}
	for _, raw := range strings.Fields(strings.ToLower(query)) {
		term := strings.Trim(raw, " ,.!?;:\"'()[]{}")
		if len([]rune(term)) >= 3 && !ignored[term] {
			terms = append(terms, term)
		}
	}
	if len(terms) == 0 {
		return []string{"startup", "ai", "launch", "funding"}
	}
	return terms
}

func Host(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return strings.TrimPrefix(parsed.Hostname(), "www.")
}
