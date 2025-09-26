package main

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

type MessageManager struct {
	userToRoom     map[string]uint32
	userToTeam     map[string]int
	userLock       sync.RWMutex
	roomToClient   map[uint32]map[int]map[string]struct{}
	roomLock       sync.RWMutex
	logger         *slog.Logger
	maxMessageSize int
}

func NewMessageManager(maxMessageSize int, logger *slog.Logger) *MessageManager {
	userToRoom := make(map[string]uint32)
	userToTeam := make(map[string]int)
	roomToClient := make(map[uint32]map[int]map[string]struct{})

	return &MessageManager{
		userToRoom:     userToRoom,
		userToTeam:     userToTeam,
		userLock:       sync.RWMutex{},
		roomToClient:   roomToClient,
		roomLock:       sync.RWMutex{},
		logger:         logger,
		maxMessageSize: maxMessageSize,
	}
}

func (mm *MessageManager) RegisterClient(client string, roomID uint32, teamID int) error {
	// Input validation
	client = strings.TrimSpace(client)
	if client == "" {
		return fmt.Errorf("client cannot be empty")
	}

	mm.userLock.RLock()
	oldRoomID, exist := mm.userToRoom[client]
	oldTeamID := mm.userToTeam[client]
	mm.userLock.RUnlock()

	// In this case we do not need to modify the client room
	if oldRoomID == roomID && exist {
		mm.logger.Warn("Client already registered in the room", "client", client, "roomID", roomID)
		return fmt.Errorf("client %s is already in the room %d", client, roomID)
	}

	// If the user exist, and change room, we need to change the userToRoom to map the correct room
	// And we need to clean the past room and update the new room
	if exist {
		mm.userLock.Lock()
		defer mm.userLock.Unlock()

		mm.roomLock.Lock()
		defer mm.roomLock.Unlock()

		// past room is updated
		mm.userToRoom[client] = roomID
		mm.userToTeam[client] = teamID
		delete(mm.roomToClient[oldRoomID][oldTeamID], client)

		// create the room/team if it doesnt exist
		if mm.roomToClient[roomID] == nil {
			mm.logger.Info("info", "roomID", roomID)
			mm.roomToClient[roomID] = make(map[int]map[string]struct{})
		}
		if mm.roomToClient[roomID][teamID] == nil {
			mm.logger.Info("info", "roomID", roomID, "teamID", teamID)
			mm.roomToClient[roomID][teamID] = make(map[string]struct{})
		}
		mm.roomToClient[roomID][teamID][client] = struct{}{}

		mm.logger.Debug("Client switched room", "client", client, "old_roomID", oldRoomID, "roomID", roomID, "old_TeamID", oldTeamID, "teamID", teamID)
		return nil
	}

	// If the user doesnt exist we adapt both map
	mm.userLock.Lock()
	defer mm.userLock.Unlock()

	mm.roomLock.Lock()
	defer mm.roomLock.Unlock()

	mm.userToRoom[client] = roomID
	mm.userToTeam[client] = teamID

	// create the room if it doesnt exist
	// when a user doesnt exist he should have team 0 by default
	// no need to check teamID map
	if mm.roomToClient[roomID][teamID] == nil {
		mm.roomToClient[roomID] = make(map[int]map[string]struct{})
		mm.roomToClient[roomID][teamID] = make(map[string]struct{})
	}
	mm.logger.Warn("info", "roomID", roomID, "teamID", teamID, "client", client)
	mm.roomToClient[roomID][teamID][client] = struct{}{}

	mm.logger.Debug("Client registered", "client", client, "roomID", roomID, "teamID", teamID)
	return nil

}

func (mm *MessageManager) UnregisterClient(client string) error {
	mm.userLock.Lock()
	defer mm.userLock.Unlock()

	mm.roomLock.Lock()
	defer mm.roomLock.Unlock()

	if roomID, exist := mm.userToRoom[client]; exist {
		if teamID, exist := mm.userToTeam[client]; exist {
			delete(mm.userToRoom, client)
			delete(mm.userToTeam, client)
			delete(mm.roomToClient[roomID][teamID], client)

			if len(mm.roomToClient[roomID]) == 0 {
				delete(mm.roomToClient, roomID)
			}
		} else {
			mm.logger.Warn("Client to unregister not found", "client", client)
			return fmt.Errorf("Failed to find client %s to unregister", client)
		}
	}
	mm.logger.Debug("Client unregister", "client", client)

	return nil
}

