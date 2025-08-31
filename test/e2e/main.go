package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GrGLeo/ctf/shared"
)

type TestClient struct {
	conn       net.Conn
	privateKey *rsa.PrivateKey
	publicKey  []byte
	username   string
	challenge  []byte
}

type TestScenario struct {
	Name     string
	Clients  []string
	TestFunc func([]*TestClient) error
}

func main() {
	fmt.Println("=== MESSAGE SYSTEM E2E TEST CLIENT ===")

	// Test scenarios - reusing users 1-4 for all tests
	testScenarios := []TestScenario{
		{Name: "Basic Two Client Test", Clients: []string{"user1", "user2"}, TestFunc: basicTwoClientTest},
		{Name: "Multi Client Room Test", Clients: []string{"user1", "user2", "user3", "user4"}, TestFunc: multiClientRoomTest},
		{Name: "Whisper Message Test", Clients: []string{"user1", "user2"}, TestFunc: whisperMessageTest},
		{Name: "Broadcast Message Test", Clients: []string{"user1", "user2", "user3", "user4"}, TestFunc: broadcastMessageTest},
		{Name: "Error Handling Test", Clients: []string{"user1", "user2"}, TestFunc: errorHandlingTest},
		{Name: "Concurrent Messaging Test", Clients: []string{"user1", "user2", "user3", "user4"}, TestFunc: concurrentMessagingTest},
	}

	// Run all test scenarios
	for i, scenario := range testScenarios {
		fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
		fmt.Printf("🚀 SCENARIO %d/%d: %s\n", i+1, len(testScenarios), scenario.Name)
		fmt.Printf("👥 Clients: %v\n", scenario.Clients)
		fmt.Printf(strings.Repeat("=", 60) + "\n")

		runTestScenario(scenario)

		fmt.Printf("\n✅ SCENARIO %d COMPLETED: %s\n", i+1, scenario.Name)
		fmt.Printf("⏳ Preparing for next scenario...\n")

		// Longer pause between scenarios to ensure complete cleanup
		time.Sleep(3 * time.Second)
	}

	fmt.Println("\n=== ALL E2E TESTS COMPLETED ===")
}

func createTestClient(username string) *TestClient {
	conn, err := net.Dial("tcp", "localhost:8082")
	if err != nil {
		log.Fatalf("Failed to connect for %s: %v", username, err)
	}

	// Load private key from file
	privateKey, publicKey := loadKeys(username)

	fmt.Printf("Created test client: %s\n", username)

	return &TestClient{
		conn:       conn,
		privateKey: privateKey,
		publicKey:  publicKey,
		username:   username,
		challenge:  nil,
	}
}

func loadKeys(username string) (*rsa.PrivateKey, []byte) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}

	keyPath := filepath.Join(homeDir, ".config", "term_arena", "keys", username+".key")

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		log.Fatalf("Failed to read key file %s: %v", keyPath, err)
	}

	// Decode PEM
	block, _ := pem.Decode(keyData)
	if block == nil {
		log.Fatalf("Failed to decode PEM for %s", username)
	}

	// Parse private key (supports both PKCS#1 and PKCS#8)
	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		log.Fatalf("Failed to parse private key for %s: %v", username, err)
	}

	// Type assert to RSA private key
	privateKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		log.Fatalf("Key is not an RSA private key for %s", username)
	}

	// Extract public key
	publicKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		log.Fatalf("Failed to marshal public key for %s: %v", username, err)
	}

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyDER,
	})

	return privateKey, publicKeyPEM
}

func (c *TestClient) authenticate() error {
	fmt.Printf("Authenticating user: %s\n", c.username)

	// Step 1: Send login challenge request
	if err := c.sendLoginChallengeRequest(); err != nil {
		return fmt.Errorf("login challenge request failed: %v", err)
	}

	// Wait for challenge response
	time.Sleep(1 * time.Second)

	// Step 2: Send auth request with signed challenge
	if err := c.sendAuthRequest(); err != nil {
		return fmt.Errorf("auth request failed: %v", err)
	}

	fmt.Printf("Authentication completed for: %s\n", c.username)
	return nil
}

func (c *TestClient) sendLoginChallengeRequest() error {
	challengePacket := shared.NewLoginChallengeRequestPacket(c.username)
	data := challengePacket.Serialize()

	_, err := c.conn.Write(data)
	if err != nil {
		return err
	}

	fmt.Printf("Sent login challenge request for: %s\n", c.username)
	return nil
}

func (c *TestClient) sendAuthRequest() error {
	// Check if we have a challenge from the server
	if c.challenge == nil {
		return fmt.Errorf("no challenge received from server")
	}

	// Sign the actual challenge received from server
	hashed := sha256.Sum256(c.challenge)
	signature, err := rsa.SignPKCS1v15(rand.Reader, c.privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return fmt.Errorf("failed to sign challenge: %v", err)
	}

	authPacket := shared.NewAuthRequestPacket(c.username, signature)
	data := authPacket.Serialize()

	_, err = c.conn.Write(data)
	if err != nil {
		return err
	}

	fmt.Printf("Sent auth request for: %s with signed challenge\n", c.username)
	return nil
}

