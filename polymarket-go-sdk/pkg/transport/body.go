package transport

import (
	"encoding/json"
	"fmt"
)

// MarshalBody marshals a request body to JSON and returns a serialized string for signing.
func MarshalBody(body any) ([]byte, *string, error) {
	if body == nil {
		return nil, nil, nil
	}

	switch v := body.(type) {
	case []byte:
		str := string(v)
		return v, &str, nil
	case json.RawMessage:
		str := string(v)
		return v, &str, nil
	case string:
		str := v
		return []byte(v), &str, nil
	default:
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal body: %w", err)
		}
		serialized := string(payload)
		return payload, &serialized, nil
	}
}
