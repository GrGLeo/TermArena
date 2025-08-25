package shared

import (
	"testing"
)

func TestMessagePacket_Serialize(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{"empty message", ""},
		{"simple message", "hello"},
		{"message with spaces", "hello world"},
		{"message with special characters", "hello\nworld\t!"},
		{"unicode message", "héllo wörld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet := NewMessagePacket(tt.message)
			result := packet.Serialize()

			// Check basic structure
			if len(result) < 4 {
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

			// Check length field
			messageLen := int(result[2])<<8 | int(result[3])
			if messageLen != len(tt.message) {
				t.Errorf("Message length field = %d, expected %d", messageLen, len(tt.message))
			}

			// Check message content
			if len(result) > 4 {
				actualMessage := string(result[4:])
				if actualMessage != tt.message {
					t.Errorf("Message content = %q, expected %q", actualMessage, tt.message)
				}
			}
		})
	}
}

func TestMessagePacket_Version(t *testing.T) {
	packet := NewMessagePacket("test")
	if packet.Version() != 1 {
		t.Errorf("Version() = %d, expected 1", packet.Version())
	}
}

func TestMessagePacket_Code(t *testing.T) {
	packet := NewMessagePacket("test")
	if packet.Code() != 100 {
		t.Errorf("Code() = %d, expected 100", packet.Code())
	}
}

func TestMessagePacket_RoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{"empty", ""},
		{"simple", "hello"},
		{"with spaces", "hello world"},
		{"with newlines", "hello\nworld"},
		{"with tabs", "hello\tworld"},
		{"unicode", "héllo wörld"},
		{"long message", "this is a very long message that should test the serialization and deserialization of longer strings to ensure everything works correctly"},
		{"special chars", "!@#$%^&*()_+-=[]{}|;:,.<>?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			original := NewMessagePacket(tt.message)
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
			data:        []byte{1, 100, 0}, // missing second byte of length
			expectedErr: "incomplete packet",
		},
		{
			name:        "incomplete message",
			data:        []byte{1, 100, 0, 5, 'h', 'e', 'l'}, // message too short
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
		message := "hello\x00world"
		packet := NewMessagePacket(message)
		serialized := packet.Serialize()

		deserialized, _, err := DeSerialize(serialized)
		if err != nil {
			t.Fatalf("DeSerialize() error = %v", err)
		}

		messagePacket := deserialized.(*MessagePacket)
		if messagePacket.Message != message {
			t.Errorf("Message with null bytes: got %q, expected %q", messagePacket.Message, message)
		}
	})

	t.Run("very long message", func(t *testing.T) {
		// Create a 10KB message
		longMessage := make([]byte, 10240)
		for i := range longMessage {
			longMessage[i] = byte(i % 256)
		}
		message := string(longMessage)

		packet := NewMessagePacket(message)
		serialized := packet.Serialize()

		deserialized, _, err := DeSerialize(serialized)
		if err != nil {
			t.Fatalf("DeSerialize() error = %v", err)
		}

		messagePacket := deserialized.(*MessagePacket)
		if messagePacket.Message != message {
			t.Errorf("Long message: length mismatch, got %d, expected %d", len(messagePacket.Message), len(message))
		}
	})

	t.Run("binary data in message", func(t *testing.T) {
		// Test with actual binary data, not just text
		binaryData := []byte{0, 1, 2, 255, 254, 253}
		message := string(binaryData)

		packet := NewMessagePacket(message)
		serialized := packet.Serialize()

		deserialized, _, err := DeSerialize(serialized)
		if err != nil {
			t.Fatalf("DeSerialize() error = %v", err)
		}

		messagePacket := deserialized.(*MessagePacket)
		if messagePacket.Message != message {
			t.Errorf("Binary message: got %v, expected %v", []byte(messagePacket.Message), binaryData)
		}
	})
}

func TestMessagePacket_Constructor(t *testing.T) {
	t.Run("normal construction", func(t *testing.T) {
		message := "test message"
		packet := NewMessagePacket(message)

		if packet.Version() != 1 {
			t.Errorf("Version() = %d, expected 1", packet.Version())
		}
		if packet.Code() != 100 {
			t.Errorf("Code() = %d, expected 100", packet.Code())
		}
		if packet.Message != message {
			t.Errorf("Message = %q, expected %q", packet.Message, message)
		}
	})

	t.Run("empty message", func(t *testing.T) {
		packet := NewMessagePacket("")

		if packet.Message != "" {
			t.Errorf("Empty message: got %q, expected empty string", packet.Message)
		}
	})
}

func BenchmarkMessagePacket_Serialize(b *testing.B) {
	messages := []string{
		"",
		"hello",
		"hello world this is a longer message for benchmarking",
		string(make([]byte, 1024)), // 1KB message
	}

	for _, message := range messages {
		b.Run("message_"+string(rune(len(message))), func(b *testing.B) {
			packet := NewMessagePacket(message)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				packet.Serialize()
			}
		})
	}
}

func BenchmarkMessagePacket_RoundTrip(b *testing.B) {
	messages := []string{
		"",
		"hello",
		"hello world this is a longer message for benchmarking",
		string(make([]byte, 1024)), // 1KB message
	}

	for _, message := range messages {
		b.Run("roundtrip_"+string(rune(len(message))), func(b *testing.B) {
			original := NewMessagePacket(message)
			serialized := original.Serialize()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				DeSerialize(serialized)
			}
		})
	}
}
