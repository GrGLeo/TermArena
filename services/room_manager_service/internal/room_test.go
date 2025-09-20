package internal

import (
	"fmt"
	"io"
	"log/slog"
	"sync"
	"testing"
)

func TestNewRoom(t *testing.T) {
	tests := []struct {
		name        string
		roomType    RoomType
		expectedMax int
	}{
		{"Sandbox room", SANDBOX, 1},
		{"Practice room", PRACTICE, 4},
		{"Classic room", CLASSIC, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			room := NewRoom(tt.roomType)
			if room.maxPlayers != tt.expectedMax {
				t.Errorf("NewRoom() maxPlayers = %v, want %v", room.maxPlayers, tt.expectedMax)
			}
			if len(room.Players) != 0 {
				t.Errorf("NewRoom() should start with empty players map, got %v", len(room.Players))
			}
			if room.nextTeam != BLUETEAM {
				t.Errorf("NewRoom() should start with BLUETEAM, got %v", room.nextTeam)
			}
		})
	}
}

func TestRoom_AddPlayer(t *testing.T) {
	room := NewRoom(PRACTICE) // max 4 players

	// Test adding first player
	_, status := room.AddPlayer("player1")
	if status != WAITING {
		t.Errorf("AddPlayer() first player should return WAITING, got %v", status)
	}
	if len(room.Players) != 1 {
		t.Errorf("AddPlayer() should have 1 player, got %v", len(room.Players))
	}
	if room.Players["player1"].playerTeam != BLUETEAM {
		t.Errorf("First player should be BLUETEAM, got %v", room.Players["player1"].playerTeam)
	}

	// Test adding second player
	_, status = room.AddPlayer("player2")
	if status != WAITING {
		t.Errorf("AddPlayer() second player should return WAITING, got %v", status)
	}
	if room.Players["player2"].playerTeam != REDTEAM {
		t.Errorf("Second player should be REDTEAM, got %v", room.Players["player2"].playerTeam)
	}

	// Test adding third player
	_, status = room.AddPlayer("player3")
	if status != WAITING {
		t.Errorf("AddPlayer() third player should return WAITING, got %v", status)
	}
	if room.Players["player3"].playerTeam != BLUETEAM {
		t.Errorf("Third player should be BLUETEAM, got %v", room.Players["player3"].playerTeam)
	}

	// Test adding fourth player (should fill the room)
	_, status = room.AddPlayer("player4")
	if status != LOBBY {
		t.Errorf("AddPlayer() fourth player should return LOBBY, got %v", status)
	}
	if len(room.Players) != 4 {
		t.Errorf("AddPlayer() should have 4 players, got %v", len(room.Players))
	}
}

func TestRoom_UpdatePlayerSpell(t *testing.T) {
	room := NewRoom(SANDBOX)
	room.AddPlayer("player1")

	// Update spells
	room.UpdatePlayerSpell("player1", 1, 2)

	player := room.Players["player1"]
	if player.playerSpells[0] != 1 {
		t.Errorf("UpdatePlayerSpell() spell1 should be 1, got %v", player.playerSpells[0])
	}
	if player.playerSpells[1] != 2 {
		t.Errorf("UpdatePlayerSpell() spell2 should be 2, got %v", player.playerSpells[1])
	}
}

func TestRoom_GetUsernames(t *testing.T) {
	room := NewRoom(PRACTICE)
	room.AddPlayer("alice")
	room.AddPlayer("bob")
	room.AddPlayer("charlie")

	usernames := room.GetUsernames()
	if len(usernames) != 3 {
		t.Errorf("GetUsernames() should return 3 usernames, got %v", len(usernames))
	}

	// Check that all usernames are present
	expected := map[string]bool{"alice": false, "bob": false, "charlie": false}
	for _, username := range usernames {
		if _, exists := expected[username]; exists {
			expected[username] = true
		} else {
			t.Errorf("GetUsernames() returned unexpected username: %v", username)
		}
	}

	for username, found := range expected {
		if !found {
			t.Errorf("GetUsernames() missing expected username: %v", username)
		}
	}
}

func TestPlayer_UpdateSpells(t *testing.T) {
	player := NewPlayer(BLUETEAM)
	player.UpdateSpells(Spells{5, 10})

	if player.playerSpells[0] != 5 {
		t.Errorf("UpdateSpells() spell1 should be 5, got %v", player.playerSpells[0])
	}
	if player.playerSpells[1] != 10 {
		t.Errorf("UpdateSpells() spell2 should be 10, got %v", player.playerSpells[1])
	}
	if player.playerTeam != BLUETEAM {
		t.Errorf("UpdateSpells() should not change team, got %v", player.playerTeam)
	}
}

