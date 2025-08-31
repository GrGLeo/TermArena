package handlers

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	connmanager "github.com/GrGLeo/ctf/server/conn_manager"
	"github.com/GrGLeo/ctf/server/event"
	pb "github.com/GrGLeo/ctf/shared/proto/message"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Mock implementations for testing

type mockMessageServiceClient struct {
	RegisterClientFunc   func(ctx context.Context, req *pb.RegisterClientRequest, opts ...grpc.CallOption) (*pb.RegisterClientResponse, error)
	UnregisterClientFunc func(ctx context.Context, req *pb.UnregisterClientRequest, opts ...grpc.CallOption) (*pb.UnregisterClientResponse, error)
	RouteMessageFunc     func(ctx context.Context, req *pb.RouteMessageRequest, opts ...grpc.CallOption) (*pb.RouteMessageResponse, error)
}

func (m *mockMessageServiceClient) RegisterClient(ctx context.Context, req *pb.RegisterClientRequest, opts ...grpc.CallOption) (*pb.RegisterClientResponse, error) {
	if m.RegisterClientFunc != nil {
		return m.RegisterClientFunc(ctx, req, opts...)
	}
	return &pb.RegisterClientResponse{}, nil
}

func (m *mockMessageServiceClient) UnregisterClient(ctx context.Context, req *pb.UnregisterClientRequest, opts ...grpc.CallOption) (*pb.UnregisterClientResponse, error) {
	if m.UnregisterClientFunc != nil {
		return m.UnregisterClientFunc(ctx, req, opts...)
	}
	return &pb.UnregisterClientResponse{}, nil
}

func (m *mockMessageServiceClient) RouteMessage(ctx context.Context, req *pb.RouteMessageRequest, opts ...grpc.CallOption) (*pb.RouteMessageResponse, error) {
	if m.RouteMessageFunc != nil {
		return m.RouteMessageFunc(ctx, req, opts...)
	}
	return &pb.RouteMessageResponse{}, nil
}

type mockConnectionManager struct {
	RegisterFunc   func(conn *net.TCPConn, clientID string)
	UnregisterFunc func(clientID string)
}

func (m *mockConnectionManager) Register(conn *net.TCPConn, clientID string) {
	if m.RegisterFunc != nil {
		m.RegisterFunc(conn, clientID)
	}
}

func (m *mockConnectionManager) Unregister(clientID string) {
	if m.UnregisterFunc != nil {
		m.UnregisterFunc(clientID)
	}
}

type mockSugaredLogger struct {
	InfowFunc  func(msg string, keysAndValues ...interface{})
	ErrorwFunc func(msg string, keysAndValues ...interface{})
}

func (m *mockSugaredLogger) Infow(msg string, keysAndValues ...interface{}) {
	if m.InfowFunc != nil {
		m.InfowFunc(msg, keysAndValues...)
	}
}

func (m *mockSugaredLogger) Errorw(msg string, keysAndValues ...interface{}) {
	if m.ErrorwFunc != nil {
		m.ErrorwFunc(msg, keysAndValues...)
	}
}

// Helper function to create test MessagesServiceClient
func createTestMessagesServiceClient(mockClient *mockMessageServiceClient, mockConnMgr *connmanager.ConnectionManager, mockLogger *zap.SugaredLogger) *MessagesServiceClient {
	if mockClient == nil {
		mockClient = &mockMessageServiceClient{}
	}
	if mockConnMgr == nil {
		mockConnMgr = connmanager.NewConnectionManager()
	}
	if mockLogger == nil {
		mockLogger = zap.NewNop().Sugar()
	}

	return &MessagesServiceClient{
		Client:      mockClient,
		connManager: mockConnMgr,
		logger:      mockLogger,
	}
}

