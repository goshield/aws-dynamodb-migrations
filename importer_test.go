package aws_dynamodb_migrations

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AwsDynamodbMigrations", func() {
	var im Importer
	BeforeEach(func() {
		ss := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))
		im = NewImporter(dynamodb.New(ss, &aws.Config{Endpoint: aws.String("http://127.0.0.1:28822")}))
	})
	It("should return error due to file does not exist", func() {
		err := im.Import("invalid_file.json")
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(Equal("open invalid_file.json: no such file or directory"))
	})
	It("should return error due to invalid json", func() {
		dir, err := os.Getwd()
		Expect(err).To(BeNil())
		err = im.Import(fmt.Sprintf("%s/fixtures/invalid.json", dir))
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(Equal("invalid character '}' looking for beginning of object key string"))
	})
	It("should import schema without errors", func() {
		dir, err := os.Getwd()
		Expect(err).To(BeNil())
		err = im.Import(fmt.Sprintf("%s/fixtures/schema.json", dir))
		Expect(err).To(BeNil())
	})
})
