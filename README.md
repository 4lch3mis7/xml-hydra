# XML-Hydra
XML-Hydra is a tool to bruteforce user passwords via public facing XML-RPC interface in a Wordpress application.

## Installation
```
go install github.com/prasant-paudel/xml-hydra@latest
```

## Usage
| Flag | Description
|------|-------------
| -t   | Target URL
| -u   | Username
| -w   | Wordlist for passwords
| -g   | Number of goroutines to execute at a time (Default=4)
| -P   | Proxy list
| -h   | Shows help message

## Example
```
xml-hydra -t https://example.com/xmlrpc.php -u username -w passwords.txt
```
```
xml-hydra -t https://example.com/xmlrpc.php -u username -w passwords.txt -P proxies.txt -g 10
```
