package rest

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"

	"flashsale/api-gateway/internal/application/usecase"
)

// Response adalah format standar sesuai response-standard.md
type Response struct {
	Meta Meta `json:"meta"`
	Data any  `json:"data,omitempty"`
}

type Meta struct {
	TraceID string `json:"trace_id"`
	Message string `json:"message"`
	Page    *int32 `json:"page,omitempty"`
	PerPage *int32 `json:"per_page,omitempty"`
	Total   *int32 `json:"total,omitempty"`
}

type CheckoutRequest struct {
	ProductID string `json:"product_id"`
}

type PayRequest struct {
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
}

func RegisterHTTPServer(srv *kratoshttp.Server, uc *usecase.GatewayUsecase, logger log.Logger) {
	log := log.NewHelper(logger)

	srv.Route("/").GET("/api/v1/products", func(ctx kratoshttp.Context) error {
		page, _ := strconv.Atoi(ctx.Query().Get("page"))
		perPage, _ := strconv.Atoi(ctx.Query().Get("per_page"))

		resp, err := uc.GetProducts(ctx, int32(page), int32(perPage))
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, Response{
				Meta: Meta{TraceID: "HTTP-TRACE", Message: err.Error()},
			})
		}

		// Mapping gRPC response to standard REST DTO
		p := resp.Meta.Page
		pp := resp.Meta.PerPage
		t := resp.Meta.Total
		return ctx.JSON(http.StatusOK, Response{
			Meta: Meta{
				TraceID: "HTTP-TRACE", // Idealnya dari OpenTelemetry
				Message: resp.Meta.Message,
				Page:    &p,
				PerPage: &pp,
				Total:   &t,
			},
			Data: resp.Data,
		})
	})

	srv.Route("/").POST("/api/v1/checkout", func(ctx kratoshttp.Context) error {
		// 1. Ambil token JWT sederhana (scaffold authentication)
		authHeader := ctx.Request().Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return ctx.JSON(http.StatusUnauthorized, Response{
				Meta: Meta{TraceID: "HTTP-TRACE", Message: "unauthorized"},
			})
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		// Dummy stateless validation: token = user_id
		userID := token
		if userID == "" {
			return ctx.JSON(http.StatusUnauthorized, Response{
				Meta: Meta{TraceID: "HTTP-TRACE", Message: "invalid token"},
			})
		}

		// 2. Parse Request
		var req CheckoutRequest
		if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil {
			return ctx.JSON(http.StatusBadRequest, Response{
				Meta: Meta{TraceID: "HTTP-TRACE", Message: "bad request"},
			})
		}

		// 3. Proses Checkout
		eventID, success, err := uc.Checkout(ctx, userID, req.ProductID)
		if err != nil || !success {
			return ctx.JSON(http.StatusConflict, Response{
				Meta: Meta{TraceID: eventID, Message: "stok habis atau sedang diproses"},
			})
		}

		// 4. Return Accepted (karena proses order sesungguhnya asynchronous via Kafka)
		return ctx.JSON(http.StatusAccepted, Response{
			Meta: Meta{TraceID: eventID, Message: "pesanan sedang diproses"},
		})
	})

	srv.Route("/").POST("/api/v1/pay", func(ctx kratoshttp.Context) error {
		var req PayRequest
		if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil {
			return ctx.JSON(http.StatusBadRequest, Response{
				Meta: Meta{TraceID: "HTTP-TRACE", Message: "bad request"},
			})
		}

		success, err := uc.ProcessPayment(ctx, req.OrderID, req.Amount)
		if err != nil || !success {
			return ctx.JSON(http.StatusInternalServerError, Response{
				Meta: Meta{TraceID: "HTTP-TRACE", Message: "payment failed"},
			})
		}

		return ctx.JSON(http.StatusOK, Response{
			Meta: Meta{TraceID: "HTTP-TRACE", Message: "payment success"},
		})
	})
}
