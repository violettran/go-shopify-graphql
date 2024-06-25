package product_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDiscount(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DiscountService Suite")
}
