package carbon_forecast

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCarbonForecast(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Carbon Forecast Suite")
}
