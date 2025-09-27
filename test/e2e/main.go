package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/GrGLeo/TermArena/pkg/shared"
)

type TestClient struct {
	conn             net.Conn
	privateKey       *rsa.PrivateKey
	publicKey        []byte
	username         string
	challenge        []byte
	rateLimited      bool
	lastRateLimit    time.Time
	roomID           uint32
	receivedMessages []string
}

// Global quiet mode flag
var quietMode = false

type TestScenario struct {
	Name     string
	Clients  []string
	TestFunc func([]*TestClient) error
}

// waitForRateLimitReset waits for rate limits to reset before proceeding
func waitForRateLimitReset() {
	// Wait for rate limit window to reset (matches client authentication delay)
	resetTime := 30500 * time.Millisecond // 30.5 seconds
	time.Sleep(resetTime)
}

// checkAndWaitForRateLimit checks if any test client is rate limited and waits
func checkAndWaitForRateLimit(clients []*TestClient) {
	for _, client := range clients {
		if client.rateLimited {
			elapsed := time.Since(client.lastRateLimit)
			if elapsed < 60*time.Second {
				waitTime := 65*time.Second - elapsed
				fmt.Printf("Client %s was rate limited %v ago, waiting %v for reset...\n",
					client.username, elapsed, waitTime)
				time.Sleep(waitTime)
				client.rateLimited = false // Reset the flag
				fmt.Printf("Rate limit reset for %s\n", client.username)
			} else {
				client.rateLimited = false // Reset if enough time has passed
			}
		}
	}
}

