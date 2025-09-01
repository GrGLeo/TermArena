package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GrGLeo/ctf/pkg/shared"
)

type ClientStats struct {
	ClientID         int
	ActionsSent      int64
	PacketsReceived  int64
	ConnectionErrors int64
	LoginErrors      int64
	RoomErrors       int64
	GameServerErrors int64
	ActionErrors     int64
	ReadErrors       int64
	WriteErrors      int64
	DecodeErrors     int64
	TotalLatency     time.Duration
	MaxLatency       time.Duration
	MinLatency       time.Duration
	LatencyCount     int64
}

func main() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: go run main.go <client_count> <server_port> <duration_in_seconds> <game_type>")
		return
	}

	clientCount, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("Invalid client count")
		return
	}

	serverPort := os.Args[2]

	duration, err := time.ParseDuration(os.Args[3] + "s")
	if err != nil {
		fmt.Println("Invalid duration")
		return
	}

	gameType, err := strconv.Atoi(os.Args[4])
	if err != nil {
		fmt.Println("Invalid game type")
		return
	}

	stats := make([]*ClientStats, clientCount)
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	for i := range clientCount {
		wg.Add(1)
		stats[i] = &ClientStats{ClientID: i, MinLatency: time.Hour}
		go runClient(ctx, &wg, i, serverPort, stats[i], gameType)
	}

	log.Printf("Simulation started for %v.", duration)
	time.Sleep(duration)
	cancel()
	log.Println("Simulation finished. Waiting for clients to exit...")

	wg.Wait()

	printSummary(stats)
}

func printSummary(stats []*ClientStats) {
	var totalActions, totalPackets, totalErrors, totalLatencyCount int64
	var totalLatency time.Duration

	log.Println("--- Simulation Summary ---")
	for _, s := range stats {
		totalActions += s.ActionsSent
		totalPackets += s.PacketsReceived
		totalErrors += s.ConnectionErrors + s.LoginErrors + s.RoomErrors + s.GameServerErrors + s.ActionErrors + s.ReadErrors + s.WriteErrors + s.DecodeErrors
		totalLatency += s.TotalLatency
		totalLatencyCount += s.LatencyCount

		avgLatency := time.Duration(0)
		if s.LatencyCount > 0 {
			avgLatency = s.TotalLatency / time.Duration(s.LatencyCount)
		}
		log.Printf("Client %d: Actions: %d, Packets Rcvd: %d, Errors: %d, Avg Latency: %v, Min Latency: %v, Max Latency: %v",
			s.ClientID, s.ActionsSent, s.PacketsReceived, s.ConnectionErrors+s.LoginErrors+s.RoomErrors+s.GameServerErrors+s.ActionErrors+s.ReadErrors+s.WriteErrors+s.DecodeErrors, avgLatency, s.MinLatency, s.MaxLatency)
	}

	avgTotalLatency := time.Duration(0)
	if totalLatencyCount > 0 {
		avgTotalLatency = totalLatency / time.Duration(totalLatencyCount)
	}
	log.Println("--- Global ---")
	log.Printf("Total Actions: %d, Total Packets Rcvd: %d, Total Errors: %d, Average Latency: %v",
		totalActions, totalPackets, totalErrors, avgTotalLatency)
}

