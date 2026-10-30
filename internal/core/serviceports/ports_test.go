package serviceports

import (
	"sort"
	"strconv"
	"testing"
)

// TestAllocation protects permanent identities and gap-free independent service allocation.
func TestAllocation(t *testing.T) {
	if Port("frontend") != "24691" || Port("backend") != "24692" {
		t.Fatal("application listeners changed")
	}
	var values []int
	for _, name := range Names() {
		value, err := strconv.Atoi(Port(name))
		if err != nil {
			t.Fatal(err)
		}
		values = append(values, value)
	}
	sort.Ints(values)
	for i, value := range values {
		if value != 24691+i {
			t.Fatalf("allocation gap or duplicate: %v", values)
		}
	}
}

// TestValidator rejects ephemeral, alternate and malformed listener assignments.
func TestValidator(t *testing.T) {
	for _, value := range []string{"0", "24693", "", "random", "-1"} {
		if Validator("backend")(value) == nil {
			t.Fatalf("accepted override %q", value)
		}
	}
	if err := Validator("backend")(Port("backend")); err != nil {
		t.Fatal(err)
	}
}
