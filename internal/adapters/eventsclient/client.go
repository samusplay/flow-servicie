package eventsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"horizon/flow/internal/domain"
	"horizon/flow/internal/ports"
)

// comprobación al compilar: Client debe cumplir el port EventLookup
var _ ports.EventLookup = (*Client)(nil)

// Client le pregunta al servicio Events por un evento
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New crea el cliente, por ejemplo New("http://localhost:5006")
func New(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 3 * time.Second},
	}
}

// eventResponse es solo la parte de la respuesta de Events que nos interesa
type eventResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (c *Client) Get(ctx context.Context, eventID string) (domain.EventInfo, error) {
	endpoint := c.baseURL + "/api/events/" + url.PathEscape(eventID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return domain.EventInfo{}, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return domain.EventInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return domain.EventInfo{}, ports.ErrEventNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return domain.EventInfo{}, fmt.Errorf("events respondió %d", resp.StatusCode)
	}

	var body eventResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return domain.EventInfo{}, err
	}

	status, err := parseStatus(body.Status)
	if err != nil {
		return domain.EventInfo{}, err
	}

	return domain.EventInfo{ID: body.ID, Status: status}, nil
}

// parseStatus traduce el texto de Events al estado del domain
func parseStatus(s string) (domain.EventStatus, error) {
	switch s {
	case "Scheduled":
		return domain.EventScheduled, nil
	case "Live":
		return domain.EventLive, nil
	case "Finished":
		return domain.EventFinished, nil
	default:
		return 0, fmt.Errorf("estado de evento desconocido: %q", s)
	}
}