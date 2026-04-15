# Lycorp Tech Blog Preview

Lycorp Tech Blog Preview 是一個基於 Go 語言打造的輕量級文章聚合器。本專案會自動取得 Lycorp 官方技術部落格（涵蓋韓國、日本、英文）最新的 RSS 文章，並運用 Google Gemini API 自動化進行以下處理：

- **標題自動翻譯**：將各國語言的文章標題翻譯為繁體中文。
- **重點摘要擷取**：透過分析文章內文或說明，產生重點式的中文摘要，快速掌握核心技術與內涵。
- **翻譯價值評估**：由 AI 簡單判斷這篇工程文章是否包含夠豐富的技術細節，給出一句話的「適合 / 不適合」翻譯建議。

## 🛠 功能與技術堆疊

- **後端 (Backend)**：`Go` (Golang)
- **前端 (Frontend)**：HTML / CSS Template (無框架)，使用現代美觀介面，支援響應式設計 (RWD)。
- **AI 解析 (AI Processing)**：`github.com/google/generative-ai-go/genai` (支援 Gemini 2.5/1.5 等各式版本 API)。
- **防限流機制 (Rate Limit Resiliency)**：實作 Go Channel Semaphore 限制最大併發請求數 (Concurrency Limit = 2)，加上 Exponential Backoff 機制自動進行錯誤退避重試，確保 API 呼叫高穩定性。

## 🚀 本地端開發 (Local Setup)

### 1. 環境需求
- [Go 1.22+](https://go.dev/dl/) 
- 一把有效的 [Google Gemini API Key](https://aistudio.google.com/)

### 2. 環境變數設定
專案使用了 `godotenv` 套件。請在專案根目錄建立一份 `.env` 檔案，或照著 `.env.example` 填寫：
```env
PORT=8080
GEMINI_API_KEY=您的_GEMINI_API_KEY_填在這裡
GEMINI_MODEL=gemini-2.5-flash-lite
```

### 3. 安裝與執行
```bash
# 下載與同步依賴套件
go mod tidy

# 編譯伺服器
go build -o server .

# 啟動伺服器
./server
```
伺服器啟動後將在 `http://localhost:8080` 提供服務。

## ☁️ 佈署至 Google Cloud Run (Deployment)

本專案可以直接利用內建的 `Dockerfile` 在 Google Cloud 平台上容器化執行。若要在 Cloud Run 上發佈，請利用您的 `gcloud CLI` 執行以下指令：

```bash
gcloud run deploy tech-blog-preview \
  --source . \
  --region asia-east1 \
  --allow-unauthenticated \
  --set-env-vars="GEMINI_API_KEY=YOUR_API_KEY,GEMINI_MODEL=gemini-2.5-flash-lite,PORT=8080"
```
這行指令不僅會將應用程式自動建置與佈署，也會一併將重要的環境變數寫入服務的 Secret/Env 內，不會有外洩的風險。
