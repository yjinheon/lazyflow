package api

import (
	"context"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

// ListPools fetches all worker pools.
func (c *Client) ListPools(ctx context.Context, opts *ListOptions) (*models.PoolCollection, error) {
	var out models.PoolCollection
	if err := c.get(ctx, EndpointPools, opts, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