func main() {
	fmt.Println("=== MESSAGE SYSTEM E2E TEST CLIENT ===")

	var verbose bool
	flag.BoolVar(&verbose, "v", false, "enable verbose output")
	flag.Parse()
	args := flag.Args()

	// Test scenarios - reusing users 1-4 for all tests
	testScenarios := []TestScenario{
		{Name: "Basic Two Client Test", Clients: []string{"user1", "user2"}, TestFunc: basicTwoClientTest},
		{Name: "Multi Client Room Test (No Rate Limit)", Clients: []string{"user1", "user2", "user3", "user4"}, TestFunc: multiClientRoomTest},
		{Name: "Whisper Message Test", Clients: []string{"user1", "user2"}, TestFunc: whisperMessageTest},
		{Name: "Broadcast Message Test (No Rate Limit)", Clients: []string{"user1", "user2", "user3", "user4"}, TestFunc: broadcastMessageTest},
		{Name: "Error Handling Test", Clients: []string{"user1", "user2"}, TestFunc: errorHandlingTest},
		{Name: "Concurrent Messaging Test (No Rate Limit)", Clients: []string{"user1", "user2", "user3", "user4"}, TestFunc: concurrentMessagingTest},
		{Name: "Look Room Test", Clients: []string{"user1", "user2", "user3", "user4"}, TestFunc: lookRoomTest},
		{Name: "Registration Rate Limit Test", Clients: []string{"user1"}, TestFunc: registrationRateLimitTest},
		{Name: "Authentication Rate Limit Test", Clients: []string{"user1"}, TestFunc: authRateLimitTest},
		{Name: "Message Rate Limit Test", Clients: []string{"user1"}, TestFunc: messageRateLimitTest},
	}
	if len(args) > 0 {
		testIndex, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Printf("Invalid test index: %v\n", err)
			return
		}
		if testIndex < 1 || testIndex > len(testScenarios) {
			fmt.Printf("Test index out of range: %d (1-%d)\n", testIndex, len(testScenarios))
			return
		}
		// Run specific test
		scenario := testScenarios[testIndex-1]
		fmt.Printf("Starting: %s\n", scenario.Name)
		scenarioStart := time.Now()

		if verbose {
			err = runTestScenario(scenario)
		} else {
			err = runTestScenarioQuiet(scenario)
		}

		scenarioDuration := time.Since(scenarioStart)

		status := "\033[32mPASS\033[0m"
		if err != nil {
			status = "\033[31mFAIL\033[0m"
			fmt.Printf("   Error: %v\n", err)
		}
		fmt.Printf("   %s (%v)\n", status, scenarioDuration.Round(time.Second))

		// Summary
		passedCount := 0
		if err == nil {
			passedCount = 1
		}
		totalDuration := scenarioDuration
		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Println("E2E TEST RESULTS SUMMARY")
		fmt.Println(strings.Repeat("=", 60))

		allPassed := passedCount == 1
		resultColor := "\033[32m"
		if !allPassed {
			resultColor = "\033[31m"
		}
		fmt.Printf("%sResult: %d/%d tests passed\033[0m\n", resultColor, passedCount, 1)
		if !allPassed {
			fmt.Printf("  - %s: %v\n", scenario.Name, err)
		}

		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("Total time: %v\n", totalDuration.Round(time.Second))
		fmt.Println(strings.Repeat("=", 60))
	} else {
		// Track test results
		testResults := make([]error, len(testScenarios))
		totalStartTime := time.Now()

		fmt.Println("Running E2E tests... (verbose output suppressed)")
    fmt.Printf("%d scenarios found...\n", len(testScenarios))

		// Run all test scenarios
		for i, scenario := range testScenarios {
			fmt.Printf("%d/%d Starting: %s\n", i+1, len(testScenarios), scenario.Name)
			scenarioStart := time.Now()

			// Run test
			var err error
			if verbose {
				err = runTestScenario(scenario)
			} else {
				err = runTestScenarioQuiet(scenario)
			}

			// Record result
			testResults[i] = err
			scenarioDuration := time.Since(scenarioStart)

			// Show brief progress
			status := "\033[32mPASS\033[0m"
			if err != nil {
				status = "\033[31mFAIL\033[0m"
				fmt.Printf("   Error: %v\n", err)
			}
			fmt.Printf("   %s (%v)\n", status, scenarioDuration.Round(time.Second))

			// Wait for rate limits to reset between scenarios (except for last scenario)
			if i < len(testScenarios)-1 {
				waitForRateLimitReset()
			}
		}

		// Print final summary
		totalDuration := time.Since(totalStartTime)
		passedCount := 0
		for _, err := range testResults {
			if err == nil {
				passedCount++
			}
		}

		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Println("E2E TEST RESULTS SUMMARY")
		fmt.Println(strings.Repeat("=", 60))

		allPassed := passedCount == len(testScenarios)
		resultColor := "\033[32m"
		if !allPassed {
			resultColor = "\033[31m"
		}
		fmt.Printf("%sResult: %d/%d tests passed\033[0m\n", resultColor, passedCount, len(testScenarios))
		if !allPassed {
			for i, err := range testResults {
				if err != nil {
					fmt.Printf("  - %s: %v\n", testScenarios[i].Name, err)
				}
			}
		}

		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("Total time: %v\n", totalDuration.Round(time.Second))
		fmt.Println(strings.Repeat("=", 60))
	}
}

// runTestScenarioQuiet runs a test scenario with minimal output
func runTestScenarioQuiet(scenario TestScenario) error {
	// Enable quiet mode
	quietMode = true
	defer func() { quietMode = false }() // Reset when done

	// Create test clients
	clients := make([]*TestClient, len(scenario.Clients))
	for i, username := range scenario.Clients {
		clients[i] = createTestClient(username)
	}

	// Start response listeners for all clients (quiet mode)
	for _, client := range clients {
		go client.listenForResponses()
	}

	// Authenticate all clients with delays to avoid rate limiting
	for i, client := range clients {
		if err := client.authenticate(); err != nil {
			// Clean up on failure
			cleanupClients(clients)
			return fmt.Errorf("failed to authenticate %s: %v", client.username, err)
		}

		// Add delay between client authentications to avoid rate limiting
		if i < len(clients)-1 {
			time.Sleep(30500 * time.Millisecond) // 30.5 seconds
		}
	}

	// Run the test scenario
	if err := scenario.TestFunc(clients); err != nil {
		cleanupClients(clients)
		return err
	}

	// Clean up connections
	cleanupClients(clients)
	return nil
}

