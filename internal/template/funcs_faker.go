package template

import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// registerFakerFuncs adds fake data generation functions to the engine.
func (e *Engine) registerFakerFuncs() {
	// Name functions
	e.funcs["fakeName"] = func() string {
		return gofakeit.Name()
	}
	e.funcs["fakeFirstName"] = func() string {
		return gofakeit.FirstName()
	}
	e.funcs["fakeLastName"] = func() string {
		return gofakeit.LastName()
	}

	// Internet functions
	e.funcs["fakeEmail"] = func() string {
		return gofakeit.Email()
	}
	e.funcs["fakeUsername"] = func() string {
		return gofakeit.Username()
	}

	// Business/Contact functions
	e.funcs["fakePhone"] = func() string {
		return gofakeit.Phone()
	}
	e.funcs["fakeCompany"] = func() string {
		return gofakeit.Company()
	}

	// ID functions (aliases to existing ones with fake prefix)
	e.funcs["fakeUUID"] = func() string {
		return uuid.New().String()
	}
	e.funcs["fakeULID"] = func() string {
		return ulid.Make().String()
	}
}
