package request

type GetEventsRequest struct {
	SportKey string `uri:"key"`
}

type GetOddsRequest struct {
	EventID  string `uri:"eventId"`
	SportKey string `query:"sport"`
}
