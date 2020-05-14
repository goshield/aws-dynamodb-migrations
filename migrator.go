package migrations

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/goshield/interfaces"
)

// NewMigrator returns a migrator for DynamoDB
func NewMigrator(db *dynamodb.DynamoDB, paths []string) interfaces.Migrator {
	return &dynamoDBMigrator{
		db:    db,
		paths: paths,
	}
}

type dynamoDBMigrator struct {
	paths []string
	db    *dynamodb.DynamoDB
}

func (m dynamoDBMigrator) Migrate() {
	im := NewImporter(m.db)
	for _, path := range m.paths {
		err := im.Import(path)
		if err != nil {
			panic(err)
		}
	}
}
