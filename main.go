package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"encoding/json"

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
	_ = godotenv.Load()
	dbURL := os.Getenv("SUPABASE_DB_URL")
	apiKey := os.Getenv("OPENROUTER_API_KEY")

	if dbURL == "" || apiKey == "" {
		log.Fatal("Set SUPABASE_DB_URL and OPENROUTER_API_KEY in .env")
	}

	var err error
	db, err = pgx.Connect(context.Background(), dbURL)
	if err != nil {
		log.Fatal("Unable to connect to DB: ", err)
	}
	defer db.Close(context.Background())

	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,OPTIONS",
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "running"})
	})

	app.Post("/user", func(c *fiber.Ctx) error {
		type Req struct{ Username string }
		var body Req
		if err := c.BodyParser(&body); err != nil || body.Username == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Username required"})
		}
		var id string
		err := db.QueryRow(context.Background(),
			"insert into users (username) values ($1) on conflict(username) do update set username=excluded.username returning id",
			body.Username).Scan(&id)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"id": id, "username": body.Username})
	})

	// Generate flashcards with count & level
	app.Post("/flashcards", func(c *fiber.Ctx) error {
		type Req struct {
			Username string `json:"username"`
			Topic    string `json:"topic"`
			Count    int    `json:"count"`
			Level    string `json:"level"`
		}
		var body Req
		if err := c.BodyParser(&body); err != nil || body.Username == "" || body.Topic == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
		}
		if body.Count <= 0 {
			body.Count = 5
		}
		if body.Level == "" {
			body.Level = "beginner"
		}

		var userID string
		err := db.QueryRow(context.Background(), "select id from users where username=$1", body.Username).Scan(&userID)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "User not found"})
		}

		prompt := fmt.Sprintf(
			"Generate %d high-quality unique flashcards for learning %s at %s level. Make sure all Questions are Answers are Top notch and add values while its been reviewed also make sure its just awesome its should explain like a research paper knowladge in a small flashcard. Format JSON list [{question, answer}].",
			body.Count, body.Topic, body.Level,
		)

		reqBody := OpenRouterRequest{
			Model: "x-ai/grok-4-fast:free",
			Messages: []OpenRouterMsg{
				{Role: "system", Content: "You are a helpful flashcard generator that produces unique and educational question-answer pairs. Questions should be clear and concise and it should be a standard one that will help in learning the topic both in breadth and depth also theoretical and practical plus some math if applicable to shine or be different from others. Think and give questions that are not commonly found in other flashcards but more useful and gains more impression and knowledge. The knowladge should be from research papers. Answers should be accurate, informative, and easy to understand"},
				{Role: "user", Content: prompt},
			},
		}

		b, _ := json.Marshal(reqBody)
		req := fasthttp.AcquireRequest()
		req.SetRequestURI(openRouterURL)
		req.Header.SetMethod("POST")
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.SetBody(b)

		resp := fasthttp.AcquireResponse()
		client := &fasthttp.Client{}
		if err := client.Do(req, resp); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "AI request failed"})
		}

		var aiResp OpenRouterResponse
		if err := json.Unmarshal(resp.Body(), &aiResp); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Invalid AI response"})
		}
		if len(aiResp.Choices) == 0 {
			return c.Status(500).JSON(fiber.Map{"error": "No choices from AI"})
		}

		raw := aiResp.Choices[0].Message.Content
		var cards []map[string]string
		if err := json.Unmarshal([]byte(raw), &cards); err != nil {
			raw = "[" + raw + "]"
			_ = json.Unmarshal([]byte(raw), &cards)
		}

		for _, card := range cards {
			_, err := db.Exec(context.Background(),
				"insert into flashcards (user_id, topic, question, answer) values ($1, $2, $3, $4)",
				userID, body.Topic, card["question"], card["answer"])
			if err != nil {
				log.Println("Insert error:", err)
			}
		}

		return c.JSON(fiber.Map{"flashcards": cards})
	})

	app.Get("/flashcards/:username", func(c *fiber.Ctx) error {
		username := c.Params("username")
		rows, err := db.Query(context.Background(),
			`select f.id, f.topic, f.question, f.answer 
			 from flashcards f
			 join users u on f.user_id = u.id
			 where u.username=$1 order by f.created_at desc`, username)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		var result []Flashcard
		for rows.Next() {
			var f Flashcard
			rows.Scan(&f.ID, &f.Topic, &f.Question, &f.Answer)
			result = append(result, f)
		}
		return c.JSON(result)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}
	log.Println("Running on :" + port)
	log.Fatal(app.Listen("0.0.0.0:" + port))
}
