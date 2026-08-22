package notifier

import "fmt"

var registry = map[string]Notifier{}

// Register adds a notifier to the global registry.
func Register(n Notifier) {
	registry[n.Name()] = n
}

// Get retrieves a notifier by type name.
func Get(name string) (Notifier, error) {
	n, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown notifier type: %s", name)
	}
	return n, nil
}

// List returns all registered notifier type names.
func List() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
