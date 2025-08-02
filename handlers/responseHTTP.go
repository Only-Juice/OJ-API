package handlers

// ResponseHTTP represents response body of this API
type ResponseHTTP struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

type PaginatedData struct {
	Items      interface{} `json:"items"`
	TotalCount int64       `json:"totalCount"`
}