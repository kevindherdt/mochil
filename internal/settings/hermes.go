package settings

import "strings"

// The Hermes key never leaves the server: Read replaces it with key_configured and an
// update that leaves the field blank keeps the saved key.
func redactHermesKey(document map[string]any) {
	hermes, ok := document["hermes"].(map[string]any)
	if !ok {
		return
	}
	if key, _ := hermes["key"].(string); key != "" {
		hermes["key_configured"] = true
	}
	delete(hermes, "key")
}

func preserveHermesKey(incoming map[string]any, readSource func() ([]byte, error)) {
	hermes, ok := incoming["hermes"].(map[string]any)
	if !ok {
		return
	}
	delete(hermes, "key_configured")
	if key, _ := hermes["key"].(string); strings.TrimSpace(key) != "" {
		return
	}
	delete(hermes, "key")
	if readSource == nil {
		return
	}
	data, err := readSource()
	if err != nil {
		return
	}
	current, err := decode(data)
	if err != nil {
		return
	}
	saved, _ := current["hermes"].(map[string]any)
	if key, _ := saved["key"].(string); key != "" {
		hermes["key"] = key
	}
}
