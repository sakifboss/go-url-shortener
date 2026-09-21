package service

import (
	"context"
	"testing"
)

func TestCreateShortURLWithAliasRejectsInvalidAliases(t *testing.T) {
	service := NewURLService(nil)

	for _, alias := range []string{"ab", "has space", "bad/slash", "a$bad"} {
		t.Run(alias, func(t *testing.T) {
			_, err := service.CreateShortURLWithAlias(
				context.Background(),
				"https://example.com",
				alias,
				1,
			)
			if err == nil {
				t.Fatalf("expected alias %q to be rejected", alias)
			}
		})
	}
}
