package main

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

type MessageManager struct {
	userToRoom   map[string]int
	userLock     sync.RWMutex
	roomToClient map[int]map[string]struct{}
	roomLock     sync.RWMutex
	logger       *slog.Logger
}

func NewMessageManager(logger *slog.Logger) *MessageManager {
	userToRoom := make(map[string]int)
	roomToClient := make(map[int]map[string]struct{})

	return &MessageManager{
		userToRoom:   userToRoom,
		userLock:     sync.RWMutex{},
		roomToClient: roomToClient,
		roomLock:     sync.RWMutex{},
		logger:       logger,
	}
}

func (mm *MessageManager) RegisterClient(client string, roomID int) error {
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
	mm.userLock.RLock()
	roomID, exists := mm.userToRoom[sender]
	mm.userLock.RUnlock()

	if !exists {
		return nil, "", fmt.Errorf("sender %s not registered", sender)
	}

	target, processedMessage := parseMessage(content, sender)

	mm.roomLock.RLock()
	roomClients := mm.roomToClient[roomID]
	mm.roomLock.RUnlock()

	if roomClients == nil {
		return nil, "", fmt.Errorf("room %d not found", roomID)
	}

	var receivers []string

	switch target {
	case "all":
		// Include all clients in room except sender
		receivers = make([]string, 0, len(roomClients)-1)
		for client := range roomClients {
			if client != sender {
				receivers = append(receivers, client)
			}
		}
	case "":
		// Regular room message - exclude sender
		receivers = make([]string, 0, len(roomClients)-1)
		for client := range roomClients {
			if client != sender {
				receivers = append(receivers, client)
			}
		}
	default:
		// Whisper to specific user
		if _, exists := roomClients[target]; exists {
			receivers = []string{target}
		} else {
			return nil, "", fmt.Errorf("target user %s not in room", target)
		}
	}

	return receivers, processedMessage, nil
}

func parseMessage(content, sender string) (string, string) {
	// First we extract the target of the message
	parts := strings.Fields(content)

	if strings.HasPrefix(parts[0], "/") {
		target := parts[0][1:] // We remove the initial "/"
		if target == "all" {
			return target, fmt.Sprintf("(all) %s: %s", sender, strings.Join(parts[1:], " "))
		} else {
			return target, fmt.Sprintf("(whisper) %s: %s", sender, strings.Join(parts[1:], " "))
		}
	}
	return "", fmt.Sprintf("(room) %s: %s", sender, strings.Join(parts[1:], " "))
}
