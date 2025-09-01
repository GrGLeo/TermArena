package main

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	pb "github.com/GrGLeo/ctf_game/pkg/shared/proto/message"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockMessageManager implements the MessageManager interface for testing
type mockMessageManager struct {
	RegisterClientFunc   func(client string, roomID int) error
	UnregisterClientFunc func(client string) error
	RouteMessageFunc     func(sender string, content string) ([]string, string, error)
}

func (m *mockMessageManager) RegisterClient(client string, roomID int) error {
	if m.RegisterClientFunc != nil {
		return m.RegisterClientFunc(client, roomID)
	}
	return nil
}

func (m *mockMessageManager) UnregisterClient(client string) error {
	if m.UnregisterClientFunc != nil {
		return m.UnregisterClientFunc(client)
	}
	return nil
}

func (m *mockMessageManager) RouteMessage(sender string, content string) ([]string, string, error) {
	if m.RouteMessageFunc != nil {
		return m.RouteMessageFunc(sender, content)
	}
	return []string{}, "", nil
}

func createTestHandler(mockManager *mockMessageManager) *MessageHandler {
	if mockManager == nil {
		mockManager = &mockMessageManager{}
	}
	logger := slog.New(slog.NewTextHandler(&nullWriter{}, nil))
	return &MessageHandler{
		manager: mockManager, // This will work with interface{}
		logger:  logger,
	}
}

func TestNewMessageHandler(t *testing.T) {
	mockManager := &mockMessageManager{}
	logger := slog.New(slog.NewTextHandler(&nullWriter{}, nil))

	handler := NewMessageHandler(mockManager, logger)

	if handler == nil {
		t.Fatal("NewMessageHandler returned nil")
	}
	if handler.manager != mockManager {
		t.Error("manager not set correctly")
	}
	if handler.logger == nil {
		t.Error("logger not set")
	}
}

func TestRegisterClientHandler(t *testing.T) {
	tests := []struct {
		name           string
		req            *pb.RegisterClientRequest
		mockSetup      func(*mockMessageManager)
		wantErr        bool
		expectedCode   codes.Code
		expectedErrMsg string
	}{
		{
			name: "successful registration",
			req: &pb.RegisterClientRequest{
				Client: "client1",
				RoomId: "1",
			},
			mockSetup: func(m *mockMessageManager) {
				m.RegisterClientFunc = func(client string, roomID int) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "empty client ID",
			req: &pb.RegisterClientRequest{
				Client: "",
				RoomId: "1",
			},
			wantErr:      true,
			expectedCode: codes.InvalidArgument,
		},
		{
			name: "invalid room ID format",
			req: &pb.RegisterClientRequest{
				Client: "client1",
				RoomId: "invalid",
			},
			wantErr:      true,
			expectedCode: codes.InvalidArgument,
		},
		{
			name: "business logic error - already exists",
			req: &pb.RegisterClientRequest{
				Client: "client1",
				RoomId: "1",
			},
			mockSetup: func(m *mockMessageManager) {
				m.RegisterClientFunc = func(client string, roomID int) error {
					return errors.New("client client1 is already in the room 1")
				}
			},
			wantErr:        true,
			expectedCode:   codes.AlreadyExists,
			expectedErrMsg: "client client1 is already in the room 1",
		},
		{
			name: "business logic error - not found",
			req: &pb.RegisterClientRequest{
				Client: "client1",
				RoomId: "1",
			},
			mockSetup: func(m *mockMessageManager) {
				m.RegisterClientFunc = func(client string, roomID int) error {
					return errors.New("client client1 not registered")
				}
			},
			wantErr:        true,
			expectedCode:   codes.NotFound,
			expectedErrMsg: "client client1 not registered",
		},
		{
			name: "business logic error - validation",
			req: &pb.RegisterClientRequest{
				Client: "client1",
				RoomId: "1",
			},
			mockSetup: func(m *mockMessageManager) {
				m.RegisterClientFunc = func(client string, roomID int) error {
					return errors.New("client cannot be empty")
				}
			},
			wantErr:        true,
			expectedCode:   codes.InvalidArgument,
			expectedErrMsg: "client cannot be empty",
		},
		{
			name: "business logic error - internal",
			req: &pb.RegisterClientRequest{
				Client: "client1",
				RoomId: "1",
			},
			mockSetup: func(m *mockMessageManager) {
				m.RegisterClientFunc = func(client string, roomID int) error {
					return errors.New("internal database error")
				}
			},
			wantErr:        true,
			expectedCode:   codes.Internal,
			expectedErrMsg: "internal database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := &mockMessageManager{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockManager)
			}

			handler := createTestHandler(mockManager)

			resp, err := handler.RegisterClient(context.Background(), tt.req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else {
					st, ok := status.FromError(err)
					if !ok {
						t.Errorf("expected gRPC status error, got %T", err)
					} else {
						if st.Code() != tt.expectedCode {
							t.Errorf("expected code %v, got %v", tt.expectedCode, st.Code())
						}
						if tt.expectedErrMsg != "" && !strings.Contains(st.Message(), tt.expectedErrMsg) {
							t.Errorf("expected error message to contain %q, got %q", tt.expectedErrMsg, st.Message())
						}
					}
				}
				if resp != nil {
					t.Errorf("expected nil response on error, got %v", resp)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if resp == nil {
					t.Error("expected non-nil response on success")
				}
			}
		})
	}
}

