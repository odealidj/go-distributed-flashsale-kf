package grpc

import (
	"context"

	"flashsale/payment-service/internal/application/usecase"
	pb "flashsale/proto/payment/v1"
)

type PaymentServer struct {
	pb.UnimplementedPaymentServiceServer
	usecase *usecase.ProcessPaymentUsecase
}

func NewPaymentServer(uc *usecase.ProcessPaymentUsecase) *PaymentServer {
	return &PaymentServer{usecase: uc}
}

func (s *PaymentServer) ProcessPayment(ctx context.Context, req *pb.ProcessPaymentRequest) (*pb.ProcessPaymentResponse, error) {
	success, err := s.usecase.Execute(ctx, req.GetOrderId(), req.GetAmount())
	if err != nil {
		return &pb.ProcessPaymentResponse{
			Meta: &pb.ProcessPaymentResponse_Meta{
				TraceId: "grpc-trace",
				Message: err.Error(),
			},
			Data: &pb.ProcessPaymentResponse_Data{
				Success: false,
			},
		}, nil
	}

	return &pb.ProcessPaymentResponse{
		Meta: &pb.ProcessPaymentResponse_Meta{
			TraceId: "grpc-trace",
			Message: "Payment processing successfully initiated",
		},
		Data: &pb.ProcessPaymentResponse_Data{
			Success: success,
		},
	}, nil
}
