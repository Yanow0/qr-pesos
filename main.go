package main

import (
	"fmt"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"text/template"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/skip2/go-qrcode"
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
	// Render the home page template
	return c.Render(http.StatusOK, "base.html", map[string]interface{}{
		"Title":                      "QR Code Generator",
		"Language":                   "en",
		"InputLabel":                 "Enter text to generate QR code",
		"GenerateButtonLabel":        "Generate",
		"QrCode":                     "",
		"CopyToClipboardButtonLabel": "Copy to Clipboard",
		"DownloadButtonLabel":        "Download",
	})
}

func generateHandler(c echo.Context) error {
	// Generate the QR code and render the template with the QR code image
	data := c.FormValue("data")
	qrCodeImage := generateQRCode(data)

	return c.Render(http.StatusOK, "base.html", map[string]interface{}{
		"Title":                      "QR Code Generator",
		"Language":                   "en",
		"InputLabel":                 "Enter text to generate QR code",
		"GenerateButtonLabel":        "Generate",
		"QrCode":                     qrCodeImage,
		"CopyToClipboardButtonLabel": "Copy to Clipboard",
		"DownloadButtonLabel":        "Download",
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
	filePath := path.Join("static", fileName)

	fmt.Println(filePath)

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
	return fmt.Sprintf("/static/%s", fileName)
}
