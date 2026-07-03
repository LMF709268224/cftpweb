package handler

import "encoding/json"

func jsonPayloadObject(v interface{}) map[string]interface{} {
	raw, err := json.Marshal(v)
	if err != nil {
		return map[string]interface{}{}
	}

	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]interface{}{}
	}
	return out
}
