package rest

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
	"go.opentelemetry.io/otel/trace"

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
	EventID string `json:"event_id,omitempty"`
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
	srv.Route("/").GET("/api/v1/products", func(ctx kratoshttp.Context) error {
		page, _ := strconv.Atoi(ctx.Query().Get("page"))
		perPage, _ := strconv.Atoi(ctx.Query().Get("per_page"))
		traceID := trace.SpanFromContext(ctx).SpanContext().TraceID().String()

		resp, err := uc.GetProducts(ctx, int32(page), int32(perPage))
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, Response{
				Meta: Meta{TraceID: traceID, Message: err.Error()},
			})
		}

		p := int32(page)
		pp := int32(perPage)
		t := resp.GetTotalItems()
		return ctx.JSON(http.StatusOK, Response{
			Meta: Meta{
				TraceID: traceID,
				Message: "success",
				Page:    &p,
				PerPage: &pp,
				Total:   &t,
			},
			Data: resp.GetProducts(),
		})
	})

	srv.Route("/").POST("/api/v1/checkout", func(ctx kratoshttp.Context) error {
		var req CheckoutRequest
		if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil {
			return ctx.JSON(http.StatusBadRequest, Response{
				Meta: Meta{TraceID: "", Message: "bad request"},
			})
		}

		traceID := trace.SpanFromContext(ctx).SpanContext().TraceID().String()

		authHeader := ctx.Request().Header.Get("Authorization")
		if authHeader == "" || len(authHeader) < 8 {
			return ctx.JSON(http.StatusUnauthorized, Response{
				Meta: Meta{TraceID: traceID, Message: "missing or invalid token"},
			})
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		userID := token
		if userID == "" {
			return ctx.JSON(http.StatusUnauthorized, Response{
				Meta: Meta{TraceID: traceID, Message: "invalid token"},
			})
		}

		idempKey := ctx.Request().Header.Get("X-Idempotency-Key")

		eventID, success, err := uc.Checkout(ctx, userID, req.ProductID, idempKey)
		if err != nil || !success {
			return ctx.JSON(http.StatusConflict, Response{
				Meta: Meta{TraceID: traceID, EventID: eventID, Message: "stok habis atau sedang diproses"},
			})
		}

		return ctx.JSON(http.StatusAccepted, Response{
			Meta: Meta{TraceID: traceID, EventID: eventID, Message: "pesanan sedang diproses"},
		})
	})

	srv.Route("/").POST("/api/v1/pay", func(ctx kratoshttp.Context) error {
		var req PayRequest
		if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil {
			return ctx.JSON(http.StatusBadRequest, Response{
				Meta: Meta{TraceID: trace.SpanFromContext(ctx).SpanContext().TraceID().String(), Message: "bad request"},
			})
		}
		
		traceID := trace.SpanFromContext(ctx).SpanContext().TraceID().String()

		success, err := uc.ProcessPayment(ctx, req.OrderID, req.Amount)
		if err != nil || !success {
			return ctx.JSON(http.StatusInternalServerError, Response{
				Meta: Meta{TraceID: traceID, Message: "payment failed"},
			})
		}
		
		return ctx.JSON(http.StatusOK, Response{
			Meta: Meta{TraceID: traceID, Message: "payment success"},
		})
	})
}