func TestNewMessageServiceClient(t *testing.T) {
	// This test would require mocking grpc.NewClient, which is complex
	// For now, we'll test the structure creation
	t.Run("successful creation", func(t *testing.T) {
		// We can't easily test NewMessageServiceClient due to grpc.NewClient dependency
		// This would require integration testing or more complex mocking
		t.Skip("NewMessageServiceClient requires gRPC mocking - test in integration suite")
	})
}

func TestMessagesServiceClient_HandleClientRegistration(t *testing.T) {
	tests := []struct {
		name           string
		req            event.ClientRegistrationMessage
		mockSetup      func(*mockMessageServiceClient, *connmanager.ConnectionManager, *zap.SugaredLogger)
		expectedResult func(event.Message) bool
	}{
		{
			name: "successful registration",
			req: event.ClientRegistrationMessage{
				ClientID: "client1",
				RoomID:   1,
				Conn:     &net.TCPConn{}, // Use real TCPConn
			},
			mockSetup: func(mockClient *mockMessageServiceClient, mockConnMgr *connmanager.ConnectionManager, mockLogger *zap.SugaredLogger) {
				mockClient.RegisterClientFunc = func(ctx context.Context, req *pb.RegisterClientRequest, opts ...grpc.CallOption) (*pb.RegisterClientResponse, error) {
					return &pb.RegisterClientResponse{}, nil
				}
			},
			expectedResult: func(result event.Message) bool {
				resp, ok := result.(event.ClientRegistrationResponse)
				return ok && resp.Success && resp.Message == "Registration successful" && resp.ClientID == "client1"
			},
		},
		{
			name: "gRPC registration failure",
			req: event.ClientRegistrationMessage{
				ClientID: "client1",
				RoomID:   1,
				Conn:     &net.TCPConn{},
			},
			mockSetup: func(mockClient *mockMessageServiceClient, mockConnMgr *connmanager.ConnectionManager, mockLogger *zap.SugaredLogger) {
				mockClient.RegisterClientFunc = func(ctx context.Context, req *pb.RegisterClientRequest, opts ...grpc.CallOption) (*pb.RegisterClientResponse, error) {
					return nil, errors.New("gRPC connection failed")
				}
			},
			expectedResult: func(result event.Message) bool {
				resp, ok := result.(event.ClientRegistrationResponse)
				return ok && !resp.Success && resp.Message == "Failed to register with message service" && resp.ClientID == "client1"
			},
		},
		{
			name: "registration with different room ID",
			req: event.ClientRegistrationMessage{
				ClientID: "client2",
				RoomID:   5,
				Conn:     &net.TCPConn{},
			},
			mockSetup: func(mockClient *mockMessageServiceClient, mockConnMgr *connmanager.ConnectionManager, mockLogger *zap.SugaredLogger) {
				mockClient.RegisterClientFunc = func(ctx context.Context, req *pb.RegisterClientRequest, opts ...grpc.CallOption) (*pb.RegisterClientResponse, error) {
					if req.Client != "client2" || req.RoomId != "5" {
						t.Errorf("expected client='client2', roomId='5', got client='%s', roomId='%s'", req.Client, req.RoomId)
					}
					return &pb.RegisterClientResponse{}, nil
				}
			},
			expectedResult: func(result event.Message) bool {
				resp, ok := result.(event.ClientRegistrationResponse)
				return ok && resp.Success && resp.ClientID == "client2"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockMessageServiceClient{}
			mockConnMgr := connmanager.NewConnectionManager()
			mockLogger := zap.NewNop().Sugar()

			if tt.mockSetup != nil {
				tt.mockSetup(mockClient, mockConnMgr, mockLogger)
			}

			client := createTestMessagesServiceClient(mockClient, mockConnMgr, mockLogger)
			result := client.HandleClientRegistration(tt.req)

			if !tt.expectedResult(result) {
				t.Errorf("HandleClientRegistration() = %v, did not match expected result", result)
			}
		})
	}
}

