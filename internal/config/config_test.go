package config

import "testing"

// TASK-4 AC#2: the config name must be a valid Avro (full)name.
func TestCheckAvroName(t *testing.T) {
	valid := []string{"orders", "_orders", "Orders_2024", "com.example.orders", "a.b_c.D9"}
	for _, name := range valid {
		if err := CheckAvroName(name); err != nil {
			t.Errorf("%q: unexpected error: %v", name, err)
		}
	}
	invalid := []string{"", "2024_orders", "my-orders", "my orders", "orders.", ".orders", "com..orders", "com.2x.orders", "ordérs"}
	for _, name := range invalid {
		if err := CheckAvroName(name); err == nil {
			t.Errorf("%q: expected an error", name)
		}
	}
}
