package main

import (
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
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
	e.Use(session.Middleware(sessions.NewCookieStore([]byte("secret"))))

	// Set the templates renderer
	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseGlob("templates/*.html")),
	}
	e.Renderer = renderer

	// Define routes
	e.GET("/sitemap", sitemapHandler)
	e.GET("/:lang", homeHandler)
	e.GET("/", homeHandlerBase)
	e.POST("/generate", generateHandler)
	e.POST("/selectlang", selectLangHandler)
	e.GET("/:lang/about", aboutHandler)
	e.GET("/:lang/faq", faqHandler)
	e.GET("/:lang/terms", termsHandler)
	e.GET("/:lang/privacy", privacyHandler)
	e.GET("/:lang/contact", contactHandler)
	e.Static("/static", cfg.StaticFilesDir)

	// goroutine to delete old files every minutes in the static/img folder if they are older than 1 hour
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			files, err := os.ReadDir("static/qrcode")
			if err != nil {
				log.Err(err).Msg("Error reading directory")
			}
			for _, file := range files {
				// skip directories
				if file.IsDir() {
					continue
				}

				// get file path
				filePath := path.Join("static/qrcode", file.Name())
				// get last modified time
				fileStats, err := os.Stat(filePath)
				if err != nil {
					fmt.Println(err)
				}

				// delete file if older than 2 minutes
				if time.Since(fileStats.ModTime()) > 2*time.Minute && file.Name() != ".gitkeep" {
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

func sitemapHandler(c echo.Context) error {
	// return sitemap.xml file using os package
	sitemap, err := os.ReadFile("static/sitemap.xml")
	if err != nil {
		log.Err(err).Msg("Error reading sitemap.xml file")
	}
	return c.XMLBlob(http.StatusOK, sitemap)
}

func homeHandler(c echo.Context) error {
	sess, err := session.Get("session", c)

	if err != nil {
		log.Err(err).Msg("Error getting session")
	}

	if sess.Values["lang"] == nil || sess.Values["lang"] == "" {
		sess.Values["lang"] = c.Param("lang")
	}

	lang := sess.Values["lang"].(string)

	if !isLanguageSupported(lang) {
		//redirect to the language specific home page
		log.Err(err).Msg("Language not supported: " + lang)
		return c.Redirect(http.StatusMovedPermanently, "/"+getLanguage(c))
	}

	// Load translations for the current language
	messages, err := loadMessages(lang)

	if err != nil {
		log.Err(err).Msg("Error loading messages")
	}

	// Render the home page template with the translations
	return c.Render(http.StatusOK, "base.html", map[string]interface{}{
		"HomeLinkLabel":              messages["homeLinkLabel"],
		"AboutLinkLabel":             messages["aboutLinkLabel"],
		"FaqLinkLabel":               messages["faqLinkLabel"],
		"TermsLinkLabel":             messages["termsLinkLabel"],
		"PrivacyLinkLabel":           messages["privacyLinkLabel"],
		"ContactLinkLabel":           messages["contactLinkLabel"],
		"Title":                      messages["title"],
		"InputLabel":                 messages["inputLabel"],
		"GenerateButtonLabel":        messages["generateButtonLabel"],
		"QrCode":                     "",
		"CopyToClipboardButtonLabel": messages["copyToClipboardButtonLabel"],
		"DownloadButtonLabel":        messages["downloadButtonLabel"],
		"Messages":                   messages, // Pass the translations as a separate data field
		"Description":                messages["description"],
		"Lang":                       lang,
		"SelectLangs":                getSupportedLanguages(),
		"Tpl":                        "generate",
	})
}

func homeHandlerBase(c echo.Context) error {
	//redirect to the language specific home page
	return c.Redirect(http.StatusMovedPermanently, "/"+getLanguage(c))
}

func generateHandler(c echo.Context) error {

	sess, err := session.Get("session", c)

	if err != nil {
		log.Err(err).Msg("Error getting session")
	}

	if sess.Values["lang"] == nil {
		sess.Values["lang"] = c.FormValue("lang")
	}

	lang := sess.Values["lang"].(string)

	data := c.FormValue("data")

	if data == "" {
		return c.Redirect(http.StatusMovedPermanently, "/"+lang)
	}

	// Load translations for the current language
	messages, err := loadMessages(lang)
	if err != nil {
		log.Err(err).Msg("Error loading messages")
	}

	// Generate the QR code and render the template with the QR code image and translations

	qrCodeImage := generateQRCode(data)

	return c.Render(http.StatusOK, "base.html", map[string]interface{}{
		"HomeLinkLabel":              messages["homeLinkLabel"],
		"AboutLinkLabel":             messages["aboutLinkLabel"],
		"FaqLinkLabel":               messages["faqLinkLabel"],
		"TermsLinkLabel":             messages["termsLinkLabel"],
		"PrivacyLinkLabel":           messages["privacyLinkLabel"],
		"ContactLinkLabel":           messages["contactLinkLabel"],
		"Title":                      messages["title"],
		"InputLabel":                 messages["inputLabel"],
		"GenerateButtonLabel":        messages["generateButtonLabel"],
		"QrCode":                     qrCodeImage,
		"CopyToClipboardButtonLabel": messages["copyToClipboardButtonLabel"],
		"DownloadButtonLabel":        messages["downloadButtonLabel"],
		"Messages":                   messages, // Pass the translations as a separate data field
		"Description":                messages["description"],
		"Lang":                       lang,
		"SelectLangs":                getSupportedLanguages(),
		"Tpl":                        "generate",
	})
}

func aboutHandler(c echo.Context) error {
	sess, err := session.Get("session", c)

	if err != nil {
		log.Err(err).Msg("Error getting session")
	}

	if sess.Values["lang"] == nil {
		sess.Values["lang"] = c.Param("lang")
	}

	lang := sess.Values["lang"].(string)

	if !isLanguageSupported(lang) {
		//redirect to the language specific home page
		return c.Redirect(http.StatusMovedPermanently, "/"+getLanguage(c)+"/about")
	}

	// Load translations for the current language
	messages, err := loadMessages(lang)

	if err != nil {
		log.Err(err).Msg("Error loading messages")
	}

	// Render the about page template
	return c.Render(http.StatusOK, "base.html", map[string]interface{}{
		"HomeLinkLabel":    messages["homeLinkLabel"],
		"AboutLinkLabel":   messages["aboutLinkLabel"],
		"FaqLinkLabel":     messages["faqLinkLabel"],
		"TermsLinkLabel":   messages["termsLinkLabel"],
		"PrivacyLinkLabel": messages["privacyLinkLabel"],
		"ContactLinkLabel": messages["contactLinkLabel"],
		"Title":            messages["title"],
		"AboutTitle":       messages["aboutTitle"],
		"AboutText1":       messages["aboutText1"],
		"AboutText2":       messages["aboutText2"],
		"Lang":             lang,
		"SelectLangs":      getSupportedLanguages(),
		"Tpl":              "about",
	})
}

func faqHandler(c echo.Context) error {
	sess, err := session.Get("session", c)

	if err != nil {
		log.Err(err).Msg("Error getting session")
	}

	if sess.Values["lang"] == nil {
		sess.Values["lang"] = c.Param("lang")
	}

	lang := sess.Values["lang"].(string)

	if !isLanguageSupported(lang) {
		//redirect to the language specific home page
		return c.Redirect(http.StatusMovedPermanently, "/"+getLanguage(c)+"/faq")
	}

	// Load translations for the current language
	messages, err := loadMessages(lang)

	if err != nil {
		log.Err(err).Msg("Error loading messages")
	}

	// Render the FAQ page template
	return c.Render(http.StatusOK, "base.html", map[string]interface{}{
		"HomeLinkLabel":    messages["homeLinkLabel"],
		"AboutLinkLabel":   messages["aboutLinkLabel"],
		"FaqLinkLabel":     messages["faqLinkLabel"],
		"TermsLinkLabel":   messages["termsLinkLabel"],
		"PrivacyLinkLabel": messages["privacyLinkLabel"],
		"ContactLinkLabel": messages["contactLinkLabel"],
		"Title":            messages["title"],
		"FaqTitle":         messages["faqTitle"],
		"FaqText1_1":       messages["faqText1_1"],
		"FaqText1_2":       messages["faqText1_2"],
		"FaqText2_1":       messages["faqText2_1"],
		"FaqText2_2":       messages["faqText2_2"],
		"FaqText3_1":       messages["faqText3_1"],
		"FaqText3_2":       messages["faqText3_2"],
		"Lang":             lang,
		"SelectLangs":      getSupportedLanguages(),
		"Tpl":              "faq",
	})
}

func termsHandler(c echo.Context) error {
	sess, err := session.Get("session", c)

	if err != nil {
		log.Err(err).Msg("Error getting session")
	}

	if sess.Values["lang"] == nil {
		sess.Values["lang"] = c.Param("lang")
	}

	lang := sess.Values["lang"].(string)

	if !isLanguageSupported(lang) {
		//redirect to the language specific home page
		return c.Redirect(http.StatusMovedPermanently, "/"+getLanguage(c)+"/terms")
	}

	// Load translations for the current language
	messages, err := loadMessages(lang)

	if err != nil {
		log.Err(err).Msg("Error loading messages")
	}

	// Render the terms page template
	return c.Render(http.StatusOK, "base.html", map[string]interface{}{
		"HomeLinkLabel":    messages["homeLinkLabel"],
		"AboutLinkLabel":   messages["aboutLinkLabel"],
		"FaqLinkLabel":     messages["faqLinkLabel"],
		"TermsLinkLabel":   messages["termsLinkLabel"],
		"PrivacyLinkLabel": messages["privacyLinkLabel"],
		"ContactLinkLabel": messages["contactLinkLabel"],
		"Title":            messages["title"],
		"TermsTitle":       messages["termsTitle"],
		"TermsText1":       messages["termsText1"],
		"TermsText2":       messages["termsText2"],
		"TermsText3":       messages["termsText3"],
		"TermsText4":       messages["termsText4"],
		"TermsText5":       messages["termsText5"],
		"TermsText6":       messages["termsText6"],
		"Lang":             lang,
		"SelectLangs":      getSupportedLanguages(),
		"Tpl":              "terms",
	})
}

func privacyHandler(c echo.Context) error {
	sess, err := session.Get("session", c)

	if err != nil {
		log.Err(err).Msg("Error getting session")
	}

	if sess.Values["lang"] == nil {
		sess.Values["lang"] = c.Param("lang")
	}

	lang := sess.Values["lang"].(string)

	if !isLanguageSupported(lang) {
		//redirect to the language specific home page
		return c.Redirect(http.StatusMovedPermanently, "/"+getLanguage(c)+"/privacy")
	}

	// Load translations for the current language
	messages, err := loadMessages(lang)

	if err != nil {
		log.Err(err).Msg("Error loading messages")
	}

	// Render the privacy page template
	return c.Render(http.StatusOK, "base.html", map[string]interface{}{
		"HomeLinkLabel":             messages["homeLinkLabel"],
		"AboutLinkLabel":            messages["aboutLinkLabel"],
		"FaqLinkLabel":              messages["faqLinkLabel"],
		"TermsLinkLabel":            messages["termsLinkLabel"],
		"PrivacyLinkLabel":          messages["privacyLinkLabel"],
		"ContactLinkLabel":          messages["contactLinkLabel"],
		"Title":                     messages["title"],
		"PrivacyTitle":              messages["privacyTitle"],
		"PrivacyText1":              messages["privacyText1"],
		"PrivacyDataCollection":     messages["privacyDataCollection"],
		"PrivacyDataCollectionText": messages["privacyDataCollectionText"],
		"PrivacyDataUsage":          messages["privacyDataUsage"],
		"PrivacyDataUsageText":      messages["privacyDataUsageText"],
		"PrivacyCookieUsage":        messages["privacyCookieUsage"],
		"PrivacyCookieUsageText":    messages["privacyCookieUsageText"],
		"Lang":                      lang,
		"SelectLangs":               getSupportedLanguages(),
		"Tpl":                       "privacy",
	})
}

func contactHandler(c echo.Context) error {
	sess, err := session.Get("session", c)

	if err != nil {
		log.Err(err).Msg("Error getting session")
	}

	if sess.Values["lang"] == nil {
		sess.Values["lang"] = c.Param("lang")
	}

	lang := sess.Values["lang"].(string)

	if !isLanguageSupported(lang) {
		//redirect to the language specific home page
		return c.Redirect(http.StatusMovedPermanently, "/"+getLanguage(c)+"/contact")
	}

	// Load translations for the current language
	messages, err := loadMessages(lang)

	if err != nil {
		log.Err(err).Msg("Error loading messages")
	}

	// Render the contact page template
	return c.Render(http.StatusOK, "base.html", map[string]interface{}{
		"HomeLinkLabel":    messages["homeLinkLabel"],
		"AboutLinkLabel":   messages["aboutLinkLabel"],
		"FaqLinkLabel":     messages["faqLinkLabel"],
		"TermsLinkLabel":   messages["termsLinkLabel"],
		"PrivacyLinkLabel": messages["privacyLinkLabel"],
		"ContactLinkLabel": messages["contactLinkLabel"],
		"Title":            messages["title"],
		"ContactTitle":     messages["contactTitle"],
		"ContactText":      messages["contactText"],
		"ContactEmail":     messages["contactEmail"],
		"Lang":             lang,
		"SelectLangs":      getSupportedLanguages(),
		"Tpl":              "contact",
	})
}

func selectLangHandler(c echo.Context) error {
	sess, err := session.Get("session", c)

	if err != nil {
		log.Err(err).Msg("Error getting session")
	}

	sess.Values["lang"] = c.FormValue("lang")

	err = sess.Save(c.Request(), c.Response())

	if err != nil {
		log.Err(err).Msg("Error saving session")
	}

	// refresh the same page we came from with the new language
	url := c.Request().Referer()

	// replace the language in the URL with these patterns referrer/lang, and referrer/lang/? with the new language
	langs := getSupportedLanguages()

	for _, lang := range langs {
		url = strings.Replace(url, "/"+lang, "/"+c.FormValue("lang"), -1)
	}

	return c.Redirect(http.StatusMovedPermanently, url)
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
	filePath := path.Join("static/qrcode", fileName)

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
	return fmt.Sprintf("/static/qrcode/%s", fileName)
}
