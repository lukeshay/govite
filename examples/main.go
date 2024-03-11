package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/lukeshay/govite/pkg/engine"
)

func main() {
	_, isDev := os.LookupEnv("DEV")
	app := fiber.New()

	appDir := os.Args[1]

	var eng engine.Engine

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	}))

	app.Use(logger.New())

	if isDev {
		log.Info("Starting development engine")

		eng = engine.MustNewDevelopmentEngine(engine.DevelopmentEngineOptions{
			AppDir:     appDir,
			Stdout:     os.Stdout,
			Stderr:     os.Stderr,
			Logger:     log,
			ServerPort: 3000,
		})
	} else {
		log.Info("Starting production engine")

		eng = engine.MustNewProductionEngine(engine.ProductionEngineOptions{
			DistDir: filepath.Join(appDir, "dist"),
			Stdout:  os.Stdout,
			Stderr:  os.Stderr,
			Logger:  log,
		})
	}
	defer eng.Close()

	app.Get("/", func(c *fiber.Ctx) error {
		result, err := eng.Render(c.Path(), map[string]string{
			"path": c.Path(),
			"time": time.Now().Format(time.RFC3339),
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error rendering: %s\n", err.Error())

			return c.SendStatus(500)
		}

		c.Set("Content-Type", result.ContentType)

		return c.Send([]byte(result.Content))
	})

	if isDev {
		app.Get("*", func(c *fiber.Ctx) error {
			log.Info("Rendering", "path", c.Path())

			result, err := eng.Render(c.Path(), map[string]string{
				"path": c.Path(),
				"time": time.Now().Format(time.RFC3339),
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error rendering: %s\n", err.Error())

				return c.SendStatus(500)
			}

			for k, v := range result.Headers {
				for _, vv := range v {
					c.Append(k, vv)
				}
			}

			c.Set("Content-Type", result.ContentType)

			return c.Send([]byte(result.Content))
		})
	} else {
		app.Static("/", eng.StaticPath(), fiber.Static{
			Browse: true,
		})
	}

	app.Listen(":3000")
}
