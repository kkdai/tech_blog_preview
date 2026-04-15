package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"github.com/mmcdole/gofeed"
	"google.golang.org/api/option"
)

const MaxPosts = 5

type Article struct {
	Title          string
	Link           string
	Published      string
	Summary        string
	TranslationRec string
}

type FeedData struct {
	Language string
	Articles []Article
}

var geminiModel string
var geminiSem = make(chan struct{}, 2) // Limit concurrent Gemini API calls to 2

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, relying on system vars")
	}

	geminiModel = os.Getenv("GEMINI_MODEL")
	if geminiModel == "" {
		geminiModel = "gemini-1.5-flash"
	}

	http.HandleFunc("/", handleIndex)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	feeds := []struct {
		Lang string
		URL  string
	}{
		{"Korean", "https://techblog.lycorp.co.jp/ko/feed/index.xml"},
		{"Japanese", "https://techblog.lycorp.co.jp/ja/feed/index.xml"},
		{"English", "https://techblog.lycorp.co.jp/en/feed/index.xml"},
	}

	var wg sync.WaitGroup
	results := make([]FeedData, len(feeds))

	for i, f := range feeds {
		wg.Add(1)
		go func(idx int, lang, url string) {
			defer wg.Done()
			articles := fetchAndProcessFeed(lang, url)
			results[idx] = FeedData{Language: lang, Articles: articles}
		}(i, f.Lang, f.URL)
	}

	wg.Wait()

	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, "Could not parse template", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, results)
	if err != nil {
		log.Printf("Template execution error: %v", err)
	}
}

func fetchAndProcessFeed(lang, url string) []Article {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		log.Printf("Error fetching feed %s: %v", url, err)
		return nil
	}

	limit := MaxPosts
	if len(feed.Items) < limit {
		limit = len(feed.Items)
	}

	var wg sync.WaitGroup
	articles := make([]Article, limit)

	for i := 0; i < limit; i++ {
		item := feed.Items[i]
		articles[i] = Article{
			Title:     item.Title,
			Link:      item.Link,
			Published: item.Published,
		}

		wg.Add(1)
		go func(idx int, contentDesc string, title string) {
			defer wg.Done()
			transTitle, summary, rec := processWithGemini(title, contentDesc)
			if transTitle != "" && transTitle != title {
				articles[idx].Title = transTitle
			}
			articles[idx].Summary = summary
			articles[idx].TranslationRec = rec
		}(i, item.Description, item.Title)
	}

	wg.Wait()
	return articles
}

func processWithGemini(title string, content string) (string, string, string) {
	geminiSem <- struct{}{}
	defer func() { <-geminiSem }()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return title, "⚠️ 缺少 GEMINI_API_KEY", "無法評估"
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Printf("Error creating Gemini client: %v", err)
		return title, "客戶端建立失敗", "無法評估"
	}
	defer client.Close()

	model := client.GenerativeModel(geminiModel)

	if len(content) > 4000 {
		content = content[:4000]
	}

	prompt := fmt.Sprintf(`請針對以下文章的提供資訊進行處理：
標題：%s
部分內容/描述：%s

請以強制格式提供以下三項（必須包含 2 個 "=====" 作為分隔線，不要有多餘內容）：
[文章標題的繁體中文翻譯]
=====
[中文重點摘要（約50-150字內，包含核心技術或重點）]
=====
[評估結果：適合 / 不適合。（請簡接用一句話解釋為何，例如：適合，包含豐富工程細節；或 不適合，僅為短公告）]`, title, content)

	var resp *genai.GenerateContentResponse

	for retries := 0; retries < 3; retries++ {
		resp, err = model.GenerateContent(ctx, genai.Text(prompt))
		if err == nil {
			break
		}
		log.Printf("Retry %d generating content for '%s': %v", retries+1, title, err)
		time.Sleep(time.Duration(1+retries*2) * time.Second)
	}

	if err != nil {
		log.Printf("Error generating content for '%s': %v", title, err)
		return title, "API 頻率限制 (Rate Limit)，請稍後重試", "無法評估"
	}

	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		var output string
		for _, part := range resp.Candidates[0].Content.Parts {
			output += fmt.Sprintf("%v", part)
		}
		
		parts := strings.Split(output, "=====")
		if len(parts) >= 3 {
			return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2])
		} else if len(parts) >= 2 {
			return title, strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		}
		return title, strings.TrimSpace(output), "無法拆分結果"
	}

	return title, "沒有回傳結果", "無法評估"
}