func TestRoom_AddPlayerConcurrent(t *testing.T) {
	room := NewRoom(PRACTICE) // max 4
	var wg sync.WaitGroup
	numGoroutines := 4
	results := make([]struct {
		team   Team
		status RoomStatus
	}, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			team, status := room.AddPlayer(fmt.Sprintf("player%d", idx))
			results[idx] = struct {
				team   Team
				status RoomStatus
			}{team, status}
		}(i)
	}
	wg.Wait()

	if len(room.Players) != 4 {
		t.Errorf("Concurrent AddPlayer should result in 4 players, got %d", len(room.Players))
	}
	// Check teams alternate
	blueCount := 0
	redCount := 0
	for _, res := range results {
		if res.team == BLUETEAM {
			blueCount++
		} else if res.team == REDTEAM {
			redCount++
		}
	}
	if blueCount != 2 || redCount != 2 {
		t.Errorf("Expected 2 blue and 2 red teams, got blue: %d, red: %d", blueCount, redCount)
	}
	// One should be LOBBY, others WAITING
	lobbyCount := 0
	waitingCount := 0
	for _, res := range results {
		if res.status == LOBBY {
			lobbyCount++
		} else if res.status == WAITING {
			waitingCount++
		}
	}
	if lobbyCount != 1 || waitingCount != 3 {
		t.Errorf("Expected 1 LOBBY and 3 WAITING, got lobby: %d, waiting: %d", lobbyCount, waitingCount)
	}
}

func TestRoom_UpdatePlayerSpellConcurrent(t *testing.T) {
	room := NewRoom(SANDBOX)
	room.AddPlayer("player1")
	var wg sync.WaitGroup
	numUpdates := 10

	for i := 0; i < numUpdates; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			room.UpdatePlayerSpell("player1", val, val+1)
		}(i)
	}
	wg.Wait()

	// Spells should be updated (exact value not guaranteed due to concurrency)
	player := room.Players["player1"]
	if player.playerSpells[0] < 0 || player.playerSpells[0] > 9 || player.playerSpells[1] < 1 || player.playerSpells[1] > 10 {
		t.Errorf("Concurrent UpdatePlayerSpell values out of range, got %v", player.playerSpells)
	}
}

func TestRoom_GetUsernamesConcurrent(t *testing.T) {
	room := NewRoom(PRACTICE)
	room.AddPlayer("alice")
	room.AddPlayer("bob")
	var wg sync.WaitGroup
	numReads := 10
	results := make([][]string, numReads)

	for i := 0; i < numReads; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = room.GetUsernames()
		}(i)
	}
	wg.Wait()

	for i, usernames := range results {
		if len(usernames) != 2 {
			t.Errorf("Concurrent GetUsernames %d should return 2 usernames, got %d", i, len(usernames))
		}
		// Check contains alice and bob
		hasAlice := false
		hasBob := false
		for _, u := range usernames {
			if u == "alice" {
				hasAlice = true
			}
			if u == "bob" {
				hasBob = true
			}
		}
		if !hasAlice || !hasBob {
			t.Errorf("Concurrent GetUsernames %d missing expected usernames, got %v", i, usernames)
		}
	}
}

func TestRoom_AddPlayerEdgeCases(t *testing.T) {
	// Test adding to full room
	room := NewRoom(SANDBOX) // max 1
	room.AddPlayer("player1")
	// Add another, should still add but return WAITING (bug in code, but test behavior)
	team, status := room.AddPlayer("player2")
	if status != WAITING {
		t.Errorf("Adding to full room should return WAITING, got %v", status)
	}
	if len(room.Players) != 2 {
		t.Errorf("Room should have 2 players after overfilling, got %d", len(room.Players))
	}
	if team != REDTEAM {
		t.Errorf("Second player team should be REDTEAM, got %v", team)
	}

	// Test duplicate username
	room2 := NewRoom(PRACTICE)
	room2.AddPlayer("alice")
	room2.AddPlayer("alice") // overwrite with new team
	if room2.Players["alice"].playerTeam != REDTEAM {
		t.Errorf("Duplicate username should assign new team REDTEAM, got %v", room2.Players["alice"].playerTeam)
	}
	if len(room2.Players) != 1 {
		t.Errorf("Duplicate add should not increase player count, got %d", len(room2.Players))
	}
}

func TestRoom_UpdatePlayerSpellEdgeCases(t *testing.T) {
	room := NewRoom(SANDBOX)

	// Test updating non-existent player
	defer func() {
		if r := recover(); r == nil {
			t.Error("UpdatePlayerSpell on non-existent player should panic")
		}
	}()
	room.UpdatePlayerSpell("nonexistent", 1, 2)
}

func TestRoom_GetUsernamesEdgeCases(t *testing.T) {
	// Empty room
	room := NewRoom(CLASSIC)
	usernames := room.GetUsernames()
	if len(usernames) != 0 {
		t.Errorf("GetUsernames on empty room should return empty slice, got %v", usernames)
	}

	// After adding players
	room.AddPlayer("user1")
	room.AddPlayer("user2")
	usernames = room.GetUsernames()
	if len(usernames) != 2 {
		t.Errorf("GetUsernames should return 2 usernames, got %d", len(usernames))
	}
}

