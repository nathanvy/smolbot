package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	server   = "irc.example.net:6697"
	channel  = "#bots"
	nick     = "smolbot"
	user     = "smolbot"
	realname = "Just a smol bean"
	password = ""
)

var ircConn net.Conn

func main() {
	var err error
	ircConn, err = tls.Dial("tcp", server, nil)
	if err != nil {
		log.Fatal("Unable to connect to IRC!", err)
	}
	defer ircConn.Close()

	//register connection manually, no capability negotiation for now

	//fmt.Fprintf(ircConn, "CAP LS 302\r\n")
	if password != "" {
		fmt.Fprintf(ircConn, "PASS %s\r\n", password)
	}
	fmt.Fprintf(ircConn, "NICK %s\r\n", nick)
	fmt.Fprintf(ircConn, "USER %s 0 * :%s\r\n", user, realname)
	//fmt.Fprintf(ircConn, "CAP END\r\n")

	// start the handler loop
	go ircListener(ircConn)

	http.HandleFunc("/sendmsg", webHookHandler)
	log.Println("Webhook listener running on :21337")
	log.Fatal(http.ListenAndServe("127.0.0.1:21337", nil))
}

func ircListener(conn net.Conn) {
	log.Println("IRC Listener running")

	registered := false //is the connection registered from our perspective?
	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		line := scanner.Text()
		//fmt.Println("<-", line)

		if strings.HasPrefix(line, "PING") {
			fmt.Fprintf(conn, "PONG %s\r\n", line[5:])
		}

		//post-registration hook
		// im not fucking parsing MOTDs, if we get an 005 we should be registered, see RFC 2812
		if !registered && strings.Contains(line, " 005 ") {
			registered = true
			fmt.Fprintf(conn, "JOIN %s\r\n", channel)
		}

		//chat command hooks
		if strings.Contains(line, "PRIVMSG") && strings.Contains(line, "!status") {
			fields := strings.Split(line, " ")
			if len(fields) >= 3 {
				target := fields[2]
				if target == channel {
					sendIRCMessage(target, "Status OK", 0)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Println("IRC parsing error:", err)
	}
}

func sendIRCMessage(target, message string, sendDelay time.Duration) {
	if ircConn == nil {
		log.Fatal("IRC connection has not been initialized")
	}

	const maxIRCLineLength = 510 // actually 512 but the CRLF consumes 2 bytes, see RFC 2812
	prefix := fmt.Sprintf("PRIVMSG %s :", target)
	maxLength := maxIRCLineLength - len(prefix)

	if utf8.RuneCountInString(message) <= maxLength {
		//no need to do any chunking
		msg := fmt.Sprintf("%s%s\r\n", prefix, message)
		_, err := fmt.Fprintf(ircConn, msg)
		if err != nil {
			log.Println("Failed to send message:", err)
		}

		return
	}

	//message too long, needs chunked
	runes := []rune(message)
	for len(runes) > 0 {
		chunksize := maxLength
		if utf8.RuneCountInString(string(runes)) > chunksize {
			chunksize := findSafeSplit(runes, maxLength-len(prefix))
			chunk := string(runes[:chunksize])
			runes = runes[chunksize:]

			msg := fmt.Sprintf("%s%s\r\n", prefix, chunk)
			_, err := fmt.Fprintf(ircConn, msg)
			if err != nil {
				log.Println("Failed to send message:", err)
				return
			}

			if sendDelay > 0 {
				time.Sleep(sendDelay)
			}
		}
	}
}

// let's try not to split mid-word shall we
func findSafeSplit(runes []rune, maxlength int) int {
	for i := maxlength; i > 0; i-- {
		if runes[i] == ' ' || runes[i] == '\t' {
			return i
		}
	}
	return maxlength //we tried our best
}

type WebhookPayload struct {
	Message string `json:"message"`
}

func webHookHandler(w http.ResponseWriter, r *http.Request) {
	//only listen on localhost for JSON POSTs
	if r.Method != http.MethodPost {
		http.Error(w, "I only accept POST requests", http.StatusMethodNotAllowed)
		return
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		//silently drop if malformed host
		return
	}

	if host != "127.0.0.1" && host != "::1" {
		//silently drop if not localhost
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Homie I only accept JSON", http.StatusUnsupportedMediaType)
		return
	}

	var payload WebhookPayload
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&payload); err != nil {
		http.Error(w, "Bogus JSON payload", http.StatusBadRequest)
		return
	}

	if payload.Message == "" {
		http.Error(w, "Message field cannot be empty", http.StatusBadRequest)
		return
	}

	//should be good, send to IRC
	sendIRCMessage(channel, payload.Message, 500*time.Millisecond)
	w.WriteHeader(http.StatusOK)
}
