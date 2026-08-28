package handlers

import "testing"

func TestEventValidate(t *testing.T) {
	cases := []struct {
		name    string
		event   Event
		wantErr bool
	}{
		{
			name: "valid event",
			event: Event{
				EventType: "order.created",
				UserID:    "user_123",
				Timestamp: "2026-08-26T12:30:00Z",
			},
			wantErr: false,
		},
		{
			name:    "missing event_type",
			event:   Event{UserID: "user_123", Timestamp: "2026-08-26T12:30:00Z"},
			wantErr: true,
		},
		{
			name:    "missing user_id",
			event:   Event{EventType: "user.login", Timestamp: "2026-08-26T12:30:00Z"},
			wantErr: true,
		},
		{
			name:    "bad timestamp format",
			event:   Event{EventType: "user.login", UserID: "user_123", Timestamp: "not-a-time"},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			problems := tc.event.Validate()
			if tc.wantErr && len(problems) == 0 {
				t.Errorf("expected validation errors, got none")
			}
			if !tc.wantErr && len(problems) > 0 {
				t.Errorf("expected no validation errors, got %v", problems)
			}
		})
	}
}