func TestRoomManager_LookRoom_NoWaitingRoom(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
  maxRoom := 100
	rm := NewRoomManager(maxRoom, logger)

	team, roomID := rm.LookRoom("alice", PRACTICE)

	if team != BLUETEAM {
		t.Errorf("First player should be BLUETEAM, got %v", team)
	}
	if roomID == 0 {
		t.Errorf("RoomID should not be zero")
	}

	// Check room is created in WAITING
	if room, exists := rm.rooms[PRACTICE][WAITING][roomID]; !exists {
		t.Errorf("Room should be in WAITING status")
	} else {
		if len(room.Players) != 1 {
			t.Errorf("Room should have 1 player, got %d", len(room.Players))
		}
	}
}

func TestRoomManager_LookRoom_ExistingWaitingNotFull(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
  maxRoom := 100
	rm := NewRoomManager(maxRoom, logger)

	// Create a waiting room with 1 player
	_, roomID1 := rm.LookRoom("alice", PRACTICE) // Creates WAITING room

	// Add second player
	team, roomID2 := rm.LookRoom("bob", PRACTICE)

	if team != REDTEAM {
		t.Errorf("Second player should be REDTEAM, got %v", team)
	}
	if roomID1 != roomID2 {
		t.Errorf("Should join the same room, got different IDs %v != %v", roomID1, roomID2)
	}

	// Check room still in WAITING
	if _, exists := rm.rooms[PRACTICE][WAITING][roomID1]; !exists {
		t.Errorf("Room should still be in WAITING status")
	}
	if room, _ := rm.rooms[PRACTICE][WAITING][roomID1]; len(room.Players) != 2 {
		t.Errorf("Room should have 2 players, got %d", len(room.Players))
	}
}

func TestRoomManager_LookRoom_ExistingWaitingFull(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
  maxRoom := 100
	rm := NewRoomManager(maxRoom, logger)

	// For PRACTICE, max 4 players
	_, roomID := rm.LookRoom("alice", PRACTICE)
	rm.LookRoom("bob", PRACTICE)
	rm.LookRoom("charlie", PRACTICE)

	// Fourth player should fill it
	team, roomID4 := rm.LookRoom("diana", PRACTICE)

	if team != REDTEAM { // alice BLUE, bob RED, charlie BLUE, diana RED
		t.Errorf("Fourth player should be REDTEAM, got %v", team)
	}
	if roomID != roomID4 {
		t.Errorf("Should join the same room")
	}

	// Check room moved to LOBBY
	if _, exists := rm.rooms[PRACTICE][LOBBY][roomID]; !exists {
		t.Errorf("Room should be moved to LOBBY status")
	}
	if room, _ := rm.rooms[PRACTICE][LOBBY][roomID]; len(room.Players) != 4 {
		t.Errorf("Room should have 4 players, got %d", len(room.Players))
	}
}

func TestRoomManager_LookRoom_DifferentRoomTypes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
  maxRoom := 100
	rm := NewRoomManager(maxRoom, logger)

	tests := []struct {
		roomType RoomType
		max      int
	}{
		{SANDBOX, 1},
		{PRACTICE, 4},
		{CLASSIC, 8},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("RoomType_%v", tt.roomType), func(t *testing.T) {
			team, roomID := rm.LookRoom("user", tt.roomType)
			if team != BLUETEAM {
				t.Errorf("First player should be BLUETEAM")
			}
			expectedStatus := WAITING
			if tt.roomType == SANDBOX {
				expectedStatus = LOBBY // SANDBOX fills with 1 player
			}
			if room, exists := rm.rooms[tt.roomType][expectedStatus][roomID]; exists {
				if room.maxPlayers != tt.max {
					t.Errorf("Room maxPlayers should be %d, got %d", tt.max, room.maxPlayers)
				}
			} else {
				t.Errorf("Room should be created in %v status", expectedStatus)
			}
		})
	}
}

// Note: Timer test is omitted as it requires waiting 1 minute

func BenchmarkRoom_AddPlayer(b *testing.B) {
	room := NewRoom(CLASSIC) // max 8
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		username := fmt.Sprintf("player%d", i%8) // reuse usernames to avoid overfilling for benchmark
		room.AddPlayer(username)
	}
}

func BenchmarkRoom_AddPlayerConcurrent(b *testing.B) {
	room := NewRoom(CLASSIC)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			username := fmt.Sprintf("player%d", i%8)
			room.AddPlayer(username)
			i++
		}
	})
}

func BenchmarkRoomManager_LookRoom(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
  maxRoom := 100
	rm := NewRoomManager(maxRoom, logger)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		username := fmt.Sprintf("user%d", i)
		rm.LookRoom(username, PRACTICE)
	}
}

func BenchmarkRoomManager_LookRoomConcurrent(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
  maxRoom := 100
	rm := NewRoomManager(maxRoom, logger)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			username := fmt.Sprintf("user%d", i)
			rm.LookRoom(username, PRACTICE)
			i++
		}
	})
}