func TestMessagesServiceClient_HandleClientUnregistration(t *testing.T) {
	tests := []struct {
		name           string
		req            event.ClientUnregistrationMessage
		mockSetup      func(*mockMessageServiceClient, *connmanager.ConnectionManager, *zap.SugaredLogger)
		expectedResult func(event.Message) bool
	}{
		{
			name: "successful unregistration",
			req: event.ClientUnregistrationMessage{
				ClientID: "client1",
			},
			mockSetup: func(mockClient *mockMessageServiceClient, mockConnMgr *connmanager.ConnectionManager, mockLogger *zap.SugaredLogger) {
				mockClient.UnregisterClientFunc = func(ctx context.Context, req *pb.UnregisterClientRequest, opts ...grpc.CallOption) (*pb.UnregisterClientResponse, error) {
					return &pb.UnregisterClientResponse{}, nil
				}
			},
			expectedResult: func(result event.Message) bool {
				resp, ok := result.(event.ClientUnregistrationResponse)
				return ok && resp.Success && resp.Message == "Unregistration successful" && resp.ClientID == "client1"
			},
		},
		{
			name: "gRPC unregistration failure",
			req: event.ClientUnregistrationMessage{
				ClientID: "client1",
			},
			mockSetup: func(mockClient *mockMessageServiceClient, mockConnMgr *connmanager.ConnectionManager, mockLogger *zap.SugaredLogger) {
				mockClient.UnregisterClientFunc = func(ctx context.Context, req *pb.UnregisterClientRequest, opts ...grpc.CallOption) (*pb.UnregisterClientResponse, error) {
					return nil, errors.New("gRPC connection failed")
				}
			},
			expectedResult: func(result event.Message) bool {
				resp, ok := result.(event.ClientUnregistrationResponse)
				return ok && !resp.Success && resp.Message == "Failed to unregister with message service" && resp.ClientID == "client1"
			},
		},
		{
			name: "unregistration with different client ID",
			req: event.ClientUnregistrationMessage{
				ClientID: "client2",
			},
			mockSetup: func(mockClient *mockMessageServiceClient, mockConnMgr *connmanager.ConnectionManager, mockLogger *zap.SugaredLogger) {
				mockClient.UnregisterClientFunc = func(ctx context.Context, req *pb.UnregisterClientRequest, opts ...grpc.CallOption) (*pb.UnregisterClientResponse, error) {
					if req.Client != "client2" {
						t.Errorf("expected client='client2', got client='%s'", req.Client)
					}
					return &pb.UnregisterClientResponse{}, nil
				}
			},
			expectedResult: func(result event.Message) bool {
				resp, ok := result.(event.ClientUnregistrationResponse)
				return ok && resp.Success && resp.ClientID == "client2"
			},
		},
		{
			name: "gRPC unregistration failure",
			req: event.ClientUnregistrationMessage{
				ClientID: "client1",
			},
			mockSetup: func(mockClient *mockMessageServiceClient, mockConnMgr *connmanager.ConnectionManager, mockLogger *zap.SugaredLogger) {
				mockClient.UnregisterClientFunc = func(ctx context.Context, req *pb.UnregisterClientRequest, opts ...grpc.CallOption) (*pb.UnregisterClientResponse, error) {
					return nil, errors.New("gRPC connection failed")
				}
			},
			expectedResult: func(result event.Message) bool {
				resp, ok := result.(event.ClientUnregistrationResponse)
				return ok && !resp.Success && resp.Message == "Failed to unregister with message service" && resp.ClientID == "client1"
			},
		},
		{
			name: "unregistration with different client ID",
			req: event.ClientUnregistrationMessage{
				ClientID: "client2",
			},
			mockSetup: func(mockClient *mockMessageServiceClient, mockConnMgr *connmanager.ConnectionManager, mockLogger *zap.SugaredLogger) {
				mockClient.UnregisterClientFunc = func(ctx context.Context, req *pb.UnregisterClientRequest, opts ...grpc.CallOption) (*pb.UnregisterClientResponse, error) {
					if req.Client != "client2" {
						t.Errorf("expected client='client2', got client='%s'", req.Client)
					}
					return &pb.UnregisterClientResponse{}, nil
				}
			},
			expectedResult: func(result event.Message) bool {
				resp, ok := result.(event.ClientUnregistrationResponse)
				return ok && resp.Success && resp.ClientID == "client2"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockMessageServiceClient{}
			mockConnMgr := connmanager.NewConnectionManager()
			mockLogger := zap.NewNop().Sugar()

			if tt.mockSetup != nil {
				tt.mockSetup(mockClient, mockConnMgr, mockLogger)
			}

			client := createTestMessagesServiceClient(mockClient, mockConnMgr, mockLogger)
			result := client.HandleClientUnregistration(tt.req)

			if !tt.expectedResult(result) {
				t.Errorf("HandleClientUnregistration() = %v, did not match expected result", result)
			}
		})
	}
}

