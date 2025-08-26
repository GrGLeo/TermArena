package shared

import (
	"testing"
)

func TestMessagePacket_Serialize(t *testing.T) {
	tests := []struct {
		name    string
		sender  string
		message string
	}{
		{"empty sender and message", "", ""},
		{"simple sender and message", "Alice", "hello"},
		{"sender with spaces", "Alice Bob", "hello world"},
		{"message with special characters", "Alice", "hello\nworld\t!"},
		{"unicode sender and message", "Álice", "héllo wörld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet := NewMessagePacket(tt.sender, tt.message)
			result := packet.Serialize()

			// Check basic structure
			if len(result) < 6 {
				t.Errorf("Serialize() too short: %d bytes", len(result))
				return
			}

			// Check version and code
			if result[0] != 1 {
				t.Errorf("Version byte = %d, expected 1", result[0])
			}
			if result[1] != 100 {
				t.Errorf("Code byte = %d, expected 100", result[1])
			}

			// Check sender length field
			senderLen := int(result[2])<<8 | int(result[3])
			if senderLen != len(tt.sender) {
				t.Errorf("Sender length field = %d, expected %d", senderLen, len(tt.sender))
			}

			// Check sender content
			if len(result) > 4 {
				actualSender := string(result[4 : 4+senderLen])
				if actualSender != tt.sender {
					t.Errorf("Sender content = %q, expected %q", actualSender, tt.sender)
				}
			}

			// Check message length field
			messageLenStart := 4 + senderLen
			if len(result) < messageLenStart+2 {
				t.Errorf("Serialize() too short for message length: %d bytes", len(result))
				return
			}
			messageLen := int(result[messageLenStart])<<8 | int(result[messageLenStart+1])
			if messageLen != len(tt.message) {
				t.Errorf("Message length field = %d, expected %d", messageLen, len(tt.message))
			}

			// Check message content
			messageStart := messageLenStart + 2
			if len(result) > messageStart {
				actualMessage := string(result[messageStart:])
				if actualMessage != tt.message {
					t.Errorf("Message content = %q, expected %q", actualMessage, tt.message)
				}
			}
		})
	}
}

func TestMessagePacket_Version(t *testing.T) {
	packet := NewMessagePacket("test", "test")
	if packet.Version() != 1 {
		t.Errorf("Version() = %d, expected 1", packet.Version())
	}
}

func TestMessagePacket_Code(t *testing.T) {
	packet := NewMessagePacket("test", "test")
	if packet.Code() != 100 {
		t.Errorf("Code() = %d, expected 100", packet.Code())
	}
}

func TestMessagePacket_RoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		sender  string
		message string
	}{
		{"empty", "", ""},
		{"simple", "Alice", "hello"},
		{"with spaces", "Alice", "hello world"},
		{"with newlines", "Alice", "hello\nworld"},
		{"with tabs", "Alice", "hello\tworld"},
		{"unicode", "Alice", "héllo wörld"},
		{"long message", "Alice", "this is a very long message that should test the serialization and deserialization of longer strings to ensure everything works correctly"},
		{"special chars", "Alice", "!@#$%^&*()_+-=[]{}|;:,.<>?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			original := NewMessagePacket(tt.sender, tt.message)
			serialized := original.Serialize()

			// Deserialize
			deserialized, bytesConsumed, err := DeSerialize(serialized)
			if err != nil {
				t.Fatalf("DeSerialize() error = %v", err)
			}

			// Check the deserialized packet
			messagePacket, ok := deserialized.(*MessagePacket)
			if !ok {
				t.Fatalf("DeSerialize() returned wrong type: %T", deserialized)
			}

			// Verify fields
			if messagePacket.Version() != original.Version() {
				t.Errorf("Version mismatch: got %d, expected %d", messagePacket.Version(), original.Version())
			}
			if messagePacket.Code() != original.Code() {
				t.Errorf("Code mismatch: got %d, expected %d", messagePacket.Code(), original.Code())
			}
			if messagePacket.Sender != original.Sender {
				t.Errorf("Sender mismatch: got %q, expected %q", messagePacket.Sender, original.Sender)
			}
			if messagePacket.Message != original.Message {
				t.Errorf("Message mismatch: got %q, expected %q", messagePacket.Message, original.Message)
			}

			// Verify bytes consumed
			if bytesConsumed != len(serialized) {
				t.Errorf("Bytes consumed = %d, expected %d", bytesConsumed, len(serialized))
			}
		})
	}
}

