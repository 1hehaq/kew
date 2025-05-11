package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var blacklist = map[string]bool{
	"await": true, "break": true, "case": true, "catch": true, "class": true,
	"const": true, "continue": true, "debugger": true, "default": true, "delete": true,
	"do": true, "else": true, "enum": true, "export": true, "extends": true,
	"false": true, "finally": true, "for": true, "function": true, "if": true,
	"implements": true, "import": true, "in": true, "instanceof": true, "interface": true,
	"let": true, "new": true, "null": true, "package": true, "private": true,
	"protected": true, "public": true, "return": true, "super": true, "switch": true,
	"static": true, "this": true, "throw": true, "try": true, "true": true,
	"typeof": true, "var": true, "void": true, "while": true, "with": true,
	"abstract": true, "boolean": true, "byte": true, "char": true, "double": true,
	"final": true, "float": true, "goto": true, "int": true, "long": true, 
	"native": true, "short": true, "synchronized": true, "throws": true, "transient": true,
	"volatile": true,
	"alert": true, "frames": true, "outerheight": true, "all": true, "framerate": true,
	"outerwidth": true, "anchor": true, "packages": true,
	"anchors": true, "getclass": true, "pagexoffset": true, "area": true,
	"hasownproperty": true, "pageyoffset": true, "array": true, "hidden": true,
	"parent": true, "assign": true, "history": true, "parsefloat": true, "blur": true,
	"image": true, "parseint": true, "button": true, "images": true, "password": true,
	"checkbox": true, "infinity": true, "pkcs11": true, "clearinterval": true,
	"isfinite": true, "plugin": true, "cleartimeout": true, "isnan": true,
	"prompt": true, "clientinformation": true, "isprototypeof": true,
	"propertyisenum": true, "close": true, "java": true, "prototype": true,
	"closed": true, "javaarray": true, "radio": true, "confirm": true, "javaclass": true,
	"reset": true, "constructor": true, "javaobject": true, "screenx": true,
	"crypto": true, "javapackage": true, "screeny": true, "date": true,
	"innerheight": true, "scroll": true, "decodeuri": true, "innerwidth": true,
	"secure": true, "decodeuricomponent": true, "layer": true, "select": true,
	"defaultstatus": true, "layers": true, "self": true, "document": true,
	"length": true, "setinterval": true, "element": true, "link": true,
	"settimeout": true, "elements": true, "location": true, "status": true,
	"embed": true, "math": true, "string": true, "embeds": true, "mimetypes": true,
	"submit": true, "encodeuri": true, "name": true, "taint": true,
	"encodeuricomponent": true, "nan": true, "text": true, "escape": true,
	"navigate": true, "textarea": true, "eval": true, "navigator": true, "top": true,
	"event": true, "number": true, "tostring": true, "fileupload": true, "object": true,
	"undefined": true, "focus": true, "offscreenbuffering": true, "unescape": true,
	"form": true, "open": true, "untaint": true, "forms": true, "opener": true,
	"valueof": true, "frame": true, "option": true, "window": true, "yield": true,
}

type Config struct {
	jsFlag        bool
	urlFlag       bool
	httpClient    *http.Client
	wordPattern   *regexp.Regexp
	dotSplitRegex *regexp.Regexp
}

func main() {
	jsFlag := flag.Bool("js", false, "Process JavaScript files for wordlist extraction")
	urlFlag := flag.Bool("url", false, "Process URLs for path component wordlist extraction")
	flag.Parse()

	config := &Config{
		jsFlag:  *jsFlag,
		urlFlag: *urlFlag,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		wordPattern:   regexp.MustCompile(`[a-zA-Z0-9_\-\.]+`),
		dotSplitRegex: regexp.MustCompile(`\.`),
	}

	args := flag.Args()
	if len(args) > 0 {
		for _, url := range args {
			processURL(url, config)
		}
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			url := strings.TrimSpace(scanner.Text())
			if url != "" {
				processURL(url, config)
			}
		}

		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			os.Exit(1)
		}
	}
}

