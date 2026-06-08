package opskit

import "errors"

var (
	ErrNilComponent                = errors.New("opskit: nil component")
	ErrEmptyComponentName          = errors.New("opskit: component name is required")
	ErrInvalidComponentName        = errors.New("opskit: component name is invalid")
	ErrDuplicateComponent          = errors.New("opskit: component already registered")
	ErrComponentNotFound           = errors.New("opskit: component not found")
	ErrInspectionUnsupported       = errors.New("opskit: component does not support inspection")
	ErrCheckerUnsupported          = errors.New("opskit: component does not support checks")
	ErrCheckDescriberUnsupported   = errors.New("opskit: component does not describe checks")
	ErrCheckGroupUnsupported       = errors.New("opskit: component does not support grouped checks")
	ErrCommandHandlerUnsupported   = errors.New("opskit: component does not support commands")
	ErrCommandDescriberUnsupported = errors.New("opskit: component does not describe commands")
)