func TestMessagePacket_DeSerialize_Errors(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expectedErr string
	}{
		{
			name:        "empty data",
			data:        []byte{},
			expectedErr: "incomplete packet header",
		},
		{
			name:        "only version",
			data:        []byte{1},
			expectedErr: "incomplete packet header",
		},
		{
			name:        "wrong version",
			data:        []byte{2, 100, 0, 5, 'h', 'e', 'l', 'l', 'o'},
			expectedErr: "invalid version",
		},
		{
			name:        "incomplete header",
			data:        []byte{1, 100, 0}, // missing second byte of sender length
			expectedErr: "incomplete packet",
		},
		{
			name:        "incomplete sender",
			data:        []byte{1, 100, 0, 5, 'h', 'e', 'l'}, // sender too short
			expectedErr: "incomplete packet",
		},
		{
			name:        "incomplete message length",
			data:        []byte{1, 100, 0, 5, 'A', 'l', 'i', 'c', 'e'}, // missing message length
			expectedErr: "incomplete packet",
		},
		{
			name:        "incomplete message",
			data:        []byte{1, 100, 0, 5, 'A', 'l', 'i', 'c', 'e', 0, 5, 'h', 'e', 'l'}, // message too short
			expectedErr: "incomplete packet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := DeSerialize(tt.data)
			if err == nil {
				t.Errorf("DeSerialize() expected error %q, got nil", tt.expectedErr)
				return
			}
			if err.Error() != tt.expectedErr {
				t.Errorf("DeSerialize() error = %q, expected %q", err.Error(), tt.expectedErr)
			}
		})
	}
}

func TestMessagePacket_EdgeCases(t *testing.T) {
	t.Run("null bytes in message", func(t *testing.T) {
		sender := "Alice"
		message := "hello\x00world"
		packet := NewMessagePacket(sender, message)
		serialized := packet.Serialize()

		deserialized, _, err := DeSerialize(serialized)
		if err != nil {
			t.Fatalf("DeSerialize() error = %v", err)
		}

		messagePacket := deserialized.(*MessagePacket)
		if messagePacket.Sender != sender {
			t.Errorf("Sender with null bytes: got %q, expected %q", messagePacket.Sender, sender)
		}
		if messagePacket.Message != message {
			t.Errorf("Message with null bytes: got %q, expected %q", messagePacket.Message, message)
		}
	})

	t.Run("very long message", func(t *testing.T) {
		sender := "Alice"
		// Create a 10KB message
		longMessage := make([]byte, 10240)
		for i := range longMessage {
			longMessage[i] = byte(i % 256)
		}
		message := string(longMessage)

		packet := NewMessagePacket(sender, message)
		serialized := packet.Serialize()

		deserialized, _, err := DeSerialize(serialized)
		if err != nil {
			t.Fatalf("DeSerialize() error = %v", err)
		}

		messagePacket := deserialized.(*MessagePacket)
		if messagePacket.Sender != sender {
			t.Errorf("Long message sender: got %q, expected %q", messagePacket.Sender, sender)
		}
		if messagePacket.Message != message {
			t.Errorf("Long message: length mismatch, got %d, expected %d", len(messagePacket.Message), len(message))
		}
	})

	t.Run("binary data in message", func(t *testing.T) {
		sender := "Alice"
		// Test with actual binary data, not just text
		binaryData := []byte{0, 1, 2, 255, 254, 253}
		message := string(binaryData)

		packet := NewMessagePacket(sender, message)
		serialized := packet.Serialize()

		deserialized, _, err := DeSerialize(serialized)
		if err != nil {
			t.Fatalf("DeSerialize() error = %v", err)
		}

		messagePacket := deserialized.(*MessagePacket)
		if messagePacket.Sender != sender {
			t.Errorf("Binary message sender: got %q, expected %q", messagePacket.Sender, sender)
		}
		if messagePacket.Message != message {
			t.Errorf("Binary message: got %v, expected %v", []byte(messagePacket.Message), binaryData)
		}
	})
}

func TestMessagePacket_Constructor(t *testing.T) {
	t.Run("normal construction", func(t *testing.T) {
		sender := "Alice"
		message := "test message"
		packet := NewMessagePacket(sender, message)

		if packet.Version() != 1 {
			t.Errorf("Version() = %d, expected 1", packet.Version())
		}
		if packet.Code() != 100 {
			t.Errorf("Code() = %d, expected 100", packet.Code())
		}
		if packet.Sender != sender {
			t.Errorf("Sender = %q, expected %q", packet.Sender, sender)
		}
		if packet.Message != message {
			t.Errorf("Message = %q, expected %q", packet.Message, message)
		}
	})

	t.Run("empty sender and message", func(t *testing.T) {
		sender := ""
		message := ""
		packet := NewMessagePacket(sender, message)

		if packet.Sender != sender {
			t.Errorf("Empty sender: got %q, expected empty string", packet.Sender)
		}
		if packet.Message != message {
			t.Errorf("Empty message: got %q, expected empty string", packet.Message)
		}
	})
}

func BenchmarkMessagePacket_Serialize(b *testing.B) {
	sender := "Alice"
	messages := []string{
		"",
		"hello",
		"hello world this is a longer message for benchmarking",
		string(make([]byte, 1024)), // 1KB message
	}

	for _, message := range messages {
		b.Run("message_"+string(rune(len(message))), func(b *testing.B) {
			packet := NewMessagePacket(sender, message)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				packet.Serialize()
			}
		})
	}
}

func BenchmarkMessagePacket_RoundTrip(b *testing.B) {
	sender := "Alice"
	messages := []string{
		"",
		"hello",
		"hello world this is a longer message for benchmarking",
		string(make([]byte, 1024)), // 1KB message
	}

	for _, message := range messages {
		b.Run("roundtrip_"+string(rune(len(message))), func(b *testing.B) {
			original := NewMessagePacket(sender, message)
			serialized := original.Serialize()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				DeSerialize(serialized)
			}
		})
	}
}
