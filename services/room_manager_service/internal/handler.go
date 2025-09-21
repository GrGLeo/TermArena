package internal

import (
	"context"
	"log/slog"

	pb "github.com/GrGLeo/TermArena/pkg/shared/proto/room_manager"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ManagerInterface interface {
	MoveRoom(roomStatusIn, roomStatusOut RoomStatus, roomType RoomType, roomID RoomID)
	LookRoom(username string, roomType RoomType) (Team, RoomID, RoomStatus)
  RemovePlayer(roomID RoomID, username string) error
	GetRoomInfo(roomType RoomType, roomID RoomID) ([]*pb.UserInfo, error)
}

type RoomHandler struct {
	pb.UnimplementedRoomServiceServer
	manager ManagerInterface
	logger  *slog.Logger
}

func NewRoomHandler(manager ManagerInterface, logger *slog.Logger) *RoomHandler {
	return &RoomHandler{
		manager: manager,
		logger:  logger,
	}
}

func (rh *RoomHandler) LookRoom(ctx context.Context, req *pb.LookRoomRequest) (*pb.LookRoomResponse, error) {
	team, roomID, status := rh.manager.LookRoom(req.Username, RoomType(req.RoomType))
  if status == LOBBY {
  }
	rh.logger.Info("LookRoom request processed", "username", req.Username, "roomType", req.RoomType, "roomID", roomID, "team", team)
	return &pb.LookRoomResponse{
		RoomID:   uint32(roomID),
		Team:     uint32(team),
	}, nil
}

func (rh *RoomHandler) QuitRoom(ctx context.Context, req *pb.QuitRoomRequest) (*pb.QuitRoomResponse, error) {
  err := rh.manager.RemovePlayer(RoomID(req.RoomID), req.Username)
  if err != nil {
    // Handle the error by returning grpc error
    return nil, status.Errorf(codes.NotFound, err.Error())
  }
  return &pb.QuitRoomResponse{
    Success: true,
  }, nil
}
