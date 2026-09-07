package storage

import "testing"

func TestValidateScheduleMinutes(t *testing.T) {
	valid := []int{15, 60, 1440, 10080}
	for _, value := range valid {
		minutes := value
		if err := validateScheduleMinutes(&minutes); err != nil {
			t.Fatalf("expected %d to be valid: %v", value, err)
		}
	}
	invalid := []int{0, 14, 10081}
	for _, value := range invalid {
		minutes := value
		if err := validateScheduleMinutes(&minutes); err == nil {
			t.Fatalf("expected %d to be invalid", value)
		}
	}
	if err := validateScheduleMinutes(nil); err != nil {
		t.Fatalf("nil schedule should disable recurrence: %v", err)
	}
}
