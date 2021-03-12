// +build windows

package main

import (
	"os"
	"testing"
)

func TestNotAdministrator(t *testing.T) {
	isAdmin := isAdmin()
	if os.Getenv("TRAVIS") == "true" {
		if !isAdmin {
			t.Errorf("isAdmin, actual: %t, expected: true", isAdmin)
		}
	} else {
		if isAdmin {
			t.Errorf("isAdmin, actual: %t, expected: false", isAdmin)
		}
	}
}
