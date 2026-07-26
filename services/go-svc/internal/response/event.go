package response

type EventResponse struct {
	ID           string `json:"id"`
	SportKey     string `json:"sportKey"`
	CommenceTime string `json:"commenceTime"`
	HomeTeam     string `json:"homeTeam"`
	AwayTeam     string `json:"awayTeam"`
}
