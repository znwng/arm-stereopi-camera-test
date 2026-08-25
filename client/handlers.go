package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func indexHandler(c *gin.Context) {
	page, err := os.ReadFile("page.html")

	if err != nil {
		c.String(
			http.StatusInternalServerError,
			"Failed to load page.html: %v",
			err,
		)

		return
	}

	c.Data(
		http.StatusOK,
		"text/html; charset=utf-8",
		page,
	)
}
