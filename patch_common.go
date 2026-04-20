package wd

import (
	"errors"

	"gorm.io/gen/field"
)

type patchOldValueState struct {
	known    bool
	value    any
	nullable bool
	isNull   bool
}

func patchApplyUpdate(
	isSet bool,
	wantNull bool,
	loadNewValue func() (any, bool),
	old patchOldValueState,
	equal func(oldValue, newValue any) bool,
	setValue func(any) (field.AssignExpr, error),
	setNull func() (field.AssignExpr, error),
	nullUnsupportedMessage string,
) (field.AssignExpr, bool, error) {
	if !isSet {
		return nil, false, nil
	}

	if !old.known {
		if wantNull {
			if setNull == nil {
				return nil, false, errors.New(nullUnsupportedMessage)
			}
			assignExpr, err := setNull()
			if err != nil {
				return nil, false, err
			}
			return assignExpr, true, nil
		}
		newValue, ok := loadNewValue()
		if !ok {
			return nil, false, nil
		}
		assignExpr, err := setValue(newValue)
		if err != nil {
			return nil, false, err
		}
		return assignExpr, true, nil
	}

	if old.nullable {
		if wantNull {
			if old.isNull {
				return nil, false, nil
			}
			if setNull == nil {
				return nil, false, errors.New(nullUnsupportedMessage)
			}
			assignExpr, err := setNull()
			if err != nil {
				return nil, false, err
			}
			return assignExpr, true, nil
		}

		newValue, ok := loadNewValue()
		if !ok {
			return nil, false, nil
		}
		if !old.isNull && equal(old.value, newValue) {
			return nil, false, nil
		}
		assignExpr, err := setValue(newValue)
		if err != nil {
			return nil, false, err
		}
		return assignExpr, true, nil
	}

	if wantNull {
		return nil, false, nil
	}

	newValue, ok := loadNewValue()
	if !ok {
		return nil, false, nil
	}
	if equal(old.value, newValue) {
		return nil, false, nil
	}

	assignExpr, err := setValue(newValue)
	if err != nil {
		return nil, false, err
	}
	return assignExpr, true, nil
}
