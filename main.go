package main

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(1)))
	e.GET("/hello", func(c echo.Context) error {
		res := struct {
			Message string
		}{
			Message: "Hello, World!!",
		}
		return c.JSON(http.StatusOK, res)
	})

	s := http.Server{
		Addr:    ":3000",
		Handler: e,
	}

	if err := s.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
