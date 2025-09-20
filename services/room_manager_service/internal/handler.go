package internal

import (
	"context"
	"log/slog"

	pb "github.com/GrGLeo/TermArena/pkg/shared/proto/room_manager"
)

type ManagerInterface interface {
	MoveRoom(roomStatusIn, roomStatusOut RoomStatus, roomType RoomType, roomID RoomID)
	LookRoom(username string, roomType RoomType) (Team, RoomID)
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
	team, roomID := rh.manager.LookRoom(req.Username, RoomType(req.RoomType))
	rh.logger.Info("LookRoom request processed", "username", req.Username, "roomType", req.RoomType, "roomID", roomID, "team", team)
	return &pb.LookRoomResponse{
		RoomID:   uint32(roomID),
		Team:     uint32(team),
	}, nil
}
