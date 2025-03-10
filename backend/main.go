package main

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
)

type FileStore struct {
	files map[string]string
	mutex sync.Mutex
}

var store = FileStore{files: make(map[string]string)}

func generateToken() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

func uploadFile(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Fehler beim Datei-Upload"})
		return
	}
	defer file.Close()

	token := generateToken()
	filename := token + filepath.Ext(header.Filename)
	filepath := "storage/" + filename

	out, err := os.Create(filepath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Fehler beim Speichern der Datei"})
		return
	}
	defer out.Close()

	io.Copy(out, file)

	store.mutex.Lock()
	store.files[token] = filepath
	store.mutex.Unlock()

	c.JSON(http.StatusOK, gin.H{"link": "/download/" + token})
}

func downloadFile(c *gin.Context) {
	token := c.Param("token")

	store.mutex.Lock()
	filepath, exists := store.files[token]
	if exists {
		delete(store.files, token)
	}
	store.mutex.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Datei nicht gefunden oder bereits heruntergeladen"})
		return
	}

	c.File(filepath)
	os.Remove(filepath)
}

func main() {
	os.MkdirAll("storage", os.ModePerm)
	
	r := gin.Default()
	r.Static("/", "./frontend")
  r.POST("/upload", uploadFile)
	r.GET("/download/:token", downloadFile)

	r.Run(":8080")
}

