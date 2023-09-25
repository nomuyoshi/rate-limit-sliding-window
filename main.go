package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()
	e.GET("/hello", func(c echo.Context) error {
		res := struct {
			Message string
		}{
			Message: "Hello, World!!",
		}
		return c.JSON(http.StatusOK, res)
	})

	e.Logger.Fatal(e.Start(":3000"))
}
