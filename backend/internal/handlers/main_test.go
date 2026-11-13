package handlers

import (
	"github.com/gin-gonic/gin"
	"os"
	"testing"
)

// Gin mode is global; set it once before parallel handler tests.
func TestMain(m *testing.M) { gin.SetMode(gin.TestMode); os.Exit(m.Run()) }
