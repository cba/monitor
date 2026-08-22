package monitor

import "fmt"

var registry = map[string]Monitor{}

// Register adds a monitor to the global registry.
// Call from init() in each monitor implementation.
func Register(m Monitor) {
	registry[m.Name()] = m
}

// Get retrieves a monitor by type name.
func Get(name string) (Monitor, error) {
	m, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown monitor type: %s", name)
	}
	return m, nil
}

// List returns all registered monitor type names.
func List() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