func TestMessagesServiceClient_HandleRouteMessage(t *testing.T) {
	tests := []struct {
		name           string
		req            event.MessageRequestMessage
		mockSetup      func(*mockMessageServiceClient, *connmanager.ConnectionManager, *zap.SugaredLogger)
		expectedResult func(event.Message) bool
	}{
		{
			name: "successful message routing",
			req: event.MessageRequestMessage{
				Sender:     "client1",
				Message:    "Hello everyone",
				ResponseCh: make(chan event.Message, 1),
			},
			mockSetup: func(mockClient *mockMessageServiceClient, mockConnMgr *connmanager.ConnectionManager, mockLogger *zap.SugaredLogger) {
				mockClient.RouteMessageFunc = func(ctx context.Context, req *pb.RouteMessageRequest, opts ...grpc.CallOption) (*pb.RouteMessageResponse, error) {
					if req.Sender != "client1" || req.Content != "Hello everyone" {
						t.Errorf("expected sender='client1', content='Hello everyone', got sender='%s', content='%s'", req.Sender, req.Content)
					}
					return &pb.RouteMessageResponse{
						Receivers: []string{"client2", "client3"},
						Content:   "(room) client1: Hello everyone",
					}, nil
				}
			},
			expectedResult: func(result event.Message) bool {
				resp, ok := result.(event.MessageResponseMessage)
				if !ok {
					return false
				}
				if len(resp.Receivers) != 2 || resp.Receivers[0] != "client2" || resp.Receivers[1] != "client3" {
					return false
				}
				if resp.Message != "(room) client1: Hello everyone" {
					return false
				}
				return resp.ResponseCh != nil
			},
		},
		{
			name: "gRPC routing failure",
			req: event.MessageRequestMessage{
				Sender:     "client1",
				Message:    "Hello",
				ResponseCh: make(chan event.Message, 1),
			},
			mockSetup: func(mockClient *mockMessageServiceClient, mockConnMgr *connmanager.ConnectionManager, mockLogger *zap.SugaredLogger) {
				mockClient.RouteMessageFunc = func(ctx context.Context, req *pb.RouteMessageRequest, opts ...grpc.CallOption) (*pb.RouteMessageResponse, error) {
					return nil, errors.New("gRPC connection failed")
				}
			},
			expectedResult: func(result event.Message) bool {
				resp, ok := result.(event.MessageErrorResponse)
				return ok && resp.Error == "Failed to route message: gRPC connection failed" && resp.ResponseCh != nil
			},
		},
		{
			name: "empty message routing",
			req: event.MessageRequestMessage{
				Sender:     "client1",
				Message:    "",
				ResponseCh: make(chan event.Message, 1),
			},
			mockSetup: func(mockClient *mockMessageServiceClient, mockConnMgr *connmanager.ConnectionManager, mockLogger *zap.SugaredLogger) {
				mockClient.RouteMessageFunc = func(ctx context.Context, req *pb.RouteMessageRequest, opts ...grpc.CallOption) (*pb.RouteMessageResponse, error) {
					return &pb.RouteMessageResponse{
						Receivers: []string{},
						Content:   "(room) client1: ",
					}, nil
				}
			},
			expectedResult: func(result event.Message) bool {
				resp, ok := result.(event.MessageResponseMessage)
				return ok && len(resp.Receivers) == 0 && resp.Message == "(room) client1: "
			},
		},
		{
			name: "whisper message routing",
			req: event.MessageRequestMessage{
				Sender:     "client1",
				Message:    "/client2 Secret message",
				ResponseCh: make(chan event.Message, 1),
			},
			mockSetup: func(mockClient *mockMessageServiceClient, mockConnMgr *connmanager.ConnectionManager, mockLogger *zap.SugaredLogger) {
				mockClient.RouteMessageFunc = func(ctx context.Context, req *pb.RouteMessageRequest, opts ...grpc.CallOption) (*pb.RouteMessageResponse, error) {
					return &pb.RouteMessageResponse{
						Receivers: []string{"client2"},
						Content:   "(whisper) client1: Secret message",
					}, nil
				}
			},
			expectedResult: func(result event.Message) bool {
				resp, ok := result.(event.MessageResponseMessage)
				return ok && len(resp.Receivers) == 1 && resp.Receivers[0] == "client2" && resp.Message == "(whisper) client1: Secret message"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockMessageServiceClient{}
			mockConnMgr := connmanager.NewConnectionManager()
			mockLogger := zap.NewNop().Sugar()

			if tt.mockSetup != nil {
				tt.mockSetup(mockClient, mockConnMgr, mockLogger)
			}

			client := createTestMessagesServiceClient(mockClient, mockConnMgr, mockLogger)
			result := client.HandleRouteMessage(tt.req)

			if !tt.expectedResult(result) {
				t.Errorf("HandleRouteMessage() = %v, did not match expected result", result)
			}
		})
	}
}

