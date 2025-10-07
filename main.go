package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/valyala/fasthttp"
)

const openRouterURL = "https://openrouter.ai/api/v1/chat/completions"

var db *pgx.Conn

type Flashcard struct {
	ID       string `json:"id"`
	Topic    string `json:"topic"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type OpenRouterRequest struct {
	Model    string          `json:"model"`
	Messages []OpenRouterMsg `json:"messages"`
}

type OpenRouterMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenRouterResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func main() {
	fmt.Println("🚀 Starting Flashcard Generator...")

	_ = godotenv.Load()
	dbURL := os.Getenv("SUPABASE_DB_URL")
	apiKey := os.Getenv("OPENROUTER_API_KEY")

	fmt.Printf("📊 Environment check - DB URL exists: %v, API Key exists: %v\n",
		dbURL != "", apiKey != "")

	if dbURL == "" || apiKey == "" {
		log.Fatal("❌ Set SUPABASE_DB_URL and OPENROUTER_API_KEY in .env")
	}

	var err error
	fmt.Println("🔌 Connecting to database...")
	db, err = pgx.Connect(context.Background(), dbURL)
	if err != nil {
		log.Fatal("❌ Unable to connect to DB: ", err)
	}
	defer db.Close(context.Background())
	fmt.Println("✅ Database connected successfully")

	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,OPTIONS",
		AllowHeaders: "*",
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		fmt.Println("📡 Health check endpoint hit")
		return c.JSON(fiber.Map{"status": "running", "database": "connected"})
	})

	app.Post("/user", func(c *fiber.Ctx) error {
		fmt.Println("👤 Creating/verifying user...")

		type Req struct {
			Username string `json:"username"`
		}
		var body Req
		if err := c.BodyParser(&body); err != nil || body.Username == "" {
			fmt.Printf("❌ Invalid user request: %v\n", err)
			return c.Status(400).JSON(fiber.Map{"error": "Username required"})
		}

		fmt.Printf("👤 Processing user: %s\n", body.Username)

		var id string
		err := db.QueryRow(context.Background(),
			"INSERT INTO users (username) VALUES ($1) ON CONFLICT(username) DO UPDATE SET username=EXCLUDED.username RETURNING id",
			body.Username).Scan(&id)
		if err != nil {
			fmt.Printf("❌ User creation error: %v\n", err)
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		fmt.Printf("✅ User created/found with ID: %s\n", id)
		return c.JSON(fiber.Map{"id": id, "username": body.Username})
	})

	// Generate flashcards with extensive logging
	app.Post("/flashcards", func(c *fiber.Ctx) error {
		fmt.Println("\n🎯 === FLASHCARD GENERATION REQUEST ===")

		type Req struct {
			Username string `json:"username"`
			Topic    string `json:"topic"`
			Count    int    `json:"count"`
			Level    string `json:"level"`
		}
		var body Req
		if err := c.BodyParser(&body); err != nil || body.Username == "" || body.Topic == "" {
			fmt.Printf("❌ Invalid flashcard request: %v\n", err)
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
		}
		if body.Count <= 0 {
			body.Count = 5
		}
		if body.Level == "" {
			body.Level = "beginner"
		}

		fmt.Printf("📝 Request details:\n")
		fmt.Printf("   Username: %s\n", body.Username)
		fmt.Printf("   Topic: %s\n", body.Topic)
		fmt.Printf("   Count: %d\n", body.Count)
		fmt.Printf("   Level: %s\n", body.Level)

		// Check if user exists
		fmt.Println("🔍 Looking up user...")
		var userID string
		err := db.QueryRow(context.Background(), "SELECT id FROM users WHERE username=$1", body.Username).Scan(&userID)
		if err != nil {
			fmt.Printf("❌ User lookup error: %v\n", err)
			return c.Status(400).JSON(fiber.Map{"error": "User not found"})
		}
		fmt.Printf("✅ Found user with ID: %s\n", userID)

		// Prepare AI prompt
		prompt := fmt.Sprintf(
			"Generate exactly %d high-quality flashcards for learning %s at %s level. Return ONLY a valid JSON array with this exact format: [{\"question\": \"...\", \"answer\": \"...\"}]. No other text.",
			body.Count, body.Topic, body.Level,
		)

		fmt.Printf("🤖 AI Prompt: %s\n", prompt)

		reqBody := OpenRouterRequest{
			Model: "deepseek/deepseek-chat-v3.1:free",
			Messages: []OpenRouterMsg{
				{Role: "system", Content: "You are a flashcard generator. Return only valid JSON array format with question and answer fields."},
				{Role: "user", Content: prompt},
			},
		}

		fmt.Println("📡 Making AI request...")
		b, _ := json.Marshal(reqBody)
		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)

		req.SetRequestURI(openRouterURL)
		req.Header.SetMethod("POST")
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.SetBody(b)

		resp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseResponse(resp)

		client := &fasthttp.Client{}
		if err := client.Do(req, resp); err != nil {
			fmt.Printf("❌ AI request failed: %v\n", err)
			return c.Status(500).JSON(fiber.Map{"error": "AI request failed"})
		}

		fmt.Printf("📡 AI Response Status: %d\n", resp.StatusCode())
		fmt.Printf("📡 AI Response Body Length: %d\n", len(resp.Body()))

		var aiResp OpenRouterResponse
		if err := json.Unmarshal(resp.Body(), &aiResp); err != nil {
			fmt.Printf("❌ AI response unmarshal error: %v\n", err)
			fmt.Printf("❌ Raw AI response: %s\n", string(resp.Body()))
			return c.Status(500).JSON(fiber.Map{"error": "Invalid AI response"})
		}

		if len(aiResp.Choices) == 0 {
			fmt.Println("❌ No choices in AI response")
			fmt.Printf("❌ Full AI response: %+v\n", aiResp)
			return c.Status(500).JSON(fiber.Map{"error": "No choices from AI"})
		}

		raw := aiResp.Choices[0].Message.Content
		fmt.Printf("🔍 Raw AI content length: %d\n", len(raw))
		fmt.Printf("🔍 Raw AI content (first 500 chars): %s...\n",
			raw[:min(len(raw), 500)])

		// Clean up the response
		raw = strings.TrimSpace(raw)

		// Remove markdown code blocks if present
		if strings.HasPrefix(raw, "```json") {
			raw = strings.TrimPrefix(raw, "```json")
			raw = strings.TrimSuffix(raw, "```")
			raw = strings.TrimSpace(raw)
			fmt.Println("🧹 Cleaned markdown formatting")
		}

		// Try to parse as JSON
		var cards []map[string]string
		if err := json.Unmarshal([]byte(raw), &cards); err != nil {
			fmt.Printf("❌ First JSON parse failed: %v\n", err)

			// Try wrapping in array brackets
			raw = "[" + raw + "]"
			if err := json.Unmarshal([]byte(raw), &cards); err != nil {
				fmt.Printf("❌ Second JSON parse failed: %v\n", err)
				fmt.Printf("❌ Final raw content: %s\n", raw)
				return c.Status(500).JSON(fiber.Map{"error": "Failed to parse flashcards"})
			}
			fmt.Println("✅ JSON parsed after wrapping in brackets")
		} else {
			fmt.Println("✅ JSON parsed successfully on first try")
		}

		fmt.Printf("📊 Parsed %d cards from AI response\n", len(cards))

		// Insert cards into database
		successCount := 0
		for i, card := range cards {
			question, qExists := card["question"]
			answer, aExists := card["answer"]

			if !qExists || !aExists {
				fmt.Printf("❌ Card %d missing question or answer fields\n", i+1)
				continue
			}

			fmt.Printf("💾 Inserting card %d: Q=%.50s... A=%.50s...\n",
				i+1, question, answer)

			_, err := db.Exec(context.Background(),
				"INSERT INTO flashcards (user_id, topic, question, answer) VALUES ($1, $2, $3, $4)",
				userID, body.Topic, question, answer)
			if err != nil {
				fmt.Printf("❌ Insert error for card %d: %v\n", i+1, err)
			} else {
				successCount++
			}
		}

		fmt.Printf("✅ Successfully inserted %d/%d cards into database\n", successCount, len(cards))
		fmt.Printf("=== FLASHCARD GENERATION COMPLETE ===\n")

		return c.JSON(fiber.Map{"flashcards": cards})
	})

	app.Get("/flashcards/:username", func(c *fiber.Ctx) error {
		username := c.Params("username")
		fmt.Printf("📖 Loading flashcards for user: %s\n", username)

		rows, err := db.Query(context.Background(),
			`SELECT f.id, f.topic, f.question, f.answer 
             FROM flashcards f
             JOIN users u ON f.user_id = u.id
             WHERE u.username=$1 ORDER BY f.created_at DESC`, username)
		if err != nil {
			fmt.Printf("❌ Query error: %v\n", err)
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		var result []Flashcard
		for rows.Next() {
			var f Flashcard
			if err := rows.Scan(&f.ID, &f.Topic, &f.Question, &f.Answer); err != nil {
				fmt.Printf("❌ Scan error: %v\n", err)
				continue
			}
			result = append(result, f)
		}

		fmt.Printf("✅ Loaded %d flashcards for user %s\n", len(result), username)
		return c.JSON(result)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}
	fmt.Printf("🚀 Server starting on port %s\n", port)
	log.Fatal(app.Listen("0.0.0.0:" + port))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
