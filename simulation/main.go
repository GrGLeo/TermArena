package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GrGLeo/ctf/shared"
)

var totalActions, totalPacketsReceived int64

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <client_count> <server_port>")
		return
	}

	clientCount, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("Invalid client count")
		return
	}

	serverPort := os.Args[2]

	var wg sync.WaitGroup
	for i := 0; i < clientCount; i++ {
		wg.Add(1)
		go runClient(&wg, i, serverPort)
	}

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			log.Printf("Total actions sent: %d, Total packets received: %d", atomic.LoadInt64(&totalActions), atomic.LoadInt64(&totalPacketsReceived))
			atomic.StoreInt64(&totalActions, 0)
			atomic.StoreInt64(&totalPacketsReceived, 0)
		}
	}()

	wg.Wait()
}

func runClient(wg *sync.WaitGroup, clientID int, serverPort string) {
	defer wg.Done()

	serverIP := os.Getenv("SERVER_IP")
	if len(os.Args) > 3 {
		serverIP = os.Args[3]
	} else if serverIP == "" {
		serverIP = "localhost"
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", serverIP, serverPort))
	if err != nil {
		log.Printf("Client %d: Failed to connect to server: %v", clientID, err)
		return
	}
	defer conn.Close()

	log.Printf("Client %d: Connected to server", clientID)

	// 1. Send Login Packet
	loginPacket := shared.NewLoginPacket("qweqwe", "qweqwe")
	_, err = conn.Write(loginPacket.Serialize())
	if err != nil {
		log.Printf("Client %d: Failed to send login packet: %v", clientID, err)
		return
	}

	// 2. Receive Login Response
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		log.Printf("Client %d: Failed to read login response: %v", clientID, err)
		return
	}

	packet, _, err := shared.DeSerialize(buf[:n])
	if err != nil {
		log.Printf("Client %d: Failed to deserialize login response: %v", clientID, err)
		return
	}

	if respPacket, ok := packet.(*shared.RespPacket); ok && respPacket.Success {
		log.Printf("Client %d: Login successful", clientID)
	} else {
		log.Printf("Client %d: Login failed", clientID)
		return
	}

	// 3. Send Room Request Packet
	roomRequestPacket := shared.NewRoomRequestPacket(0) // 0 for public room
	_, err = conn.Write(roomRequestPacket.Serialize())
	if err != nil {
		log.Printf("Client %d: Failed to send room request packet: %v", clientID, err)
		return
	}

	// 4. Receive Room Info
	n, err = conn.Read(buf)
	if err != nil {
		log.Printf("Client %d: Failed to read room info: %v", clientID, err)
		return
	}

	packet, _, err = shared.DeSerialize(buf[:n])
	if err != nil {
		log.Printf("Client %d: Failed to deserialize room info: %v", clientID, err)
		return
	}

	lookRoomPacket, ok := packet.(*shared.LookRoomPacket)
	if !ok || lookRoomPacket.Success != 0 {
		log.Printf("Client %d: Failed to get room info", clientID)
		return
	}

	log.Printf("Client %d: Received room info: %s", clientID, lookRoomPacket.RoomIP)

	// 5. Connect to Game Server
	gameConn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", serverIP, lookRoomPacket.RoomIP))
	if err != nil {
		log.Printf("Client %d: Failed to connect to game server: %v", clientID, err)
		return
	}
	defer gameConn.Close()

	log.Printf("Client %d: Connected to game server", clientID)

	// 6. Send Spell Selection Packet
	spellSelectionPacket := shared.NewSpellSelectionPacket(0, 1)
	_, err = gameConn.Write(spellSelectionPacket.Serialize())
	if err != nil {
		log.Printf("Client %d: Failed to send spell selection packet: %v", clientID, err)
		return
	}

	// 7. Wait for GameStartPacket
	log.Printf("Client %d: Waiting for GameStartPacket...", clientID)
	gameBuf := make([]byte, 0, 4096)
	tempBuf := make([]byte, 2048)
	var gameStarted bool
gameStartLoop:
	for {
		n, err := gameConn.Read(tempBuf)
		if err != nil {
			log.Printf("Client %d: Failed to read from game server while waiting for start: %v", clientID, err)
			return
		}
		gameBuf = append(gameBuf, tempBuf[:n]...)

		// A GameStartPacket is 3 bytes long (version, code, success)
		if len(gameBuf) >= 3 {
			// Check if it's a GameStartPacket (code 7)
			if gameBuf[1] == 7 {
				log.Printf("Client %d: Received GameStartPacket. Starting to send actions.", clientID)
				gameStarted = true
				gameBuf = gameBuf[3:] // Consume the packet from the buffer
				break gameStartLoop
			} else {
				log.Printf("Client %d: Received unexpected packet with code %d while waiting for start. Discarding buffer.", clientID, gameBuf[1])
				gameBuf = gameBuf[:0] // Clear buffer and wait for a clean packet
			}
		}
	}

	if !gameStarted {
		log.Printf("Client %d: Did not receive GameStartPacket. Exiting.", clientID)
		return
	}

	// 8. Send Random Actions and Read Board Packets
	for {
		action := rand.Intn(4) + 1 // 1 to 4
		actionPacket := shared.NewActionPacket(action)
		_, err := gameConn.Write(actionPacket.Serialize())
		if err != nil {
			log.Printf("Client %d: Failed to send action packet: %v", clientID, err)
			return
		}
		atomic.AddInt64(&totalActions, 1)

		// Read response from game server
		n, err := gameConn.Read(tempBuf)
		if err != nil {
			log.Printf("Client %d: Failed to read from game server: %v", clientID, err)
			return
		}
		if n > 0 {
			gameBuf = append(gameBuf, tempBuf[:n]...)
		}

		// Process all complete packets in the buffer
		for {
			packet, bytesConsumed, err := shared.DeSerialize(gameBuf)
			if err != nil {
				if err.Error() == "incomplete packet header" || err.Error() == "incomplete login packet" || err.Error() == "incomplete signin packet" || err.Error() == "incomplete response packet" || err.Error() == "incomplete room request packet" || err.Error() == "incomplete room create packet" || err.Error() == "incomplete look room packet" || err.Error() == "incomplete game start packet" || err.Error() == "incomplete action packet" || err.Error() == "incomplete board packet" || err.Error() == "incomplete delta packet" || err.Error() == "incomplete game close packet" || err.Error() == "incomplete end game packet" || err.Error() == "incomplete spell selection packet" || err.Error() == "incomplete shop response packet" || err.Error() == "incomplete purchase item packet" {
					// Not enough data, wait for more
					break
				} else {
					log.Printf("Client %d: Error deserializing packet: %v", clientID, err)
					// Discard the buffer to prevent getting stuck on a bad packet
					gameBuf = nil
					continue
				}
			}

			gameBuf = gameBuf[bytesConsumed:]

			if _, ok := packet.(*shared.BoardPacket); ok {
				atomic.AddInt64(&totalPacketsReceived, 1)
			} else {
				log.Printf("Client %d: Did not receive BoardPacket, but got %T", clientID, packet)
			}
		}
		time.Sleep(1 * time.Second)
	}
}