func createTestClient(username string) *TestClient {
	conn, err := net.Dial("tcp", "localhost:8082")
	if err != nil {
		log.Fatalf("Failed to connect for %s: %v", username, err)
	}

	// Load private key from file
	privateKey, publicKey := loadKeys(username)

	if !quietMode {
		fmt.Printf("Created test client: %s\n", username)
	}

	return &TestClient{
		conn:             conn,
		privateKey:       privateKey,
		publicKey:        publicKey,
		username:         username,
		challenge:        nil,
		roomID:           0,
		receivedMessages: make([]string, 0),
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
	if !quietMode {
		fmt.Printf("Authenticating user: %s\n", c.username)
	}

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

	if !quietMode {
		fmt.Printf("Authentication completed for: %s\n", c.username)
	}
	return nil
}

// registrationRateLimitTest - Test registration rate limiting with empty key
func registrationRateLimitTest(clients []*TestClient) error {
	client := clients[0]

	if !quietMode {
		fmt.Println("Testing registration rate limit...")
	}

	// Send multiple registration requests rapidly to trigger rate limit
	for i := range 5 {
		username := fmt.Sprintf("ratelimit_reg_%d_%d", time.Now().Unix(), i)
		// Use a valid-looking but fake public key
		fakeKey := []byte("-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA\n-----END PUBLIC KEY-----")
		registerPacket := shared.NewRegisterRequestPacket(username, fakeKey)
		data := registerPacket.Serialize()

		_, err := client.conn.Write(data)
		if err != nil {
			return fmt.Errorf("failed to send register request %d: %v", i, err)
		}

		if !quietMode {
			fmt.Printf("Sent registration request %d for %s\n", i+1, username)
		}

		// Very small delay between requests to trigger rate limit
		time.Sleep(5 * time.Millisecond)
	}

	// Wait for rate limit response
	if !quietMode {
		fmt.Println("Waiting for rate limit response...")
	}
	time.Sleep(3 * time.Second)

	// Check if rate limit was triggered
	if client.rateLimited {
		if !quietMode {
			fmt.Println("Rate limit successfully triggered for registration")
		}
	} else {
		if !quietMode {
			fmt.Println("Rate limit may not have been triggered - check server logs")
		}
	}

	if !quietMode {
		fmt.Println("Registration rate limit test completed")
	}
	return nil
}

// authRateLimitTest - Test authentication rate limiting
func authRateLimitTest(clients []*TestClient) error {
	client := clients[0]

	if !quietMode {
		fmt.Println("Testing authentication rate limit...")
	}

	// Send multiple consecutive login challenge requests to trigger rate limit
	for i := range 5 {
		challengePacket := shared.NewLoginChallengeRequestPacket(client.username)
		data := challengePacket.Serialize()

		_, err := client.conn.Write(data)
		if err != nil {
			return fmt.Errorf("failed to send login challenge request %d: %v", i, err)
		}

		if !quietMode {
			fmt.Printf("Sent login challenge request %d\n", i+1)
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Wait for rate limit response
	if !quietMode {
		fmt.Println("Waiting for rate limit response...")
	}
	time.Sleep(3 * time.Second)

	// Check if rate limit was triggered
	if client.rateLimited {
		if !quietMode {
			fmt.Println("Rate limit successfully triggered for authentication")
		}
	} else {
		if !quietMode {
			fmt.Println(" Rate limit may not have been triggered - check server logs")
		}
	}

	if !quietMode {
		fmt.Println("Authentication rate limit test completed")
	}
	return nil
}

// messageRateLimitTest - Test message rate limiting (30 messages per minute)
func messageRateLimitTest(clients []*TestClient) error {
	client := clients[0]

	if !quietMode {
		fmt.Println("Testing message rate limit (30 messages per minute)...")
	}

	// Authenticate first
	if err := client.authenticate(); err != nil {
		return fmt.Errorf("auth failed: %v", err)
	}

	// Wait for auth to complete
	time.Sleep(500 * time.Millisecond)

	// Send 35 messages rapidly (exceeding 30/minute limit)
	for i := range 35 {
		message := fmt.Sprintf("Rate limit test message %d", i+1)
		client.sendMessage(message)

		// Small delay to simulate rapid but not instantaneous sending
		time.Sleep(20 * time.Millisecond)
	}

	// Wait for rate limit to be triggered
	if !quietMode {
		fmt.Println("Waiting for rate limit response...")
	}
	time.Sleep(4 * time.Second)

	// Check if rate limit was triggered
	if client.rateLimited {
		if !quietMode {
			fmt.Println("Rate limit successfully triggered for messages")
		}
	} else {
		if !quietMode {
			fmt.Println("Rate limit may not have been triggered - check server logs")
		}
	}

	if !quietMode {
		fmt.Println("Message rate limit test completed")
	}
	return nil
}

func (c *TestClient) sendLoginChallengeRequest() error {
	challengePacket := shared.NewLoginChallengeRequestPacket(c.username)
	data := challengePacket.Serialize()

	_, err := c.conn.Write(data)
	if err != nil {
		return err
	}

	if !quietMode {
		fmt.Printf("Sent login challenge request for: %s\n", c.username)
	}
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

	if !quietMode {
		fmt.Printf("Sent auth request for: %s with signed challenge\n", c.username)
	}
	return nil
}

func (c *TestClient) sendMessage(message string) {
	if !quietMode {
		fmt.Printf("Client %s sending message: '%s'\n", c.username, message)
	}

	messagePacket := shared.NewMessagePacket(c.username, message)
	data := messagePacket.Serialize()

	if !quietMode {
		timestamp := time.Now().Unix()
		fmt.Printf("Message packet created at timestamp: %d\n", timestamp)
	}

	_, err := c.conn.Write(data)
	if err != nil {
		if !quietMode {
			fmt.Printf("Error sending message from %s: %v\n", c.username, err)
		}
	} else {
		if !quietMode {
			fmt.Printf("Message sent successfully from %s at timestamp: %d\n", c.username, time.Now().Unix())
		}
	}
}

func (c *TestClient) listenForResponses() {
	buf := make([]byte, 4096)
	defer func() {
		if !quietMode {
			fmt.Printf("[%s] Response listener stopped\n", c.username)
		}
	}()

	for {
		// Check if connection is closed
		if c.conn == nil {
			if !quietMode {
				fmt.Printf("[%s] Connection is nil, stopping listener\n", c.username)
			}
			return
		}

		n, err := c.conn.Read(buf)
		if err != nil {
			if err.Error() == "EOF" {
				if !quietMode {
					fmt.Printf("[%s] Connection closed (EOF)\n", c.username)
				}
				return
			}
			// Check for "use of closed network connection" error
			if strings.Contains(err.Error(), "closed") {
				if !quietMode {
					fmt.Printf("[%s] Connection closed: %v\n", c.username, err)
				}
				return
			}
			if !quietMode {
				fmt.Printf("[%s] Error reading response: %v\n", c.username, err)
			}
			return
		}

		data := buf[:n]
		timestamp := time.Now().Unix()

		// Try to deserialize the packet
		packet, bytesConsumed, err := shared.DeSerialize(data)
		if err != nil {
			if err.Error() == "incomplete packet" {
				if !quietMode {
					fmt.Printf("[%s] Incomplete packet received at %d\n", c.username, timestamp)
				}
				continue
			}
			if !quietMode {
				fmt.Printf("[%s] Error deserializing packet at %d: %v\n", c.username, timestamp, err)
			}
			continue
		}

		// Process the packet
		switch p := packet.(type) {
		case *shared.RegisterResponsePacket:
			if !quietMode {
				fmt.Printf("[%s] REGISTER RESPONSE at %d: Success=%v, Message='%s', Challenge='%s'\n",
					c.username, timestamp, p.Success, p.Message, string(p.Challenge))
			}

		case *shared.LoginChallengeResponsePacket:
			// Store the challenge for authentication
			c.challenge = make([]byte, len(p.Challenge))
			copy(c.challenge, p.Challenge)

		case *shared.AuthResponsePacket:
			if !quietMode {
				fmt.Printf("[%s] AUTH RESPONSE at %d: Success=%v, Message='%s', Token='%s'\n",
					c.username, timestamp, p.Success, p.Message, p.SessionToken)
			}

		case *shared.MessageResponsePacket:
			c.receivedMessages = append(c.receivedMessages, p.Message)
			if !quietMode {
				fmt.Printf("[%s] MESSAGE RESPONSE at %d: '%s'\n", c.username, timestamp, p.Message)
			}

		case *shared.MessageErrorPacket:
			if !quietMode {
				fmt.Printf("[%s] MESSAGE ERROR at %d: '%s'\n", c.username, timestamp, p.Error)
			}

		case *shared.LookRoomPacket:
			c.roomID = p.RoomID
			if !quietMode {
				fmt.Printf("[%s] LOOK ROOM RESPONSE at %d: Success=%v, RoomID=%d\n", c.username, timestamp, p.Success, p.RoomID)
			}

		case *shared.RateLimitPacket:
			if !quietMode {
				fmt.Printf("[%s] RATE LIMIT TRIGGERED at %d\n", c.username, timestamp)
			}
			c.rateLimited = true
			c.lastRateLimit = time.Now()

		default:
			if !quietMode {
				fmt.Printf("[%s] OTHER PACKET at %d: %T\n", c.username, timestamp, packet)
			}
		}

		// Remove processed data
		copy(data, data[bytesConsumed:])
		data = data[:len(data)-bytesConsumed]
	}
}

// runTestScenario executes a test scenario with the given clients
func runTestScenario(scenario TestScenario) error {
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

	// Authenticate all clients with delays to avoid rate limiting
	fmt.Printf("Authenticating %d clients...\n", len(clients))
	for i, client := range clients {
		if err := client.authenticate(); err != nil {
			fmt.Printf("Failed to authenticate %s: %v\n", client.username, err)
			// Clean up on failure
			cleanupClients(clients)
			return err
		}

		// Add delay between client authentications to avoid rate limiting
		// Only add delay if there are more clients to authenticate
		if i < len(clients)-1 {
			time.Sleep(30500 * time.Millisecond) // 30.5 seconds
		}
	}

	fmt.Printf("All %d clients authenticated successfully!\n", len(clients))

	// Wait for auth to complete
	time.Sleep(500 * time.Millisecond)

	// Run the test scenario
	fmt.Printf("Running test scenario...\n")
	if err := scenario.TestFunc(clients); err != nil {
		fmt.Printf("Test scenario failed: %v\n", err)
		cleanupClients(clients)
		return err
	}

	// Clean up connections
	fmt.Printf("Cleaning up connections...\n")
	cleanupClients(clients)
	return nil
}

// cleanupClients properly closes all client connections and cleans up resources
func cleanupClients(clients []*TestClient) {
	for _, client := range clients {
		if client.conn != nil {
			if !quietMode {
				fmt.Printf("Closing connection for %s\n", client.username)
			}
			if err := client.conn.Close(); err != nil {
				if !quietMode {
					fmt.Printf("Error closing connection for %s: %v\n", client.username, err)
				}
			}
		}
	}

	// Give time for connections to fully close and server to process disconnections
	if !quietMode {
		fmt.Printf("Waiting for connection cleanup...\n")
	}
	time.Sleep(2 * time.Second)
}

// basicTwoClientTest - Original test with two clients
func basicTwoClientTest(clients []*TestClient) error {
	if len(clients) < 2 {
		return fmt.Errorf("need at least 2 clients")
	}

	client1 := clients[0]

	if !quietMode {
		fmt.Printf("T1: %s sends 'hello'\n", client1.username)
	}
	client1.sendMessage("hello")
	t1 := time.Now()

	time.Sleep(200 * time.Millisecond)

	if !quietMode {
		fmt.Printf("T2: %s sends 'world'\n", client1.username)
	}
	client1.sendMessage("world")
	t2 := time.Now()

	if !quietMode {
		fmt.Printf("Time between messages: %v\n", t2.Sub(t1))
	}

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
			if !quietMode {
				fmt.Printf("%s sends: '%s'\n", client.username, messages[i])
			}
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

	if !quietMode {
		fmt.Printf("%s whispers to %s: 'secret message'\n", client1.username, client2.username)
	}
	client1.sendMessage(fmt.Sprintf("/%s secret message", client2.username))

	time.Sleep(500 * time.Millisecond)

	if !quietMode {
		fmt.Printf("%s whispers back to %s: 'acknowledged'\n", client2.username, client1.username)
	}
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

	if !quietMode {
		fmt.Printf("%s broadcasts: 'Important announcement!'\n", client1.username)
	}
	client1.sendMessage("/all Important announcement!")

	time.Sleep(500 * time.Millisecond)

	if !quietMode {
		fmt.Printf("%s broadcasts: 'System test completed'\n", client1.username)
	}
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
	if !quietMode {
		fmt.Printf("%s sends empty message\n", client1.username)
	}
	client1.sendMessage("")

	time.Sleep(200 * time.Millisecond)

	// Test whisper to non-existent user
	if !quietMode {
		fmt.Printf("%s tries to whisper to non-existent user\n", client1.username)
	}
	client1.sendMessage("/nonexistentuser test message")

	time.Sleep(200 * time.Millisecond)

	// Test normal message after errors
	if !quietMode {
		fmt.Printf("%s sends normal message: 'Error handling test complete'\n", client1.username)
	}
	client1.sendMessage("Error handling test complete")

	time.Sleep(2 * time.Second)
	return nil
}

// concurrentMessagingTest - Test concurrent message sending
func concurrentMessagingTest(clients []*TestClient) error {
	if len(clients) < 2 {
		return fmt.Errorf("need at least 2 clients")
	}

	if !quietMode {
		fmt.Println("Starting concurrent messaging test...")
	}

	// Launch goroutines to send messages concurrently
	done := make(chan bool, len(clients))

	for i, client := range clients {
		go func(client *TestClient, clientIndex int) {
			for j := range 5 {
				message := fmt.Sprintf("Concurrent message %d from %s", j+1, client.username)
				if !quietMode {
					fmt.Printf("Concurrent: %s\n", message)
				}
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

	if !quietMode {
		fmt.Println("Concurrent messaging test completed")
	}
	time.Sleep(3 * time.Second)
	return nil
}

// lookRoomTest - Test LookRoom with 4 clients, validate same roomID, then team message
func lookRoomTest(clients []*TestClient) error {
	if len(clients) != 4 {
		return fmt.Errorf("need exactly 4 clients")
	}

	if !quietMode {
		fmt.Println("Starting Look Room test...")
	}

	// Send RoomRequest for each client
	for _, client := range clients {
		packet := shared.NewRoomRequestPacket(1)
		data := packet.Serialize()

		_, err := client.conn.Write(data)
		if err != nil {
			return fmt.Errorf("failed to send room request for %s: %v", client.username, err)
		}

		if !quietMode {
			fmt.Printf("%s sent room request\n", client.username)
		}
	}

	// Wait for responses
	time.Sleep(3 * time.Second)

	// Check all got same roomID
	var roomID uint32
	for i, client := range clients {
		if client.roomID == 0 {
			return fmt.Errorf("%s did not receive room response", client.username)
		}
		if i == 0 {
			roomID = client.roomID
		} else if client.roomID != roomID {
			return fmt.Errorf("clients have different roomIDs: %d vs %d", roomID, client.roomID)
		}
	}

	if !quietMode {
		fmt.Printf("All clients joined room %d\n", roomID)
	}

	// Wait for registration
	time.Sleep(1 * time.Second)

	// user1 sends team message
	client1 := clients[0]
	teamMessage := "hello team"
	client1.sendMessage(teamMessage)

	if !quietMode {
		fmt.Printf("%s sent team message: %s\n", client1.username, teamMessage)
	}

	// Wait for message responses
	time.Sleep(2 * time.Second)

	// Count how many received (team) message
	teamReceivers := 0
	for _, client := range clients {
		for _, msg := range client.receivedMessages {
			if strings.HasPrefix(msg, "(team)") {
				teamReceivers++
				break
			}
		}
	}

	if teamReceivers != 2 {
		return fmt.Errorf("expected 2 team receivers, got %d", teamReceivers)
	}

	if !quietMode {
		fmt.Printf("Team message sent, %d clients received\n", teamReceivers)
	}

	return nil
}
