package main

import (
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"os"
	"path"
	"text/template"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/skip2/go-qrcode"
)

func main() {
	// Zerolog config
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load config
	cfg := LoadConfig()

	e := echo.New()

	// Set the templates renderer
	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseGlob("templates/*.html")),
	}
	e.Renderer = renderer

	// Define routes
	e.GET("/:lang", homeHandler)
	e.GET("/", homeHandlerBase)
	e.POST("/generate", generateHandler)
	e.Static("/static", cfg.StaticFilesDir)

	// goroutine to delete old files every 30 minutes in the static/img folder if they are older than 1 hour
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			files, err := os.ReadDir("static/img")
			if err != nil {
				log.Err(err).Msg("Error reading directory")
			}
			for _, file := range files {
				// skip directories
				if file.IsDir() {
					continue
				}

				// get file path
				filePath := path.Join("static/img", file.Name())
				// get last modified time
				fileStats, err := os.Stat(filePath)
				if err != nil {
					fmt.Println(err)
				}

				// delete file if older than 1 hour
				if time.Since(fileStats.ModTime()) > 1*time.Hour {
					err := os.Remove(filePath)
					if err != nil {
						log.Err(err).Msg("Error deleting file")
					}
					log.Info().Msg("Deleted file " + file.Name())
				}
			}
		}
	}()

	// Print routes
	data, err := json.MarshalIndent(e.Routes(), "", "  ")
	if err != nil {
		log.Err(err).Msg("Error marshalling routes")
	}

	err = os.WriteFile("routes.json", data, 0777)
	if err != nil {
		log.Err(err).Msg("Error writing routes to file")
	}

	// Start server
	e.Logger.Fatal(e.Start(":" + cfg.Port))
}

// TemplateRenderer is a custom html/template renderer for Echo framework
type TemplateRenderer struct {
	templates *template.Template
}

// Render renders a template document
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {

	// Add global methods if data is a map
	if viewContext, isMap := data.(map[string]interface{}); isMap {
		viewContext["reverse"] = c.Echo().Reverse
	}

	return t.templates.ExecuteTemplate(w, name, data)
}

func homeHandler(c echo.Context) error {
	lang := c.Param("lang")

	if lang == "sitemap.xml" {
		return c.File("sitemap.xml")
	}

	if !isLanguageSupported(lang) {
		//redirect to the language specific home page
		return c.Redirect(http.StatusMovedPermanently, "/"+getLanguage(c))
	}

	// Load translations for the current language
	messages, err := loadMessages(lang)

	if err != nil {
		return err
	}

	// Render the home page template with the translations
	return c.Render(http.StatusOK, "base.html", map[string]interface{}{
		"Title":                      messages["title"],
		"InputLabel":                 messages["inputLabel"],
		"GenerateButtonLabel":        messages["generateButtonLabel"],
		"QrCode":                     "",
		"CopyToClipboardButtonLabel": messages["copyToClipboardButtonLabel"],
		"DownloadButtonLabel":        messages["downloadButtonLabel"],
		"Messages":                   messages, // Pass the translations as a separate data field
		"Description":                messages["description"],
		"Lang":                       lang,
	})
}

func homeHandlerBase(c echo.Context) error {
	//redirect to the language specific home page
	return c.Redirect(http.StatusMovedPermanently, "/"+getLanguage(c))
}

func generateHandler(c echo.Context) error {
	lang := c.FormValue("lang")
	data := c.FormValue("data")

	if data == "" {
		return c.Redirect(http.StatusMovedPermanently, "/"+getLanguage(c))
	}

	// Load translations for the current language
	messages, err := loadMessages(lang)
	if err != nil {
		return err
	}

	// Generate the QR code and render the template with the QR code image and translations

	qrCodeImage := generateQRCode(data)

	return c.Render(http.StatusOK, "base.html", map[string]interface{}{
		"Title":                      messages["title"],
		"InputLabel":                 messages["inputLabel"],
		"GenerateButtonLabel":        messages["generateButtonLabel"],
		"QrCode":                     qrCodeImage,
		"CopyToClipboardButtonLabel": messages["copyToClipboardButtonLabel"],
		"DownloadButtonLabel":        messages["downloadButtonLabel"],
		"Messages":                   messages, // Pass the translations as a separate data field
		"Description":                messages["description"],
		"Lang":                       lang,
	})
}

func generateQRCode(data string) string {
	// Generate the QR code using go-qrcode library
	qrCode, err := qrcode.New(data, qrcode.Medium)
	if err != nil {
		log.Err(err).Msg("Error generating QR code")
	}

	// Encode the QR code as a PNG image
	qrCodeImage := qrCode.Image(256)

	// Save the QR code image to a file in the static directory
	fileName := fmt.Sprintf("%d.png", time.Now().UnixNano())
	filePath := path.Join("static/img", fileName)

	file, err := os.Create(filePath)
	if err != nil {
		log.Err(err).Msg("Error creating file")
	}
	defer file.Close()

	err = png.Encode(file, qrCodeImage)
	if err != nil {
		log.Err(err).Msg("Error encoding QR code image")
	}

	// Return the URL of the file
	return fmt.Sprintf("/static/img/%s", fileName)
}
