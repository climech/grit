package multitree

import (
	"errors"
	"fmt"
	"time"
)

func ValidateNodeName(name string) error {
	if ValidateDateNodeName(name) == nil {
		return errors.New("name is reserved")
	}
	if len(name) == 0 {
		return errors.New("invalid node name (empty name)")
	}
	if len(name) > 100 {
		return errors.New("invalid node name (name too long)")
	}
	return nil
}

func ValidateDateNodeName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("invalid date node name: empty string")
	}
	if _, err := time.Parse("2006-01-02", name); err != nil {
		return fmt.Errorf("invalid date node name: %v", name)
	}
	return nil
}

func ValidateNodeAlias(alias string) error {
	// TODO
	if len(alias) == 0 {
		return errors.New("invalid alias (empty)")
	}
	if len(alias) > 100 {
		return errors.New("invalid alias (too long)")
	}
	return nil
}