func (c *TestClient) sendMessage(message string) {
	fmt.Printf("Client %s sending message: '%s'\n", c.username, message)

	messagePacket := shared.NewMessagePacket(c.username, message)
	data := messagePacket.Serialize()

	timestamp := time.Now().Unix()
	fmt.Printf("Message packet created at timestamp: %d\n", timestamp)

	_, err := c.conn.Write(data)
	if err != nil {
		fmt.Printf("Error sending message from %s: %v\n", c.username, err)
	} else {
		fmt.Printf("Message sent successfully from %s at timestamp: %d\n", c.username, time.Now().Unix())
	}
}

func (c *TestClient) listenForResponses() {
	buf := make([]byte, 4096)
	defer fmt.Printf("[%s] Response listener stopped\n", c.username)

	for {
		// Check if connection is closed
		if c.conn == nil {
			fmt.Printf("[%s] Connection is nil, stopping listener\n", c.username)
			return
		}

		n, err := c.conn.Read(buf)
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Printf("[%s] Connection closed (EOF)\n", c.username)
				return
			}
			// Check for "use of closed network connection" error
			if strings.Contains(err.Error(), "closed") {
				fmt.Printf("[%s] Connection closed: %v\n", c.username, err)
				return
			}
			fmt.Printf("[%s] Error reading response: %v\n", c.username, err)
			return
		}

		data := buf[:n]
		timestamp := time.Now().Unix()

		// Try to deserialize the packet
		packet, bytesConsumed, err := shared.DeSerialize(data)
		if err != nil {
			if err.Error() == "incomplete packet" {
				fmt.Printf("[%s] Incomplete packet received at %d\n", c.username, timestamp)
				continue
			}
			fmt.Printf("[%s] Error deserializing packet at %d: %v\n", c.username, timestamp, err)
			continue
		}

		// Process the packet
		switch p := packet.(type) {
		case *shared.RegisterResponsePacket:
			fmt.Printf("[%s] REGISTER RESPONSE at %d: Success=%v, Message='%s', Challenge='%s'\n",
				c.username, timestamp, p.Success, p.Message, string(p.Challenge))

		case *shared.LoginChallengeResponsePacket:
			fmt.Printf("[%s] LOGIN CHALLENGE RESPONSE at %d: Challenge='%s'\n",
				c.username, timestamp, string(p.Challenge))
			// Store the challenge for authentication
			c.challenge = make([]byte, len(p.Challenge))
			copy(c.challenge, p.Challenge)

		case *shared.AuthResponsePacket:
			fmt.Printf("[%s] AUTH RESPONSE at %d: Success=%v, Message='%s', Token='%s'\n",
				c.username, timestamp, p.Success, p.Message, p.SessionToken)

		case *shared.MessageResponsePacket:
			fmt.Printf("[%s] MESSAGE RESPONSE at %d: '%s'\n", c.username, timestamp, p.Message)

		case *shared.MessageErrorPacket:
			fmt.Printf("[%s] MESSAGE ERROR at %d: '%s'\n", c.username, timestamp, p.Error)

		default:
			fmt.Printf("[%s] OTHER PACKET at %d: %T\n", c.username, timestamp, packet)
		}

		// Remove processed data
		copy(data, data[bytesConsumed:])
		data = data[:len(data)-bytesConsumed]
	}
}

// runTestScenario executes a test scenario with the given clients
func runTestScenario(scenario TestScenario) {
	fmt.Printf("Setting up %d clients for test scenario...\n", len(scenario.Clients))

	// Create test clients
	clients := make([]*TestClient, len(scenario.Clients))
	for i, username := range scenario.Clients {
		clients[i] = createTestClient(username)
	}

	// Start response listeners for all clients
	for _, client := range clients {
		go client.listenForResponses()
	}

	// Authenticate all clients
	fmt.Printf("Authenticating %d clients...\n", len(clients))
	for _, client := range clients {
		if err := client.authenticate(); err != nil {
			fmt.Printf("Failed to authenticate %s: %v\n", client.username, err)
			// Clean up on failure
			cleanupClients(clients)
			return
		}
	}

	fmt.Printf("All %d clients authenticated successfully!\n", len(clients))

	// Wait for auth to complete
	time.Sleep(500 * time.Millisecond)

	// Run the test scenario
	fmt.Printf("Running test scenario...\n")
	if err := scenario.TestFunc(clients); err != nil {
		fmt.Printf("Test scenario failed: %v\n", err)
	}

	// Clean up connections
	fmt.Printf("Cleaning up connections...\n")
	cleanupClients(clients)
}

