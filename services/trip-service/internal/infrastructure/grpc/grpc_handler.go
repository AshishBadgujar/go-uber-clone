package grpc

import (
	"context"
	"ride-sharing/services/trip-service/internal/domain"
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type gRPCHandler struct {
	pb.UnimplementedTripServiceServer
	service domain.TripService
}

func NewGRPCHandler(server *grpc.Server, service domain.TripService) *gRPCHandler {
	handler := &gRPCHandler{
		service: service,
	}
	pb.RegisterTripServiceServer(server, handler)
	return handler
}
func (h *gRPCHandler) PreviewTrip(ctx context.Context, req *pb.PreviewTripRequest) (*pb.PreviewTripResponse, error) {
	pickup := req.GetStartLocation()
	destination := req.GetEndLocation()
	pickupCoordinates := &types.Coordinate{
		Latitude:  pickup.GetLatitude(),
		Longitude: pickup.GetLongitude(),
	}
	destinationCoordinates := &types.Coordinate{
		Latitude:  destination.GetLatitude(),
		Longitude: destination.GetLongitude(),
	}
	route, err := h.service.GetRoute(ctx, pickupCoordinates, destinationCoordinates)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get route: %v", err)
	}
	userID := req.GetUserID()
	estimeatedFares := h.service.EstimatePackagesPriceWithRoute(route)
	fares, err := h.service.GenerateTripFares(ctx, estimeatedFares, userID, route)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate trip fares: %v", err)
	}
	return &pb.PreviewTripResponse{
		Route:     route.ToProto(),
		RideFares: domain.ToRideFaresProto(fares),
	}, nil
}

func (h *gRPCHandler) CreateTrip(ctx context.Context, req *pb.CreateTripRequest) (*pb.CreateTripResponse, error) {
	fareID := req.GetRideFareID()
	userID := req.GetUserID()

	rideFare, err := h.service.GetAndValidateFare(ctx, fareID, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to validate the fare: %v", err)
	}

	trip, err := h.service.CreateTrip(ctx, rideFare)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create the trip: %v", err)
	}

	// Add a comment at the end of the function to publish an event on the Async Comms module.

	return &pb.CreateTripResponse{
		TripID: trip.ID.Hex(),
	}, nil
}