func processURL(urlStr string, config *Config) {
	if config.urlFlag {
		extractURLPathWords(urlStr)
		return
	}
	
	if config.jsFlag {
		if !strings.Contains(urlStr, "://") || !strings.Contains(urlStr, ".js") {
			fmt.Fprintf(os.Stderr, "Bad URL: %s, please check your url, pass..\n", urlStr)
			return
		}

		content, err := fetchURL(urlStr, config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching %s: %v\n", urlStr, err)
			return
		}

		words := extractWords(content, config)
		printWords(words)
		return
	}
	
	fmt.Fprintf(os.Stderr, "No processing mode specified. Use -js or -url flag.\n")
}

func fetchURL(url string, config *Config) (string, error) {
	resp, err := config.httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func extractWords(content string, config *Config) []string {
	matches := config.wordPattern.FindAllString(content, -1)
	
	uniqueWords := make(map[string]bool)
	
	for _, word := range matches {
		if strings.Contains(word, ".") {
			parts := config.dotSplitRegex.Split(word, -1)
			w := parts[len(parts)-1]
			if w != "" && !blacklist[strings.ToLower(w)] {
				uniqueWords[w] = true
			}
		} else if len(word) == 1 {
			if (word[0] >= 'a' && word[0] <= 'z') || (word[0] >= 'A' && word[0] <= 'Z') {
				if !blacklist[strings.ToLower(word)] {
					uniqueWords[word] = true
				}
			}
		} else {
			if !blacklist[strings.ToLower(word)] {
				uniqueWords[word] = true
			}
		}
	}
	
	result := make([]string, 0, len(uniqueWords))
	for word := range uniqueWords {
		result = append(result, word)
	}
	
	return result
}

func printWords(words []string) {
	for _, word := range words {
		fmt.Println(word)
	}
}

func extractURLPathWords(urlStr string) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing URL %s: %v\n", urlStr, err)
		return
	}
	
	path := parsedURL.Path
	decodedPath, err := url.PathUnescape(path)
	if err == nil {
		path = decodedPath
	}
	
	query := parsedURL.RawQuery
	decodedQuery, err := url.QueryUnescape(query)
	if err == nil {
		query = decodedQuery
	}
	
	words := make(map[string]bool)
	
	pathParts := strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == '-' || r == '_' || r == '.' || r == '='
	})
	
	for _, part := range pathParts {
		if len(part) > 0 && !blacklist[strings.ToLower(part)] {
			words[part] = true
			
			camelParts := splitCamelCase(part)
			for _, camelPart := range camelParts {
				if len(camelPart) > 0 && !blacklist[strings.ToLower(camelPart)] {
					words[camelPart] = true
				}
			}
		}
	}
	
	queryParts := strings.FieldsFunc(query, func(r rune) bool {
		return r == '&' || r == '=' || r == ';'
	})
	
	for _, part := range queryParts {
		if len(part) > 0 && !blacklist[strings.ToLower(part)] {
			words[part] = true
			
			camelParts := splitCamelCase(part)
			for _, camelPart := range camelParts {
				if len(camelPart) > 0 && !blacklist[strings.ToLower(camelPart)] {
					words[camelPart] = true
				}
			}
		}
	}
	
	result := make([]string, 0, len(words))
	for word := range words {
		result = append(result, word)
	}
	
	printWords(result)
}

func splitCamelCase(s string) []string {
	var result []string
	var current strings.Builder
	
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
		}
		current.WriteRune(r)
	}
	
	if current.Len() > 0 {
		result = append(result, current.String())
	}
	
	return result
}

func printUsage() {
	fmt.Println("Wordlist Extractor")
	fmt.Println("Usage:")
	fmt.Printf("\t%s -js https://example.com/js/main.js\n", os.Args[0])
	fmt.Printf("\t%s -url https://example.com/path/resource?param=value\n", os.Args[0])
	fmt.Printf("\tcat jsfiles.txt | %s -js\n", os.Args[0])
	fmt.Printf("\tcat urls.txt | %s -url\n", os.Args[0])
	fmt.Println("\nModes:")
	fmt.Println("\t-js\tExtract keywords from JavaScript files")
	fmt.Println("\t-url\tExtract keywords from URL paths and query parameters")
}