// cleanupClients properly closes all client connections and cleans up resources
func cleanupClients(clients []*TestClient) {
	for _, client := range clients {
		if client.conn != nil {
			fmt.Printf("Closing connection for %s\n", client.username)
			if err := client.conn.Close(); err != nil {
				fmt.Printf("Error closing connection for %s: %v\n", client.username, err)
			}
		}
	}

	// Give time for connections to fully close and server to process disconnections
	fmt.Printf("Waiting for connection cleanup...\n")
	time.Sleep(2 * time.Second)
}

// basicTwoClientTest - Original test with two clients
func basicTwoClientTest(clients []*TestClient) error {
	if len(clients) < 2 {
		return fmt.Errorf("need at least 2 clients")
	}

	client1 := clients[0]

	fmt.Printf("T1: %s sends 'hello'\n", client1.username)
	client1.sendMessage("hello")
	t1 := time.Now()

	time.Sleep(200 * time.Millisecond)

	fmt.Printf("T2: %s sends 'world'\n", client1.username)
	client1.sendMessage("world")
	t2 := time.Now()

	fmt.Printf("Time between messages: %v\n", t2.Sub(t1))

	// Wait to receive responses
	time.Sleep(2 * time.Second)
	return nil
}

// multiClientRoomTest - Test with multiple clients in a room
func multiClientRoomTest(clients []*TestClient) error {
	if len(clients) < 3 {
		return fmt.Errorf("need at least 3 clients")
	}

	// Send messages from different clients
	messages := []string{
		"Hello everyone!",
		"Nice to meet you all",
		"This is a group chat",
		"Testing multi-client messaging",
	}

	for i, client := range clients {
		if i < len(messages) {
			fmt.Printf("%s sends: '%s'\n", client.username, messages[i])
			client.sendMessage(messages[i])
			time.Sleep(100 * time.Millisecond)
		}
	}

	time.Sleep(3 * time.Second)
	return nil
}

// whisperMessageTest - Test private messaging between two clients
func whisperMessageTest(clients []*TestClient) error {
	if len(clients) < 2 {
		return fmt.Errorf("need at least 2 clients")
	}

	client1, client2 := clients[0], clients[1]

	fmt.Printf("%s whispers to %s: 'secret message'\n", client1.username, client2.username)
	client1.sendMessage(fmt.Sprintf("/%s secret message", client2.username))

	time.Sleep(500 * time.Millisecond)

	fmt.Printf("%s whispers back to %s: 'acknowledged'\n", client2.username, client1.username)
	client2.sendMessage(fmt.Sprintf("/%s acknowledged", client1.username))

	time.Sleep(2 * time.Second)
	return nil
}

// broadcastMessageTest - Test broadcasting to all clients
func broadcastMessageTest(clients []*TestClient) error {
	if len(clients) < 2 {
		return fmt.Errorf("need at least 2 clients")
	}

	client1 := clients[0]

	fmt.Printf("%s broadcasts: 'Important announcement!'\n", client1.username)
	client1.sendMessage("/all Important announcement!")

	time.Sleep(500 * time.Millisecond)

	fmt.Printf("%s broadcasts: 'System test completed'\n", client1.username)
	client1.sendMessage("/all System test completed")

	time.Sleep(2 * time.Second)
	return nil
}

// errorHandlingTest - Test error scenarios
func errorHandlingTest(clients []*TestClient) error {
	if len(clients) < 2 {
		return fmt.Errorf("need at least 2 clients")
	}

	client1 := clients[0]

	// Test empty message
	fmt.Printf("%s sends empty message\n", client1.username)
	client1.sendMessage("")

	time.Sleep(200 * time.Millisecond)

	// Test whisper to non-existent user
	fmt.Printf("%s tries to whisper to non-existent user\n", client1.username)
	client1.sendMessage("/nonexistentuser test message")

	time.Sleep(200 * time.Millisecond)

	// Test normal message after errors
	fmt.Printf("%s sends normal message: 'Error handling test complete'\n", client1.username)
	client1.sendMessage("Error handling test complete")

	time.Sleep(2 * time.Second)
	return nil
}

// concurrentMessagingTest - Test concurrent message sending
func concurrentMessagingTest(clients []*TestClient) error {
	if len(clients) < 2 {
		return fmt.Errorf("need at least 2 clients")
	}

	fmt.Println("Starting concurrent messaging test...")

	// Launch goroutines to send messages concurrently
	done := make(chan bool, len(clients))

	for i, client := range clients {
		go func(client *TestClient, clientIndex int) {
			for j := range 5 {
				message := fmt.Sprintf("Concurrent message %d from %s", j+1, client.username)
				fmt.Printf("Concurrent: %s\n", message)
				client.sendMessage(message)
				time.Sleep(50 * time.Millisecond)
			}
			done <- true
		}(client, i)
	}

	// Wait for all goroutines to complete
	for range clients {
		<-done
	}

	fmt.Println("Concurrent messaging test completed")
	time.Sleep(3 * time.Second)
	return nil
}