func (mm *MessageManager) RouteMessage(sender string, content string) ([]string, string, error) {
	mm.logger.Debug(" RouteMessage called", "sender", sender, "content", content)

	// Validation step
	sender = strings.TrimSpace(sender)
	content = strings.TrimSpace(content)

	if sender == "" {
		return nil, "", fmt.Errorf("sender cannot be empty")
	}
	if content == "" {
		return nil, "", fmt.Errorf("content cannot be empty")
	}

	if len(content) > mm.maxMessageSize {
		return nil, "", fmt.Errorf("message too long (max message size: %d)", mm.maxMessageSize)
	}

	mm.userLock.RLock()
	roomID, exists := mm.userToRoom[sender]
	teamID := mm.userToTeam[sender] // when roomID is set teamID should be set
	mm.userLock.RUnlock()

	if !exists {
		mm.logger.Error("Sender not registered", "sender", sender)
		return nil, "", fmt.Errorf("sender %s not registered", sender)
	}

	mm.logger.Debug("Sender validated", "sender", sender, "room_id", roomID)

	var target string
	var processedMessage string
	if roomID == 0 {
		target, processedMessage = parseMessage(content, sender, true)
		mm.logger.Debug("Message parsed", "original_content", content, "target", target, "processed_message", processedMessage)
	} else {
		target, processedMessage = parseMessage(content, sender, false)
	}

	mm.roomLock.RLock()
	roomClients := mm.roomToClient[roomID]
	mm.roomLock.RUnlock()

	if roomClients == nil {
		mm.logger.Error("Room not found", "room_id", roomID)
		return nil, "", fmt.Errorf("room %d not found", roomID)
	}

	mm.logger.Debug("Room found", "room_id", roomID, "clients_in_room", len(roomClients))

	var receivers []string

	switch target {
	case "all":
		// Include all clients in room
		receivers = make([]string, 0, len(roomClients)-1)
		for _, team := range roomClients {
			for client := range team {
				receivers = append(receivers, client)
			}
		}
		mm.logger.Debug("Broadcasting to all in room", "sender", sender, "receivers", receivers)
	case "":
		if roomID == 0 {
			// Room message, for the general lobby
			receivers = make([]string, 0, len(roomClients)-1)
			for _, team := range roomClients {
				for client := range team {
					receivers = append(receivers, client)
				}
			}
		} else {
			// Team message outside of general lobby
			team := roomClients[teamID]
			for client := range team {
				receivers = append(receivers, client)
			}
		}

		mm.logger.Debug("Room message", "sender", sender, "receivers", receivers)
	default:
		// Whisper to specific user
		if _, exists := mm.userToRoom[target]; exists {
			receivers = []string{target}
			mm.logger.Debug("Whisper message", "sender", sender, "target", target)
		} else {
			mm.logger.Error("Target user not online", "target", target)
			return nil, "", fmt.Errorf("target user %s not in room", target)
		}
	}

	mm.logger.Debug("RouteMessage completed", "sender", sender, "receivers_count", len(receivers), "final_message", processedMessage)
	return receivers, processedMessage, nil
}

func parseMessage(content, sender string, isLooby bool) (string, string) {
	// First we extract the target of the message
	parts := strings.Fields(content)

	// Validation
	if len(parts) == 0 {
		return "", fmt.Sprintf("(room) %s: %s", sender, content)
	}

	if strings.HasPrefix(parts[0], "/") {
		target := parts[0][1:] // We remove the initial "/"
		messageContent := ""
		if len(parts) > 1 {
			messageContent = strings.Join(parts[1:], " ")
		}
		if target == "all" {
			return target, fmt.Sprintf("(all) %s: %s", sender, messageContent)
		} else {
			return target, fmt.Sprintf("(whisper) %s: %s", sender, messageContent)
		}
	}
	if isLooby {
		return "", fmt.Sprintf("(room) %s: %s", sender, content)
	} else {
		return "", fmt.Sprintf("(team) %s: %s", sender, content)
	}
}
