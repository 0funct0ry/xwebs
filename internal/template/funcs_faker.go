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

	// Address & Location functions
	e.funcs["fakeAddress"] = func() string {
		return gofakeit.Address().Address
	}
	e.funcs["fakeCity"] = func() string {
		return gofakeit.City()
	}
	e.funcs["fakeCountry"] = func() string {
		return gofakeit.Country()
	}
	e.funcs["fakeCountryCode"] = func() string {
		return gofakeit.CountryAbr()
	}
	e.funcs["fakeZipCode"] = func() string {
		return gofakeit.Zip()
	}
	e.funcs["fakeLatitude"] = func(r ...float64) (float64, error) {
		if len(r) >= 2 {
			return gofakeit.LatitudeInRange(r[0], r[1])
		}
		return gofakeit.Latitude(), nil
	}
	e.funcs["fakeLongitude"] = func(r ...float64) (float64, error) {
		if len(r) >= 2 {
			return gofakeit.LongitudeInRange(r[0], r[1])
		}
		return gofakeit.Longitude(), nil
	}
	e.funcs["fakeStreet"] = func() string {
		return gofakeit.Street()
	}
	e.funcs["fakeState"] = func() string {
		return gofakeit.State()
	}

	// Finance functions
	e.funcs["fakePrice"] = func(r ...float64) float64 {
		min := 0.0
		max := 1000.0
		if len(r) >= 2 {
			min = r[0]
			max = r[1]
		}
		return gofakeit.Price(min, max)
	}
	e.funcs["fakeAmount"] = func(r ...float64) float64 {
		min := 0.0
		max := 1000.0
		if len(r) >= 2 {
			min = r[0]
			max = r[1]
		}
		return gofakeit.Price(min, max)
	}
	e.funcs["fakeCurrency"] = func() string {
		return gofakeit.CurrencyShort()
	}
	e.funcs["fakeCreditCard"] = func() string {
		return gofakeit.CreditCardNumber(nil)
	}
	e.funcs["fakeAccountNumber"] = func() string {
		return gofakeit.AchAccount()
	}

	// Commerce & Product functions
	e.funcs["fakeProductName"] = func() string {
		return gofakeit.ProductName()
	}
	e.funcs["fakeCompanyCatchPhrase"] = func() string {
		return gofakeit.BS()
	}
	e.funcs["fakeColor"] = func() string {
		return gofakeit.Color()
	}
	e.funcs["fakeProductCategory"] = func() string {
		return gofakeit.ProductCategory()
	}

	// Text & Content functions
	e.funcs["fakeWord"] = func() string {
		return gofakeit.Word()
	}
	e.funcs["fakeSentence"] = func(count ...int) string {
		wordCount := 5
		if len(count) > 0 {
			wordCount = count[0]
		}
		return gofakeit.Sentence(wordCount)
	}
	e.funcs["fakeParagraph"] = func(count ...int) string {
		sentenceCount := 3
		if len(count) > 0 {
			sentenceCount = count[0]
		}
		return gofakeit.Paragraph(1, sentenceCount, 10, "\n")
	}
	e.funcs["fakeTitle"] = func() string {
		return gofakeit.Sentence(3)
	}
	e.funcs["fakeText"] = e.funcs["fakeParagraph"]
	e.funcs["fakeEmoji"] = func() string {
		return gofakeit.Emoji()
	}
	e.funcs["fakePassword"] = func(length ...int) string {
		l := 12
		if len(length) > 0 {
			l = length[0]
		}
		// lower, upper, numeric, special, space bool, num int
		return gofakeit.Password(true, true, true, true, false, l)
	}

	// Lorem Ipsum functions
	e.funcs["fakeLoremIpsum"] = func(count ...int) string {
		wordCount := 10
		if len(count) > 0 {
			wordCount = count[0]
		}
		return gofakeit.LoremIpsumSentence(wordCount)
	}
}
