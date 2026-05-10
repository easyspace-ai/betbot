package store

import (
	"context"
)

// ListDistinctSxEventIDs returns non-empty sx_event_id values for active events/markets.
func (s *Store) ListDistinctSxEventIDs(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT e.sx_event_id FROM events e
		JOIN markets m ON m.event_id = e.id
		WHERE m.status = 'active' AND e.status = 'active'
		  AND e.sx_event_id IS NOT NULL AND TRIM(e.sx_event_id) != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
