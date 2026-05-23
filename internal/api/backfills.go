package api

import (
	"context"
	"fmt"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

const endpointBackfills = "/api/v2/backfills"

// ListBackfills returns backfills for a dag_id (optional). Pass empty dagId to list all.
func (c *Client) ListBackfills(ctx context.Context, dagId string, opts *ListOptions) (*models.BackfillCollection, error) {
	var out models.BackfillCollection
	path := endpointBackfills
	if dagId != "" {
		path += "?dag_id=" + dagId
	}
	if err := c.get(ctx, path, opts, &out); err != nil {
		return nil, fmt.Errorf("list backfills: %w", err)
	}
	return &out, nil
}

func (c *Client) CancelBackfill(ctx context.Context, id int) error {
	endpoint := fmt.Sprintf("%s/%d", endpointBackfills, id)
	return c.delete(ctx, endpoint)
}

func (c *Client) PauseBackfill(ctx context.Context, id int) error {
	endpoint := fmt.Sprintf("%s/%d", endpointBackfills, id)
	var out models.Backfill
	return c.patch(ctx, endpoint, map[string]any{"is_paused": true}, &out)
}

func (c *Client) UnpauseBackfill(ctx context.Context, id int) error {
	endpoint := fmt.Sprintf("%s/%d", endpointBackfills, id)
	var out models.Backfill
	return c.patch(ctx, endpoint, map[string]any{"is_paused": false}, &out)
}

func (c *Client) DryRunBackfill(ctx context.Context, body map[string]any) (*models.DryRunResponse, error) {
	var out models.DryRunResponse
	if err := c.post(ctx, endpointBackfills+"/dryRun", body, &out); err != nil {
		return nil, fmt.Errorf("dry run: %w", err)
	}
	return &out, nil
}
