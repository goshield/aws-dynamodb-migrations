package migrations_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAwsDynamodbMigrations(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AwsDynamodbMigrations Suite")
}
