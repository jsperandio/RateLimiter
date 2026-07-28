package handler

type DummyResponse struct {
	Kind      string `json:"kind"`
	Limit     int    `json:"limit"`
	Remaining int    `json:"remaining"`
}
