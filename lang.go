package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"golang.org/x/text/language"
)

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

	return validateLanguage(lang)
}

func validateLanguage(lang string) string {
	//validate the language
	if isLanguageSupported(lang) {
		return lang
	} else {
		return "en"
	}
}

func isLanguageSupported(lang string) bool {
	//check if the language is supported
	for _, supportedLang := range getSupportedLanguages() {
		if lang == supportedLang {
			return true
		}
	}
	return false
}

func getSupportedLanguages() []string {
	//get all file names in static/lang and remove the .json extension
	files, err := os.ReadDir("static/lang")
	if err != nil {
		log.Fatal(err)
	}
	filesNames := make([]string, len(files))
	for i, file := range files {
		filesNames[i] = strings.TrimSuffix(file.Name(), ".json")
	}

	return filesNames
}
