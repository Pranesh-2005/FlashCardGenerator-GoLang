# 🚀 FlashCardGenerator-GoLang

FlashCardGenerator-GoLang is a modern web application that empowers users to generate and manage flashcards efficiently using a Go backend and a sleek, PWA-enabled frontend. Designed for students, educators, and lifelong learners, this project leverages Go’s performance and a simple web interface to create, store, and review flashcards with ease.

---

## ✨ Features

- **Fast Go Backend:** Powered by [Fiber](https://gofiber.io/) for blazing-fast API responses.
- **Progressive Web App (PWA):** Installable on devices for offline access and a native app feel.
- **Modern UI:** Clean frontend with responsive layout and Google Fonts.
- **Database Integration:** Ready for PostgreSQL with [pgx](https://github.com/jackc/pgx) support.
- **API-Ready:** Easily extend or integrate with other services.
- **OpenAI/OpenRouter Integration:** (Planned) for AI-assisted flashcard generation.
- **Cross-Origin Support:** Seamless API usage from anywhere with CORS middleware.

---

## 🛠 Installation

### Prerequisites

- [Go 1.20+](https://go.dev/dl/) installed
- [Node.js](https://nodejs.org/) (for advanced frontend development, optional)
- [PostgreSQL](https://www.postgresql.org/) running (default config)

### Clone the Repository

```bash
git clone https://github.com/your-username/FlashCardGenerator-GoLang.git
cd FlashCardGenerator-GoLang
```

### Backend Setup

1. Copy `.env.example` to `.env` and update database credentials and API keys as needed.
2. Install Go dependencies:

   ```bash
   go mod download
   ```

3. Run the server:

   ```bash
   go run main.go
   ```

### Frontend Setup

The static frontend is located in `frontend/` and can be served as static files.

- To preview, simply open `frontend/index.html` in your browser.
- For PWA features, serve with any static file server (e.g., [serve](https://www.npmjs.com/package/serve)):

  ```bash
  npx serve frontend/
  ```

---

## 🚦 Usage

1. Start the Go backend:

   ```bash
   go run main.go
   ```

2. Open the frontend:

   - Directly: Open `frontend/index.html` in your browser (basic).
   - Or: Serve with a static file server for full PWA and API compatibility.

3. Create and manage your flashcards via the web interface!

---

## 🤝 Contributing

Contributions are welcome! To get started:

1. Fork the repository.
2. Create a new branch (`git checkout -b feature/my-feature`).
3. Make your changes and commit (`git commit -am 'Add new feature'`).
4. Push to your fork (`git push origin feature/my-feature`).
5. Open a Pull Request.

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).

---

> **Made with Go, Fiber, and 💡 for learners everywhere!**


## License
This project is licensed under the **MIT** License.

---
🔗 GitHub Repo: https://github.com/Pranesh-2005/FlashCardGenerator-GoLang