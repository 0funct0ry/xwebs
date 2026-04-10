package template

import (
	"strings"

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
	e.funcs["fakeURL"] = func(protocol ...string) string {
		url := gofakeit.URL()
		if len(protocol) > 0 {
			parts := strings.SplitN(url, "://", 2)
			if len(parts) == 2 {
				return protocol[0] + "://" + parts[1]
			}
		}
		return url
	}
	e.funcs["fakeDomain"] = func() string {
		return gofakeit.DomainName()
	}
	e.funcs["fakeIPv4"] = func() string {
		return gofakeit.IPv4Address()
	}
	e.funcs["fakeIPv6"] = func() string {
		return gofakeit.IPv6Address()
	}
	e.funcs["fakeUserAgent"] = func(browser ...string) string {
		if len(browser) > 0 {
			switch strings.ToLower(browser[0]) {
			case "chrome":
				return gofakeit.ChromeUserAgent()
			case "firefox":
				return gofakeit.FirefoxUserAgent()
			case "safari":
				return gofakeit.SafariUserAgent()
			case "opera":
				return gofakeit.OperaUserAgent()
			}
		}
		return gofakeit.UserAgent()
	}
	e.funcs["fakeHTTPMethod"] = func() string {
		return gofakeit.HTTPMethod()
	}

	// Network functions
	e.funcs["fakeMacAddress"] = func() string {
		return gofakeit.MacAddress()
	}
	e.funcs["fakePort"] = func(r ...int) int {
		if len(r) >= 2 {
			return gofakeit.Number(r[0], r[1])
		}
		return int(gofakeit.Uint16())
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