func TestUnregisterClientHandler(t *testing.T) {
	tests := []struct {
		name           string
		req            *pb.UnregisterClientRequest
		mockSetup      func(*mockMessageManager)
		wantErr        bool
		expectedCode   codes.Code
		expectedErrMsg string
	}{
		{
			name: "successful unregistration",
			req: &pb.UnregisterClientRequest{
				Client: "client1",
			},
			mockSetup: func(m *mockMessageManager) {
				m.UnregisterClientFunc = func(client string) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "business logic error - not found",
			req: &pb.UnregisterClientRequest{
				Client: "client1",
			},
			mockSetup: func(m *mockMessageManager) {
				m.UnregisterClientFunc = func(client string) error {
					return errors.New("Failed to find client client1 to unregister")
				}
			},
			wantErr:        true,
			expectedCode:   codes.NotFound,
			expectedErrMsg: "Failed to find client client1 to unregister",
		},
		{
			name: "business logic error - internal",
			req: &pb.UnregisterClientRequest{
				Client: "client1",
			},
			mockSetup: func(m *mockMessageManager) {
				m.UnregisterClientFunc = func(client string) error {
					return errors.New("database connection failed")
				}
			},
			wantErr:        true,
			expectedCode:   codes.Internal,
			expectedErrMsg: "database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := &mockMessageManager{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockManager)
			}

			handler := createTestHandler(mockManager)

			resp, err := handler.UnregisterClient(context.Background(), tt.req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else {
					st, ok := status.FromError(err)
					if !ok {
						t.Errorf("expected gRPC status error, got %T", err)
					} else {
						if st.Code() != tt.expectedCode {
							t.Errorf("expected code %v, got %v", tt.expectedCode, st.Code())
						}
						if tt.expectedErrMsg != "" && !strings.Contains(st.Message(), tt.expectedErrMsg) {
							t.Errorf("expected error message to contain %q, got %q", tt.expectedErrMsg, st.Message())
						}
					}
				}
				if resp != nil {
					t.Errorf("expected nil response on error, got %v", resp)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if resp == nil {
					t.Error("expected non-nil response on success")
				}
			}
		})
	}
}

func TestRouteMessageHandler(t *testing.T) {
	tests := []struct {
		name              string
		req               *pb.RouteMessageRequest
		mockSetup         func(*mockMessageManager)
		wantErr           bool
		expectedCode      codes.Code
		expectedReceivers []string
		expectedContent   string
	}{
		{
			name: "successful message routing",
			req: &pb.RouteMessageRequest{
				Sender:  "client1",
				Content: "Hello everyone",
			},
			mockSetup: func(m *mockMessageManager) {
				m.RouteMessageFunc = func(sender string, content string) ([]string, string, error) {
					return []string{"client2", "client3"}, "(room) client1: Hello everyone", nil
				}
			},
			wantErr:           false,
			expectedReceivers: []string{"client2", "client3"},
			expectedContent:   "(room) client1: Hello everyone",
		},
		{
			name: "business logic error - sender not registered",
			req: &pb.RouteMessageRequest{
				Sender:  "client1",
				Content: "Hello",
			},
			mockSetup: func(m *mockMessageManager) {
				m.RouteMessageFunc = func(sender string, content string) ([]string, string, error) {
					return nil, "", errors.New("sender client1 not registered")
				}
			},
			wantErr:           true,
			expectedCode:      codes.NotFound,
			expectedReceivers: nil,
			expectedContent:   "",
		},
		{
			name: "business logic error - message too long",
			req: &pb.RouteMessageRequest{
				Sender:  "client1",
				Content: "Hello",
			},
			mockSetup: func(m *mockMessageManager) {
				m.RouteMessageFunc = func(sender string, content string) ([]string, string, error) {
					return nil, "", errors.New("message too long (max message size: 1000)")
				}
			},
			wantErr:           true,
			expectedCode:      codes.InvalidArgument,
			expectedReceivers: nil,
			expectedContent:   "",
		},
		{
			name: "business logic error - target not in room",
			req: &pb.RouteMessageRequest{
				Sender:  "client1",
				Content: "/client2 Hello",
			},
			mockSetup: func(m *mockMessageManager) {
				m.RouteMessageFunc = func(sender string, content string) ([]string, string, error) {
					return nil, "", errors.New("target user client2 not in room")
				}
			},
			wantErr:           true,
			expectedCode:      codes.NotFound,
			expectedReceivers: nil,
			expectedContent:   "",
		},
		{
			name: "business logic error - internal error",
			req: &pb.RouteMessageRequest{
				Sender:  "client1",
				Content: "Hello",
			},
			mockSetup: func(m *mockMessageManager) {
				m.RouteMessageFunc = func(sender string, content string) ([]string, string, error) {
					return nil, "", errors.New("database connection failed")
				}
			},
			wantErr:           true,
			expectedCode:      codes.Internal,
			expectedReceivers: nil,
			expectedContent:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := &mockMessageManager{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockManager)
			}

			handler := createTestHandler(mockManager)

			resp, err := handler.RouteMessage(context.Background(), tt.req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else {
					st, ok := status.FromError(err)
					if !ok {
						t.Errorf("expected gRPC status error, got %T", err)
					} else {
						if st.Code() != tt.expectedCode {
							t.Errorf("expected code %v, got %v", tt.expectedCode, st.Code())
						}
					}
				}
				if resp != nil {
					t.Errorf("expected nil response on error, got %v", resp)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if resp == nil {
					t.Error("expected non-nil response on success")
				} else {
					// Check receivers
					if len(resp.Receivers) != len(tt.expectedReceivers) {
						t.Errorf("receivers = %v, want %v", resp.Receivers, tt.expectedReceivers)
					} else {
						for _, expected := range tt.expectedReceivers {
							found := false
							for _, actual := range resp.Receivers {
								if actual == expected {
									found = true
									break
								}
							}
							if !found {
								t.Errorf("expected receiver %s not found in %v", expected, resp.Receivers)
							}
						}
					}

					// Check content
					if resp.Content != tt.expectedContent {
						t.Errorf("content = %q, want %q", resp.Content, tt.expectedContent)
					}
				}
			}
		})
	}
}

