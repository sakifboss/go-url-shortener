package auth

import "testing"

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "Password123",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
		},
		{
			name:     "too short",
			password: "Pass123",
			wantErr:  true,
		},
		{
			name:     "no letter",
			password: "12345678",
			wantErr:  true,
		},
		{
			name:     "no number",
			password: "Password",
			wantErr:  true,
		},
		{
			name:     "72 characters",
			password: "Abcdefgh1234567890Abcdefgh1234567890Abcdefgh1234567890Abcdefgh1234567890",
			wantErr:  false,
		},
		{
			name:     "more than 72 characters",
			password: "Abcdefgh1234567890Abcdefgh1234567890Abcdefgh1234567890Abcdefgh1234567890X",
			wantErr:  true,
		},
		{
			name:     "letters and numbers",
			password: "GoShort2026",
			wantErr:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validatePassword(test.password)

			if test.wantErr && err == nil {
				t.Fatalf(
					"expected error for password %q",
					test.password,
				)
			}

			if !test.wantErr && err != nil {
				t.Fatalf(
					"unexpected error: %v",
					err,
				)
			}
		})
	}
}
