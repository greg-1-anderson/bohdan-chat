package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"

	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]bool) // connected clients
var broadcast = make(chan Message)           // broadcast channel

// Configure the upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Define our message object
type Message struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

func main() {
	// Create a simple file server
	fs := http.FileServer(http.Dir("../public"))
	http.Handle("/", fs)

	// Configure websocket route
	http.HandleFunc("/ws", handleConnections)

	// Start listening for incoming chat messages
	go handleMessages()

	// Start the server on localhost port 8001 and log any errors
	log.Println("http server started on :8001")
	err := http.ListenAndServe(":8001", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Make sure we close the connection when the function returns
	defer ws.Close()

	// Register our new client
	clients[ws] = true

	for {
		var msg Message
		// Read in a new message as JSON and map it to a Message object
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}
		// Send the newly received message to the broadcast channel
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-broadcast
		// Send it out to every client that is currently connected
		sendMessage(msg)
		processMessageForChatOps(msg)
	}
}

func processMessageForChatOps(msg Message) {
	var isRollBot = regexp.MustCompile(`^@roll `)
	if isRollBot.MatchString(msg.Message) {
		processRollBot(msg.Message[6:])
		return
	}
}

func processRollBot(whatToRoll string) {
	re := regexp.MustCompile("([0-9]+)?d([0-9]+)([\\+-])?([0-9]+)?")
	dice := re.FindStringSubmatch(whatToRoll)
	if len(dice) == 0 {
		sendMessageFromBot("Error: cannot roll " + whatToRoll)
		return
	}
	rollDice(dice[1], dice[2], dice[3], dice[4])
}

func rollDice(number string, sides string, sign string, addition string) {
	n := 1
	if len(number) > 0 {
		n, _ = strconv.Atoi(number)
	}
	s, _ := strconv.Atoi(sides)

	total := randomDice(n, s)

	if len(addition) > 0 {
		a, _ := strconv.Atoi(addition)
		total = total + a
	}

	// var result = fmt.Sprintf("%s %s %s", number, sides, addition)
	result := fmt.Sprintf("%d", total)
	sendMessageFromBot(result)
}

func randomDice(n int, s int) int {
	total := 0

	for i := 0; i < n; i++ {
		roll := rand.Intn(s) + 1
		total = total + roll
	}

	return total
}

func sendMessageFromBot(message string) {
	msg := Message{Email: "bot@getpantheon.com", Username: "Bot", Message: message}
	sendMessage(msg)
}

func sendMessage(msg Message) {
	for client := range clients {
		err := client.WriteJSON(msg)
		if err != nil {
			log.Printf("error: %v", err)
			client.Close()
			delete(clients, client)
		}
	}
}
