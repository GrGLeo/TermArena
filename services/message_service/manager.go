package main

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

type MessageManager struct {
	userToRoom     map[string]int
	userLock       sync.RWMutex
	roomToClient   map[int]map[string]struct{}
	roomLock       sync.RWMutex
	logger         *slog.Logger
	maxMessageSize int
}

func NewMessageManager(maxMessageSize int, logger *slog.Logger) *MessageManager {
	userToRoom := make(map[string]int)
	roomToClient := make(map[int]map[string]struct{})

	return &MessageManager{
		userToRoom:     userToRoom,
		userLock:       sync.RWMutex{},
		roomToClient:   roomToClient,
		roomLock:       sync.RWMutex{},
		logger:         logger,
		maxMessageSize: maxMessageSize,
	}
}

func (mm *MessageManager) RegisterClient(client string, roomID int) error {
	// Input validation
	client = strings.TrimSpace(client)
	if client == "" {
		return fmt.Errorf("client cannot be empty")
	}

	mm.userLock.RLock()
	oldRoomID, exist := mm.userToRoom[client]
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
		delete(mm.roomToClient[oldRoomID], client)

		// create the room if it doesnt exist
		if mm.roomToClient[roomID] == nil {
			mm.roomToClient[roomID] = make(map[string]struct{})
		}
		mm.roomToClient[roomID][client] = struct{}{}

		mm.logger.Info("Client switched room", "client", client, "old_roomID", oldRoomID, "roomID", roomID)
		return nil
	}

	// If the user doesnt exist with adapt both map
	mm.userLock.Lock()
	defer mm.userLock.Unlock()

	mm.roomLock.Lock()
	defer mm.roomLock.Unlock()

	mm.userToRoom[client] = roomID

	// create the room if it doesnt exist
	if mm.roomToClient[roomID] == nil {
		mm.roomToClient[roomID] = make(map[string]struct{})
	}
	mm.roomToClient[roomID][client] = struct{}{}

	mm.logger.Info("Client registered", "client", client, "roomID", roomID)
	return nil

}

func (mm *MessageManager) UnregisterClient(client string) error {
	mm.userLock.Lock()
	defer mm.userLock.Unlock()

	mm.roomLock.Lock()
	defer mm.roomLock.Unlock()

	if roomID, exist := mm.userToRoom[client]; exist {
		delete(mm.userToRoom, client)
		delete(mm.roomToClient[roomID], client)

		if len(mm.roomToClient[roomID]) == 0 {
			delete(mm.roomToClient, roomID)
		}
	} else {
		mm.logger.Warn("Client to unregister not found", "client", client)
		return fmt.Errorf("Failed to find client %s to unregister", client)
	}
	mm.logger.Info("Client unregister", "client", client)

	return nil
}

func (mm *MessageManager) RouteMessage(sender string, content string) ([]string, string, error) {
	mm.logger.Info("[MESSAGE MANAGER] RouteMessage called", "sender", sender, "content", content)

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
	mm.userLock.RUnlock()

	if !exists {
		mm.logger.Error("[MESSAGE MANAGER] Sender not registered", "sender", sender)
		return nil, "", fmt.Errorf("sender %s not registered", sender)
	}

	mm.logger.Info("[MESSAGE MANAGER] Sender validated", "sender", sender, "room_id", roomID)

	target, processedMessage := parseMessage(content, sender)
	mm.logger.Info("[MESSAGE MANAGER] Message parsed", "original_content", content, "target", target, "processed_message", processedMessage)

	mm.roomLock.RLock()
	roomClients := mm.roomToClient[roomID]
	mm.roomLock.RUnlock()

	if roomClients == nil {
		mm.logger.Error("[MESSAGE MANAGER] Room not found", "room_id", roomID)
		return nil, "", fmt.Errorf("room %d not found", roomID)
	}

	mm.logger.Info("[MESSAGE MANAGER] Room found", "room_id", roomID, "clients_in_room", len(roomClients))

	var receivers []string

	switch target {
	case "all":
		// Include all clients in room except sender
		receivers = make([]string, 0, len(roomClients)-1)
		for client := range roomClients {
			receivers = append(receivers, client)
		}
		mm.logger.Info("[MESSAGE MANAGER] Broadcasting to all in room", "sender", sender, "receivers", receivers)
	case "":
		// Regular room message - exclude sender
		receivers = make([]string, 0, len(roomClients)-1)
		for client := range roomClients {
			receivers = append(receivers, client)
		}
		mm.logger.Info("[MESSAGE MANAGER] Room message", "sender", sender, "receivers", receivers)
	default:
		// Whisper to specific user
		if _, exists := roomClients[target]; exists {
			receivers = []string{target}
			mm.logger.Info("[MESSAGE MANAGER] Whisper message", "sender", sender, "target", target)
		} else {
			mm.logger.Error("[MESSAGE MANAGER] Target user not in room", "target", target, "room_id", roomID)
			return nil, "", fmt.Errorf("target user %s not in room", target)
		}
	}

	mm.logger.Info("[MESSAGE MANAGER] RouteMessage completed", "sender", sender, "receivers_count", len(receivers), "final_message", processedMessage)
	return receivers, processedMessage, nil
}

func parseMessage(content, sender string) (string, string) {
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
	return "", fmt.Sprintf("(room) %s: %s", sender, content)
}
