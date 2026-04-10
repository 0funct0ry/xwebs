package template

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakerFuncs(t *testing.T) {
	e := New(false)

	tests := []struct {
		name     string
		template string
	}{
		{"fakeName", "{{fakeName}}"},
		{"fakeFirstName", "{{fakeFirstName}}"},
		{"fakeLastName", "{{fakeLastName}}"},
		{"fakeEmail", "{{fakeEmail}}"},
		{"fakeUsername", "{{fakeUsername}}"},
		{"fakePhone", "{{fakePhone}}"},
		{"fakeCompany", "{{fakeCompany}}"},
		{"fakeURL", "{{fakeURL}}"},
		{"fakeURL_https", `{{fakeURL "https"}}`},
		{"fakeDomain", "{{fakeDomain}}"},
		{"fakeIPv4", "{{fakeIPv4}}"},
		{"fakeIPv6", "{{fakeIPv6}}"},
		{"fakeUserAgent", "{{fakeUserAgent}}"},
		{"fakeUserAgent_chrome", `{{fakeUserAgent "chrome"}}`},
		{"fakeHTTPMethod", "{{fakeHTTPMethod}}"},
		{"fakeMacAddress", "{{fakeMacAddress}}"},
		{"fakePort", "{{fakePort}}"},
		{"fakePort_range", "{{fakePort 8000 9000}}"},
		{"fakeUUID", "{{fakeUUID}}"},
		{"fakeULID", "{{fakeULID}}"},
		{"fakeAddress", "{{fakeAddress}}"},
		{"fakeCity", "{{fakeCity}}"},
		{"fakeCountry", "{{fakeCountry}}"},
		{"fakeCountryCode", "{{fakeCountryCode}}"},
		{"fakeZipCode", "{{fakeZipCode}}"},
		{"fakeLatitude", "{{fakeLatitude}}"},
		{"fakeLatitude_range", "{{fakeLatitude 10.0 20.0}}"},
		{"fakeLongitude", "{{fakeLongitude}}"},
		{"fakeLongitude_range", "{{fakeLongitude 10.0 20.0}}"},
		{"fakeStreet", "{{fakeStreet}}"},
		{"fakeState", "{{fakeState}}"},
		{"fakePrice", "{{fakePrice}}"},
		{"fakePrice_range", "{{fakePrice 100.0 200.0}}"},
		{"fakeAmount", "{{fakeAmount}}"},
		{"fakeCurrency", "{{fakeCurrency}}"},
		{"fakeCreditCard", "{{fakeCreditCard}}"},
		{"fakeAccountNumber", "{{fakeAccountNumber}}"},
		{"fakeProductName", "{{fakeProductName}}"},
		{"fakeCompanyCatchPhrase", "{{fakeCompanyCatchPhrase}}"},
		{"fakeColor", "{{fakeColor}}"},
		{"fakeProductCategory", "{{fakeProductCategory}}"},
		{"fakeWord", "{{fakeWord}}"},
		{"fakeSentence", "{{fakeSentence}}"},
		{"fakeSentence_10", "{{fakeSentence 10}}"},
		{"fakeParagraph", "{{fakeParagraph}}"},
		{"fakeParagraph_5", "{{fakeParagraph 5}}"},
		{"fakeTitle", "{{fakeTitle}}"},
		{"fakeText", "{{fakeText}}"},
		{"fakeEmoji", "{{fakeEmoji}}"},
		{"fakePassword", "{{fakePassword}}"},
		{"fakePassword_20", "{{fakePassword 20}}"},
		{"fakeLoremIpsum", "{{fakeLoremIpsum}}"},
		{"fakeLoremIpsum_20", "{{fakeLoremIpsum 20}}"},
		{"fakePastDate", "{{fakePastDate}}"},
		{"fakePastDate_30", "{{fakePastDate 30}}"},
		{"fakePastDate_format", `{{fakePastDate 30 "2006-01-02"}}`},
		{"fakeFutureDate", "{{fakeFutureDate}}"},
		{"fakeRecentDate", "{{fakeRecentDate}}"},
		{"fakeTimestamp", "{{fakeTimestamp}}"},
		{"fakeUnixTime", "{{fakeUnixTime}}"},
		{"fakeOrderID", "{{fakeOrderID}}"},
		{"fakeTransactionID", "{{fakeTransactionID}}"},
		{"fakeSessionID", "{{fakeSessionID}}"},
		{"fakeHexColor", "{{fakeHexColor}}"},
		{"fakeImageURL", "{{fakeImageURL}}"},
		{"fakeImageURL_size", "{{fakeImageURL 800 600}}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test first call
			res1, err := e.Execute(tt.name, tt.template, nil)
			require.NoError(t, err)
			assert.NotEmpty(t, res1)

			// Test second call to ensure variation (statistically likely to be different)
			res2, err := e.Execute(tt.name, tt.template, nil)
			require.NoError(t, err)
			assert.NotEmpty(t, res2)

			// Specific validations
			switch tt.name {
			case "fakeEmail":
				assert.Contains(t, res1, "@")
			case "fakeURL_https":
				assert.True(t, strings.HasPrefix(res1, "https://"))
			case "fakeIPv4":
				assert.Contains(t, res1, ".")
			case "fakeIPv6":
				assert.Contains(t, res1, ":")
			case "fakeMacAddress":
				assert.Contains(t, res1, ":")
			case "fakeUUID":
				assert.Len(t, res1, 36) // Standard UUID length
			case "fakeULID":
				assert.Len(t, res1, 26) // Standard ULID length
			case "fakeCurrency":
				assert.Len(t, res1, 3) // Typically 3-letter currency code (USD, EUR, etc)
			case "fakeCreditCard":
				assert.GreaterOrEqual(t, len(res1), 13) // Basic CC length check
			case "fakeAccountNumber":
				assert.NotEmpty(t, res1)
			case "fakePassword_20":
				assert.Len(t, res1, 20)
			case "fakeEmoji":
				assert.NotEmpty(t, res1)
			case "fakePastDate_format":
				assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, res1)
			case "fakePastDate", "fakeFutureDate", "fakeRecentDate", "fakeTimestamp":
				assert.Contains(t, res1, "T") // RFC3339 typically has T
			case "fakeOrderID":
				assert.True(t, strings.HasPrefix(res1, "ORD-"))
			case "fakeTransactionID":
				assert.True(t, strings.HasPrefix(res1, "TXN-"))
			case "fakeHexColor":
				assert.True(t, strings.HasPrefix(res1, "#"))
				assert.Len(t, res1, 7)
			case "fakeImageURL_size":
				assert.Contains(t, res1, "800/600")
			}
		})
	}
}

func TestFakerFunctionsVariability(t *testing.T) {
    e := New(false)
    
    // Test that multiple calls produce different results
    // We use a loop to ensure we don't just get lucky with the same name twice
    // (though in reality the chance of two random names being the same is low)
    
    seen := make(map[string]bool)
    for i := 0; i < 10; i++ {
        res, err := e.Execute("test", "{{fakeName}}", nil)
        require.NoError(t, err)
        seen[res] = true
    }
    
    // We expect to have seen multiple different names
    assert.Greater(t, len(seen), 1, "Expected varied output from fakeName")
}

func TestFakerAvailabilityInJSON(t *testing.T) {
    e := New(false)
    tmpl := `{"name": "{{fakeName}}", "email": "{{fakeEmail}}"}`
    res, err := e.Execute("json", tmpl, nil)
    require.NoError(t, err)
    
    assert.True(t, strings.HasPrefix(res, `{"name": "`))
    assert.Contains(t, res, `"email": "`)
}