func TestMapToGRPCError(t *testing.T) {
	tests := []struct {
		name         string
		inputErr     error
		expectedCode codes.Code
	}{
		{
			name:         "already exists error",
			inputErr:     errors.New("client client1 is already in the room 1"),
			expectedCode: codes.AlreadyExists,
		},
		{
			name:         "not found error",
			inputErr:     errors.New("client client1 not registered"),
			expectedCode: codes.NotFound,
		},
		{
			name:         "validation error",
			inputErr:     errors.New("client cannot be empty"),
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "too long error",
			inputErr:     errors.New("message too long (max message size: 1000)"),
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "room not found",
			inputErr:     errors.New("room 1 not found"),
			expectedCode: codes.NotFound,
		},
		{
			name:         "internal error",
			inputErr:     errors.New("database connection failed"),
			expectedCode: codes.Internal,
		},
		{
			name:         "unknown error",
			inputErr:     errors.New("some unknown error"),
			expectedCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapToGRPCError(tt.inputErr)

			st, ok := status.FromError(result)
			if !ok {
				t.Errorf("expected gRPC status error, got %T", result)
				return
			}

			if st.Code() != tt.expectedCode {
				t.Errorf("expected code %v, got %v", tt.expectedCode, st.Code())
			}

			// Verify the original error message is preserved
			if !strings.Contains(st.Message(), tt.inputErr.Error()) {
				t.Errorf("expected error message to contain %q, got %q", tt.inputErr.Error(), st.Message())
			}
		})
	}
}

func TestHandlerInputValidation(t *testing.T) {
	handler := createTestHandler(nil)

	t.Run("RegisterClient empty client", func(t *testing.T) {
		req := &pb.RegisterClientRequest{
			Client: "",
			RoomId: "1",
		}

		_, err := handler.RegisterClient(context.Background(), req)
		if err == nil {
			t.Error("expected error for empty client")
		}

		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument error, got %v", err)
		}
	})

	t.Run("RegisterClient invalid room ID", func(t *testing.T) {
		req := &pb.RegisterClientRequest{
			Client: "client1",
			RoomId: "invalid",
		}

		_, err := handler.RegisterClient(context.Background(), req)
		if err == nil {
			t.Error("expected error for invalid room ID")
		}

		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument error, got %v", err)
		}
	})
}

func BenchmarkRegisterClientHandler(b *testing.B) {
	mockManager := &mockMessageManager{
		RegisterClientFunc: func(client string, roomID int) error {
			return nil
		},
	}
	handler := createTestHandler(mockManager)

	req := &pb.RegisterClientRequest{
		Client: "client1",
		RoomId: "1",
	}

	b.ResetTimer()
	for range b.N {
		_, _ = handler.RegisterClient(context.Background(), req)
	}
}

func BenchmarkRouteMessageHandler(b *testing.B) {
	mockManager := &mockMessageManager{
		RouteMessageFunc: func(sender string, content string) ([]string, string, error) {
			return []string{"client2"}, "(room) client1: Hello", nil
		},
	}
	handler := createTestHandler(mockManager)

	req := &pb.RouteMessageRequest{
		Sender:  "client1",
		Content: "Hello",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handler.RouteMessage(context.Background(), req)
	}
}
