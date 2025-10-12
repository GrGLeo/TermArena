package internal

import (
	"context"
	"io"
	"log/slog"
	"sync"

	pb "github.com/GrGLeo/TermArena/pkg/shared/proto/room_manager"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ManagerInterface interface {
	MoveRoom(roomStatusIn, roomStatusOut RoomStatus, roomType RoomType, roomID RoomID)
	LookRoom(username string, roomType RoomType) (Team, RoomID, RoomStatus)
	CloseRoom(roomID RoomID, username string) ([]string, RoomType, error)
	RemovePlayer(roomID RoomID, username string) error
	GetRoomInfo(roomType RoomType, roomStatus RoomStatus, roomID RoomID) ([]*pb.UserInfo, error)
	UpdatePlayerSpell(roomType RoomType, roomID RoomID, username string, spells Spells) ([]string, error)
}

type RoomHandler struct {
	pb.UnimplementedRoomServiceServer
	manager ManagerInterface
	changes chan *pb.RoomChangeNotification
	logger  *slog.Logger
}

func NewRoomHandler(changes chan *pb.RoomChangeNotification, manager ManagerInterface, logger *slog.Logger) *RoomHandler {
	return &RoomHandler{
		manager: manager,
		changes: changes,
		logger:  logger,
	}
}

func (rh *RoomHandler) LookRoom(ctx context.Context, req *pb.LookRoomRequest) (*pb.LookRoomResponse, error) {
	team, roomID, status := rh.manager.LookRoom(req.Username, RoomType(req.RoomType))
	if status == LOBBY {
		// Error here should not be possible
		userInfos, _ := rh.manager.GetRoomInfo(RoomType(req.RoomType), status, roomID)
		notif := &pb.RoomChangeNotification{
			RoomID:    uint32(roomID),
			Ready:     false,
			UserInfos: userInfos,
		}
		rh.changes <- notif
	}
	rh.logger.Info("LookRoom request processed", "username", req.Username, "roomType", req.RoomType, "roomID", roomID, "team", team)
	return &pb.LookRoomResponse{
		RoomID: uint32(roomID),
		Team:   uint32(team),
	}, nil
}

func (rh *RoomHandler) QuitRoom(ctx context.Context, req *pb.QuitRoomRequest) (*pb.QuitRoomResponse, error) {
	usernames, roomType, err := rh.manager.CloseRoom(RoomID(req.RoomID), req.Username)
	if err != nil {
		// Handle the error by returning grpc error
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	response := make(chan *pb.RequeueInfo, len(usernames)-1)
	var wg sync.WaitGroup
	for _, user := range usernames {
		// We do not queue back the user who quit
		if user == req.Username {
			continue
		}
		wg.Add(1)
		go func(u string) {
      defer wg.Done()
			resp, err := rh.LookRoom(ctx, &pb.LookRoomRequest{Username: u, RoomType: uint32(roomType)})
			if err != nil {
				rh.logger.Error("Failed to re-queue user", "user", u, "error", err)
				response <- nil
			} else {
        info := &pb.RequeueInfo{
          Username: u,
          RoomID: resp.RoomID,
          Team: resp.Team,
        }
				response <- info
			}
		}(user)
	}
	wg.Wait()
	var reQueued []*pb.RequeueInfo
	for i := 0; i < len(usernames)-1; i++ {
		if resp := <-response; resp != nil {
      reQueued = append(reQueued, resp)
		}
	}
	return &pb.QuitRoomResponse{RequeueInfos: reQueued}, nil
}

func (rh *RoomHandler) NotifyRoomChanges(stream grpc.BidiStreamingServer[pb.Ack, pb.RoomChangeNotification]) error {
	for {
		select {
		case notification := <-rh.changes:
			rh.logger.Info("notification received", "roomID", notification.RoomID, "ready", notification.Ready)
			if err := stream.Send(notification); err != nil {
				rh.logger.Error("Failed to send notification: %v", err)
				return err
			}

			// Wait for Ack from the client
			_, err := stream.Recv()
			if err == io.EOF {
				return nil // Client closed the stream
			}
			if err != nil {
				rh.logger.Error("Failed to receive Ack", "error", err)
				return err
			}

		// Case 2: Handle client disconnection or other cases
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

func (rh *RoomHandler) UpdateSpell(ctx context.Context, req *pb.UpdateSpellRequest) (*pb.UpdateSpellResponse, error) {
	usernames, err := rh.manager.UpdatePlayerSpell(RoomType(req.RomType), RoomID(req.RoomID), req.Username, Spells{int(req.Spell1), int(req.Spell2)})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	userInfo := &pb.UserInfo{
		Username: req.Username,
		Spell1:   req.Spell1,
		Spell2:   req.Spell2,
	}

	return &pb.UpdateSpellResponse{
		Usernames: usernames,
		User:      userInfo,
	}, nil

}
