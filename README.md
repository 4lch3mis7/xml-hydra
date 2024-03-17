# XML-Hydra
XML-Hydra is a tool to bruteforce user passwords via public facing XML-RPC interface in an Wordpress application.

## Installation
go install github.com/prasant-paudel/xml-hydra@latest

## Usage
| Flag | Description
|------|-------------
| -t   | Target URL
| -u   | Username
| -w   | Wordlist for passwords
| -h   | Shows help message

## Example
```
xml-hydra -t https://example.com/xmlrpc.php -u username -w passwords.txt
```