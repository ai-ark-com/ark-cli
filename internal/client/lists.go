package client

import "context"

// List types.
const (
	ListPeople    = "people_id"
	ListCompanies = "company_id"
)

// List update modes.
const (
	ListAppend  = "APPEND"  // merge values into the list (API default)
	ListReplace = "REPLACE" // overwrite the list
)

// ListRequest creates or updates a suppression list. Lists are free, hold up
// to 10,000 ids, and expire 24 hours after creation.
type ListRequest struct {
	ID     string   `json:"id,omitempty"`   // existing list to update; omit to create
	Type   string   `json:"type,omitempty"` // required when creating
	Values []string `json:"values"`
	Mode   string   `json:"mode,omitempty"`
}

// SaveList creates a list (no ID) or updates one (with ID).
func (c *Client) SaveList(ctx context.Context, req *ListRequest) (*Result, error) {
	if req.ID != "" {
		if err := ValidateID("list id", req.ID); err != nil {
			return nil, err
		}
	}
	return c.post(ctx, "/v1/lists", req)
}
