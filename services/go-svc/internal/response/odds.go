package response

type OddsResponse struct {
	ID           string      `json:"id"`
	SportKey     string      `json:"sportKey"`
	CommenceTime string      `json:"commenceTime"`
	HomeTeam     string      `json:"homeTeam"`
	AwayTeam     string      `json:"awayTeam"`
	Bookmakers   []Bookmaker `json:"bookmakers"`
}

type Bookmaker struct {
	Key     string   `json:"key"`
	Title   string   `json:"title"`
	Markets []Market `json:"markets"`
}

type Market struct {
	Key      string    `json:"key"`
	Outcomes []Outcome `json:"outcomes"`
}

type Outcome struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}
