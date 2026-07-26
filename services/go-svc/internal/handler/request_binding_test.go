package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flotaro/be-test/services/go-svc/internal/enums"
	"github.com/flotaro/be-test/services/go-svc/internal/middleware"
	"github.com/flotaro/be-test/services/go-svc/internal/request"
	"github.com/flotaro/be-test/services/go-svc/internal/response"
	"github.com/flotaro/be-test/services/go-svc/internal/service"
	"github.com/gofiber/fiber/v3"
)

func TestHandlersBindDomainRequests(t *testing.T) {
	auth := authMiddlewareStub{userID: "user-1"}

	t.Run("place bet body", func(t *testing.T) {
		service := &betServiceStub{}
		app := fiber.New()
		app.Post("/bets", NewBetsHandler(service, auth).Place)

		response := testRequest(t, app, http.MethodPost, "/bets", `{"userId":"user-1","eventId":"event-1","sportKey":"soccer_epl","market":"h2h","selection":"Home","stake":10,"odds":2}`)
		if response.StatusCode != fiber.StatusCreated {
			t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusCreated)
		}
		if service.placeRequest.EventID != "event-1" || service.placeRequest.Stake != 10 {
			t.Fatalf("unexpected place request: %+v", service.placeRequest)
		}
	})

	t.Run("bet history query", func(t *testing.T) {
		service := &betServiceStub{}
		app := fiber.New()
		app.Get("/bets", NewBetsHandler(service, auth).Get)

		response := testRequest(t, app, http.MethodGet, "/bets?userId=user-1&limit=7", "")
		if response.StatusCode != fiber.StatusOK {
			t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusOK)
		}
		if service.getRequest != (request.GetBetsRequest{UserID: "user-1", Limit: 7}) {
			t.Fatalf("unexpected bet history request: %+v", service.getRequest)
		}
	})

	t.Run("event and odds path query", func(t *testing.T) {
		service := &sportsServiceStub{}
		handler := NewEventsHandler(service)
		app := fiber.New()
		app.Get("/sports/:key/events", handler.GetEvents)
		app.Get("/events/:eventId/odds", handler.GetOdds)

		response := testRequest(t, app, http.MethodGet, "/sports/soccer_epl/events", "")
		if response.StatusCode != fiber.StatusOK || service.eventsRequest.SportKey != "soccer_epl" {
			t.Fatalf("unexpected event request: status=%d request=%+v", response.StatusCode, service.eventsRequest)
		}

		response = testRequest(t, app, http.MethodGet, "/events/event-1/odds?sport=soccer_epl", "")
		if response.StatusCode != fiber.StatusOK || service.oddsRequest != (request.GetOddsRequest{EventID: "event-1", SportKey: "soccer_epl"}) {
			t.Fatalf("unexpected odds request: status=%d request=%+v", response.StatusCode, service.oddsRequest)
		}
	})

	t.Run("wallet path", func(t *testing.T) {
		service := &walletServiceStub{}
		app := fiber.New()
		app.Get("/wallet/:userId", NewWalletHandler(service, auth).Get)

		response := testRequest(t, app, http.MethodGet, "/wallet/user-1", "")
		if response.StatusCode != fiber.StatusOK {
			t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusOK)
		}
		if service.getRequest.UserID != "user-1" {
			t.Fatalf("unexpected wallet request: %+v", service.getRequest)
		}
	})
}

func testRequest(t *testing.T, app *fiber.App, method string, target string, body string) *http.Response {
	t.Helper()

	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("test request: %v", err)
	}

	return response
}

type authMiddlewareStub struct {
	userID string
}

func (m authMiddlewareStub) RequireJWT(c fiber.Ctx) error {
	return c.Next()
}

func (m authMiddlewareStub) UserID(_ fiber.Ctx) string {
	return m.userID
}

var _ middleware.AuthMiddleware = authMiddlewareStub{}

type betServiceStub struct {
	placeRequest request.PlaceBetRequest
	getRequest   request.GetBetsRequest
}

func (s *betServiceStub) Place(_ context.Context, req request.PlaceBetRequest) (*response.BetResponse, error) {
	s.placeRequest = req
	return &response.BetResponse{BetID: "bet-1", Status: enums.BetStatusPending}, nil
}

func (s *betServiceStub) Get(_ context.Context, req request.GetBetsRequest) ([]response.BetHistoryResponse, error) {
	s.getRequest = req
	return []response.BetHistoryResponse{}, nil
}

var _ service.BetService = (*betServiceStub)(nil)

type sportsServiceStub struct {
	eventsRequest request.GetEventsRequest
	oddsRequest   request.GetOddsRequest
}

func (s *sportsServiceStub) GetSports(_ context.Context) ([]response.SportsResponse, error) {
	return []response.SportsResponse{}, nil
}

func (s *sportsServiceStub) GetEvents(_ context.Context, req request.GetEventsRequest) ([]response.EventResponse, error) {
	s.eventsRequest = req
	return []response.EventResponse{}, nil
}

func (s *sportsServiceStub) GetOdds(_ context.Context, req request.GetOddsRequest) (*response.OddsResponse, error) {
	s.oddsRequest = req
	return &response.OddsResponse{}, nil
}

var _ service.SportsService = (*sportsServiceStub)(nil)

type walletServiceStub struct {
	getRequest request.GetWalletRequest
}

func (s *walletServiceStub) Get(_ context.Context, req request.GetWalletRequest) (*response.WalletResponse, error) {
	s.getRequest = req
	return &response.WalletResponse{}, nil
}

func (s *walletServiceStub) Credit(_ context.Context, _ string, _ float64, _ string) (*response.WalletResponse, error) {
	return nil, nil
}

func (s *walletServiceStub) Debit(_ context.Context, _ string, _ float64, _ string) (*response.WalletResponse, error) {
	return nil, nil
}

var _ service.WalletService = (*walletServiceStub)(nil)
