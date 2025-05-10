package service

import (
	"context"
	"io"
	"log/slog"

	api "github.com/makarmolochaev/subscribe-service/api"
	"github.com/makarmolochaev/subscribe-service/internal/subpub"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PubSubService struct {
	api.UnimplementedPubSubServer
	pubsub subpub.SubPub
	log    *slog.Logger
}

func NewPubSubService(log *slog.Logger) *PubSubService {
	return &PubSubService{
		pubsub: subpub.NewSubPub(),
		log:    log,
	}
}

func (s *PubSubService) Subscribe(req *api.SubscribeRequest, stream api.PubSub_SubscribeServer) error {
	ctx := stream.Context()
	key := req.GetKey()

	s.log.Info("New subscription", slog.String("key", key))

	eventChan := make(chan *api.Event, 100)
	defer close(eventChan)

	sub, err := s.pubsub.Subscribe(key, func(msg interface{}) {
		select {
		case eventChan <- &api.Event{Data: msg.(string)}:
		default:
			s.log.Warn("Event channel full, dropping message", slog.String("key", key))
		}
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			s.log.Info("Subscription closed by client", slog.String("key", key))
			return nil

		case event := <-eventChan:
			if err := stream.Send(event); err != nil {
				if err == io.EOF {
					s.log.Info("Client closed stream", slog.String("key", key))
					return nil
				}
				s.log.Error("Failed to send event",
					slog.String("key", key), slog.Any("error", err))
				return status.Errorf(codes.Internal, "failed to send event: %v", err)
			}
		}
	}
}

func (s *PubSubService) Publish(ctx context.Context, req *api.PublishRequest) (*api.Empty, error) {
	key := req.GetKey()
	data := req.GetData()

	s.log.Debug("Publishing event",
		slog.String("key", key),
		slog.String("data", data))

	if err := s.pubsub.Publish(key, data); err != nil {
		s.log.Error("Failed to publish event",
			slog.String("key", key),
			slog.Any("error", err))
		return nil, status.Errorf(codes.Internal, "failed to publish: %v", err)
	}

	return &api.Empty{}, nil
}

func (s *PubSubService) Shutdown(ctx context.Context) error {
	return s.pubsub.Close(ctx)
}