// Mock connection for testing
type mockConn struct{}

func (m *mockConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (m *mockConn) Write(b []byte) (n int, err error)  { return len(b), nil }
func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() interface{}             { return nil }
func (m *mockConn) RemoteAddr() interface{}            { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

// Benchmark tests
func BenchmarkHandleClientRegistration(b *testing.B) {
	mockClient := &mockMessageServiceClient{
		RegisterClientFunc: func(ctx context.Context, req *pb.RegisterClientRequest, opts ...grpc.CallOption) (*pb.RegisterClientResponse, error) {
			return &pb.RegisterClientResponse{}, nil
		},
	}
	mockConnMgr := connmanager.NewConnectionManager()
	mockLogger := zap.NewNop().Sugar()

	client := createTestMessagesServiceClient(mockClient, mockConnMgr, mockLogger)
	req := event.ClientRegistrationMessage{
		ClientID: "client1",
		RoomID:   1,
		Conn:     &net.TCPConn{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.HandleClientRegistration(req)
	}
}

func BenchmarkHandleRouteMessage(b *testing.B) {
	mockClient := &mockMessageServiceClient{
		RouteMessageFunc: func(ctx context.Context, req *pb.RouteMessageRequest, opts ...grpc.CallOption) (*pb.RouteMessageResponse, error) {
			return &pb.RouteMessageResponse{
				Receivers: []string{"client2"},
				Content:   "(room) client1: Hello",
			}, nil
		},
	}
	mockConnMgr := connmanager.NewConnectionManager()
	mockLogger := zap.NewNop().Sugar()

	client := createTestMessagesServiceClient(mockClient, mockConnMgr, mockLogger)
	req := event.MessageRequestMessage{
		Sender:     "client1",
		Message:    "Hello",
		ResponseCh: make(chan event.Message, 1),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.HandleRouteMessage(req)
	}
}
