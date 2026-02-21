package plugins

import "testing"

func TestExtractTableFromQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{"select from", "SELECT * FROM users WHERE id = 1", "users"},
		{"select from with schema", "SELECT * FROM public.users WHERE id = 1", "public.users"},
		{"insert into", "INSERT INTO orders (id, name) VALUES (1, 'test')", "orders"},
		{"update", "UPDATE products SET name = 'foo' WHERE id = 1", "products"},
		{"select with join", "SELECT * FROM users JOIN orders ON users.id = orders.user_id", "users"},
		{"lowercase", "select * from events where type = 'click'", "events"},
		{"unknown query", "EXPLAIN ANALYZE something", "unknown"},
		{"empty string", "", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTableFromQuery(tt.query)
			if result != tt.expected {
				t.Errorf("extractTableFromQuery(%q) = %q, want %q", tt.query, result, tt.expected)
			}
		})
	}
}