func runClient(ctx context.Context, wg *sync.WaitGroup, clientID int, serverPort string, stats *ClientStats, gameType int) {
	defer wg.Done()

	serverIP := os.Getenv("SERVER_IP")
	if len(os.Args) > 5 {
		serverIP = os.Args[5]
	} else if serverIP == "" {
		serverIP = "localhost"
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", serverIP, serverPort))
	if err != nil {
		log.Printf("Client %d: Failed to connect to server: %v", clientID, err)
		stats.ConnectionErrors++
		return
	}
	defer conn.Close()

	log.Printf("Client %d: Connected to server", clientID)

	// 1. Send Login Packet
	loginPacket := shared.NewLoginChallengeRequestPacket("simulation_user")
	_, err = conn.Write(loginPacket.Serialize())
	if err != nil {
		log.Printf("Client %d: Failed to send login packet: %v", clientID, err)
		stats.WriteErrors++
		return
	}

	// 2. Receive Login Response
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		log.Printf("Client %d: Failed to read login response: %v", clientID, err)
		stats.ReadErrors++
		return
	}

	packet, _, err := shared.DeSerialize(buf[:n])
	if err != nil {
		log.Printf("Client %d: Failed to deserialize login response: %v", clientID, err)
		stats.DecodeErrors++
		return
	}

	if _, ok := packet.(*shared.LoginChallengeResponsePacket); ok {
		log.Printf("Client %d: Login successful", clientID)
	} else {
		log.Printf("Client %d: Login failed", clientID)
		stats.LoginErrors++
		return
	}

	// 3. Send Room Request Packet
	roomRequestPacket := shared.NewRoomRequestPacket(gameType)
	_, err = conn.Write(roomRequestPacket.Serialize())
	if err != nil {
		log.Printf("Client %d: Failed to send room request packet: %v", clientID, err)
		stats.WriteErrors++
		return
	}

	// 4. Receive Room Info
	n, err = conn.Read(buf)
	if err != nil {
		log.Printf("Client %d: Failed to read room info: %v", clientID, err)
		stats.ReadErrors++
		return
	}

	packet, _, err = shared.DeSerialize(buf[:n])
	if err != nil {
		log.Printf("Client %d: Failed to deserialize room info: %v", clientID, err)
		stats.DecodeErrors++
		return
	}

	lookRoomPacket, ok := packet.(*shared.LookRoomPacket)
	if !ok || lookRoomPacket.Success != 0 {
		log.Printf("Client %d: Failed to get room info", clientID)
		stats.RoomErrors++
		return
	}

	log.Printf("Client %d: Received room info: %s", clientID, lookRoomPacket.RoomIP)

	// 5. Connect to Game Server
	gameConn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", serverIP, lookRoomPacket.RoomIP))
	if err != nil {
		log.Printf("Client %d: Failed to connect to game server: %v", clientID, err)
		stats.GameServerErrors++
		return
	}
	defer gameConn.Close()

	log.Printf("Client %d: Connected to game server", clientID)

	// 6. Send Spell Selection Packet
	spellSelectionPacket := shared.NewSpellSelectionPacket(0, 1)
	_, err = gameConn.Write(spellSelectionPacket.Serialize())
	if err != nil {
		log.Printf("Client %d: Failed to send spell selection packet: %v", clientID, err)
		stats.WriteErrors++
		return
	}

	// 7. Wait for GameStartPacket
	log.Printf("Client %d: Waiting for GameStartPacket...", clientID)
	gameBuf := make([]byte, 0, 4096)
	tempBuf := make([]byte, 2048)
	var gameStarted bool
gameStartLoop:
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := gameConn.Read(tempBuf)
			if err != nil {
				log.Printf("Client %d: Failed to read from game server while waiting for start: %v", clientID, err)
				stats.ReadErrors++
				return
			}
			gameBuf = append(gameBuf, tempBuf[:n]...)

			if len(gameBuf) >= 3 && gameBuf[1] == 10 {
				log.Printf("Client %d: Received GameStartPacket. Starting to send actions.", clientID)
				gameStarted = true
				gameBuf = gameBuf[3:]
				break gameStartLoop
			} else if len(gameBuf) >= 3 {
				log.Printf("Client %d: Received unexpected packet with code %d while waiting for start. Discarding buffer.", clientID, gameBuf[1])
				gameBuf = gameBuf[:0]
			}
		}
	}

	if !gameStarted {
		log.Printf("Client %d: Did not receive GameStartPacket. Exiting.", clientID)
		return
	}

	// 8. Send Random Actions and Read Board Packets
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			action := rand.Intn(4) + 1 // 1 to 4
			actionPacket := shared.NewActionPacket(action)
			startTime := time.Now()
			_, err := gameConn.Write(actionPacket.Serialize())
			if err != nil {
				log.Printf("Client %d: Failed to send action packet: %v", clientID, err)
				stats.ActionErrors++
				continue
			}
			atomic.AddInt64(&stats.ActionsSent, 1)

			n, err := gameConn.Read(tempBuf)
			if err != nil {
				log.Printf("Client %d: Failed to read from game server: %v", clientID, err)
				stats.ReadErrors++
				continue
			}
			if n > 0 {
				gameBuf = append(gameBuf, tempBuf[:n]...)
			}

			for {
				packet, bytesConsumed, err := shared.DeSerialize(gameBuf)
				if err != nil {
					if err.Error() == "incomplete packet header" || err.Error() == "incomplete packet" {
						break
					} else {
						log.Printf("Client %d: Error deserializing packet: %v", clientID, err)
						stats.DecodeErrors++
						gameBuf = nil
						continue
					}
				}

				gameBuf = gameBuf[bytesConsumed:]

				if _, ok := packet.(*shared.BoardPacket); ok {
					latency := time.Since(startTime)
					stats.TotalLatency += latency
					if latency > stats.MaxLatency {
						stats.MaxLatency = latency
					}
					if latency < stats.MinLatency {
						stats.MinLatency = latency
					}
					stats.LatencyCount++
					atomic.AddInt64(&stats.PacketsReceived, 1)
				} else {
					log.Printf("Client %d: Did not receive BoardPacket, but got %T", clientID, packet)
				}
			}
		}
	}
}
