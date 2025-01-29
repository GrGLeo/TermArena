package main

import (
	"bufio"
	"errors"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/GrGLeo/ctf/shared"
)

func main() {
	var wg sync.WaitGroup
	clientCount := 100
	duration := 1 * time.Minute

	// Start 1000 clients
	for i := 0; i < clientCount; i++ {
		wg.Add(1)
    go func(clientID int) {
			defer wg.Done()
			runClient(clientID, duration)
		}(i)

	}

	// Wait for all clients to finish
	wg.Wait()
	log.Println("All clients finished.")
}

func runClient(clientID int, duration time.Duration) {
	conn, err := net.Dial("tcp", "localhost:8082")
	if err != nil {
		log.Printf("Client %d: Connection failed: %v", clientID, err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Send LoginPacket
	login := shared.NewLoginPacket("testuser", "testpass")
	if _, err := conn.Write(login.Serialize()); err != nil {
		log.Printf("Client %d: Send login failed: %v", clientID, err)
		return
	}

	// Read Login Response (RespPacket code 1)
	if err := readPacket(reader, 1, 2); err != nil {
		log.Printf("Client %d: %v", clientID, err)
		return
	}

	// Send RoomRequestPacket
	roomReq := shared.NewRoomRequestPacket(2)
	if _, err := conn.Write(roomReq.Serialize()); err != nil {
		log.Printf("Client %d: Send room request failed: %v", clientID, err)
		return
	}

	// Wait for GameStartPacket
	if err := waitForGameStart(reader); err != nil {
		log.Printf("Client %d: %v", clientID, err)
		return
	}

	// Start sending actions for the specified duration
	actions := []int{1, 2, 3, 4}
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(duration)
	for {
		select {
		case <-ticker.C:
			action := actions[rand.Intn(len(actions))]
			actionPacket := shared.NewActionPacket(action)
			if _, err := conn.Write(actionPacket.Serialize()); err != nil {
				log.Printf("Client %d: Send action failed: %v", clientID, err)
				return
			}

      startTime := time.Now()
      if _, err := reader.Peek(1); err != nil {
				log.Printf("Client %d: No response received: %v", clientID, err)
				return
			}
      elapsedTime := time.Since(startTime)
      log.Printf("Client %d: Response time %d", clientID, elapsedTime.Milliseconds())

		case <-timeout:
			log.Printf("Client %d: Finished after %v", clientID, duration)
			return
		}
	}
}

func readPacket(reader *bufio.Reader, expectedCode, length int) error {
	buf := make([]byte, length)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return err
	}

	packet, err := shared.DeSerialize(buf)
	if err != nil {
		return err
	}

	if packet.Code() != expectedCode {
		return errors.New("unexpected packet code")
	}
	return nil
}

func waitForGameStart(reader *bufio.Reader) error {
	for {
		// Read header (version + code)
		header := make([]byte, 2)
		if _, err := io.ReadFull(reader, header); err != nil {
			return err
		}

		code := int(header[1])
		var data []byte

		switch code {
		case 3: // LookRoomPacket (3 bytes)
			data = append(header, readRemaining(reader, 1)...)
		case 4: // GameStartPacket (3 bytes)
			data = append(header, readRemaining(reader, 1)...)
		default:
			log.Printf("Received unexpected packet code: %d", code)
			continue
		}

		packet, err := shared.DeSerialize(data)
		if err != nil {
			return err
		}

		if gameStartPacket, ok := packet.(*shared.GameStartPacket); ok && gameStartPacket.Success == 0 {
			return nil
		}
	}
}

func readRemaining(reader *bufio.Reader, n int) []byte {
	buf := make([]byte, n)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return nil
	}
	return buf
}
