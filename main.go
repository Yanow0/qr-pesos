package main

import (
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/skip2/go-qrcode"
	"golang.org/x/text/language"
)

func main() {
	cfg := LoadConfig()

	e := echo.New()

	// Set the templates renderer
	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseGlob("templates/*.html")),
	}
	e.Renderer = renderer

	// Define routes
	e.GET("/", homeHandler)
	e.POST("/generate", generateHandler)
	e.Static("/static", cfg.StaticFilesDir)

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
	// Load translations for the current language
	messages, err := loadMessages(getLanguage(c))

	if err != nil {
		return err
	}

	// Render the home page template with the translations
	return c.Render(http.StatusOK, "base.html", map[string]interface{}{
		"Title":                      messages["Title"],
		"InputLabel":                 messages["inputLabel"],
		"GenerateButtonLabel":        messages["generateButtonLabel"],
		"QrCode":                     "",
		"CopyToClipboardButtonLabel": messages["copyToClipboardButtonLabel"],
		"DownloadButtonLabel":        messages["downloadButtonLabel"],
		"Messages":                   messages, // Pass the translations as a separate data field
	})
}

func generateHandler(c echo.Context) error {
	// Load translations for the current language
	messages, err := loadMessages(getLanguage(c))
	if err != nil {
		return err
	}

	// Generate the QR code and render the template with the QR code image and translations
	data := c.FormValue("data")
	qrCodeImage := generateQRCode(data)

	return c.Render(http.StatusOK, "base.html", map[string]interface{}{
		"Title":                      messages["Title"],
		"InputLabel":                 messages["inputLabel"],
		"GenerateButtonLabel":        messages["generateButtonLabel"],
		"QrCode":                     qrCodeImage,
		"CopyToClipboardButtonLabel": messages["copyToClipboardButtonLabel"],
		"DownloadButtonLabel":        messages["downloadButtonLabel"],
		"Messages":                   messages, // Pass the translations as a separate data field
	})
}

func generateQRCode(data string) string {
	// Generate the QR code using go-qrcode library
	qrCode, err := qrcode.New(data, qrcode.Medium)
	if err != nil {
		log.Fatal(err)
	}

	// Encode the QR code as a PNG image
	qrCodeImage := qrCode.Image(256)

	// Save the QR code image to a file in the static directory
	fileName := fmt.Sprintf("%d.png", time.Now().UnixNano())
	filePath := path.Join("static/img", fileName)

	file, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	err = png.Encode(file, qrCodeImage)
	if err != nil {
		log.Fatal(err)
	}

	// Return the URL of the file
	return fmt.Sprintf("/static/img/%s", fileName)
}

func loadMessages(lang string) (map[string]string, error) {
	// Load messages from the JSON file for the specified language

	file, err := os.Open(fmt.Sprintf("static/lang/%s.json", lang))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var messages map[string]string
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&messages); err != nil {
		return nil, err
	}

	return messages, nil
}

func getLanguage(c echo.Context) string {
	// Get the user's language preference from the Accept-Language header
	acceptLanguage := c.Request().Header.Get("Accept-Language")
	if acceptLanguage == "" {
		return "en" // Default to English if no language preference is specified
	}

	// Parse the Accept-Language header to get the user's preferred language
	languages, _, err := language.ParseAcceptLanguage(acceptLanguage)

	if err != nil || len(languages) == 0 {
		return "en" // Default to English if parsing fails
	}

	//get the first 2 characters of language[0].String()
	lang := languages[0].String()[0:2]
	lang = strings.ToLower(lang)

	if (lang != "en") && (lang != "es") && (lang != "fr") && (lang != "de") {
		return "en"
	}

	// Return the language code for the user's preferred language
	return lang
}
