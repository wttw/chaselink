package main

import (
	"encoding/json"
	"fmt"
	"github.com/chaselink"
	"github.com/fatih/color"
	"github.com/spf13/pflag"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var color2 = color.New(color.FgGreen)
var color3 = color.New(color.FgYellow)
var color4 = color.New(color.FgRed)
var colorNeutral = color.New(color.FgCyan)

func main() {
	silent := false
	output := ""
	details := ""
	limit := pflag.Int("limit", 0, "limit number of requests")
	timeout := pflag.Int("timeout", 0, "timeout for all requests")
	useragent := pflag.String("useragent", "", "user-agent header")
	pflag.BoolVar(&silent, "silent", false, "print no progress")
	pflag.StringVar(&details, "details", "", "output file for json details")
	pflag.StringVarP(&output, "output", "o", "", "output file for final page")
	pflag.Parse()
	args := pflag.Args()
	if len(args) != 1 {
		log.Fatal("Usage: chaselink <url>")
	}

	options := []chaselink.Option{}

	if *limit > 0 {
		options = append(options, chaselink.Limit(*limit))
	}
	if *timeout > 0 {
		options = append(options, chaselink.Timeout(time.Duration(*timeout)*time.Second))
	}
	if *useragent != "" {
		options = append(options, chaselink.UserAgent(*useragent))
	}
	if !silent {
		options = append(options, chaselink.Progress(progress))
	}

	ua, err := chaselink.NewUser(options...)
	if err != nil {
		fatalf("Failed to create user: %s\n", err)
	}

	req, err := http.NewRequest(http.MethodGet, args[0], nil)
	if err != nil {
		fatalf("Failed to create request: %s\n", err)
	}

	uaErr := ua.Do(req)

	if details != "" {
		var f io.Writer
		if details == "-" {
			f = os.Stdout
		} else {
			f, err = os.Create(details)
			if err != nil {
				fatalf("Failed to create output file: %s\n", err)
			}
		}
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		err := enc.Encode(ua.Pages)
		if err != nil {
			fatalf("Failed to write output file: %s\n", err)
		}
	}

	if output != "" {
		if len(ua.Pages) == 0 {
			printWarn("No pages retrieved\n")
		} else {
			var f io.Writer
			if output == "-" {
				f = os.Stdout
			} else {
				f, err = os.Create(output)
				if err != nil {
					fatalf("Failed to create output file: %s\n", err)
				}
			}
			_, err = f.Write(ua.Pages[len(ua.Pages)-1].ResponseBody)
			if err != nil {
				fatalf("Failed to write output file: %s\n", err)
			}
		}
	}

	if uaErr != nil {
		fatalf("something failed: %s\n", err)
	}
}

func fatalf(format string, args ...interface{}) {
	printError(format, args...)
	os.Exit(1)
}

func printError(format string, args ...interface{}) {
	_, _ = color.New(color.FgRed).Fprintf(os.Stderr, format, args...)
}

func printWarn(format string, args ...interface{}) {
	_, _ = color.New(color.FgYellow).Fprintf(os.Stderr, format, args...)
}

func progress(page chaselink.Page) error {
	_, _ = fmt.Fprintf(os.Stderr, "%s\n", page.RequestUrl)
	_, _ = colorNeutral.Fprintf(os.Stderr, "Sent to %s", page.RemoteAddr)
	if page.TlsVersion != "" {
		_, _ = colorNeutral.Fprintf(os.Stderr, " (using TLS version %s)", page.TlsVersion)
	}
	_, _ = fmt.Fprintf(os.Stderr, "\n")
	_, _ = responseColor(page.StatusCode).Fprintf(os.Stderr, "%s\n", page.StatusMessage)
	if len(page.ResponseCookies) > 0 {
		cookies := make([]string, len(page.ResponseCookies))
		for i, cookie := range page.ResponseCookies {
			cookies[i] = cookie.Name
		}
		_, _ = colorNeutral.Fprintf(os.Stderr, "Cookies set: %s\n", strings.Join(cookies, ", "))
	}
	_, _ = fmt.Fprintf(os.Stderr, "\n")
	return nil
}

func responseColor(code int) *color.Color {
	switch {
	case code >= 200 && code < 300:
		return color2
	case code >= 300 && code < 400:
		return color3
	case code >= 400 && code < 500:
		return color4
	default:
		return colorNeutral
	}
}
