package grpc

import (
	"context"
	"errors"

	pb "flashsale/proto/payment/v1"
	"flashsale/payment-service/internal/application/usecase"
)

type PaymentServer struct {
	pb.UnimplementedPaymentServiceServer
	usecase *usecase.ProcessPaymentUsecase
}

func NewPaymentServer(uc *usecase.ProcessPaymentUsecase) *PaymentServer {
	return &PaymentServer{
		usecase: uc,
	}
}

func (s *PaymentServer) ProcessPayment(ctx context.Context, req *pb.ProcessPaymentRequest) (*pb.ProcessPaymentResponse, error) {
	success, err := s.usecase.Execute(ctx, req.GetOrderId(), req.GetAmount())
	if err != nil {
		return nil, err
	}
	if !success {
		return nil, errors.New("payment gagal diproses")
	}

	return &pb.ProcessPaymentResponse{
		PaymentId:  req.GetOrderId() + "-pay",
		PaymentUrl: "https://mock-payment.example.com/pay/" + req.GetOrderId(),
	}, nil
}
