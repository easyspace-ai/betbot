package heartbeat

type HeartbeatRequest struct {
	HeartbeatID string `json:"heartbeat_id,omitempty"`
}

type HeartbeatResponse struct {
	Status string `json:"status"`
}
