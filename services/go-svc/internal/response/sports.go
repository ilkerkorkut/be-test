package response

type SportsResponse struct {
	Key          string `json:"key"`
	Group        string `json:"group"`
	Title        string `json:"title"`
	Active       bool   `json:"active"`
	HasOutrights bool   `json:"hasOutrights"`
}
