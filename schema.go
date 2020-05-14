package migrations

import "github.com/aws/aws-sdk-go/service/dynamodb"

type jsonSchema struct {
	Table                 string                `json:"table"`
	DropIfExists          bool                  `json:"dropIfExists"`
	ProvisionedThroughput provisionedThroughput `json:"provisionedThroughput"`
	Columns               []column              `json:"columns"`
	GlobalIndexes         []struct {
		Name                  string                `json:"name"`
		ProvisionedThroughput provisionedThroughput `json:"provisionedThroughput"`
		Projection            projection            `json:"projection"`
		Keys                  []column              `json:"keys"`
	} `json:"globalIndexes,omitempty"`
	LocalIndexes []struct {
		Name       string     `json:"name"`
		Projection projection `json:"projection"`
		Keys       []column   `json:"keys"`
	} `json:"localIndexes,omitempty"`
	Items       []map[string]interface{} `json:"items,omitempty"`
	ColumnTypes map[string]string        `json:"-"`
}

type column struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Index bool   `json:"index"`
	Hash  bool   `json:"hash"`
	Range bool   `json:"range"`
}

func (c column) ToAttributeDefinition() *dynamodb.AttributeDefinition {
	return &dynamodb.AttributeDefinition{
		AttributeName: &c.Name,
		AttributeType: &c.Type,
	}
}

func (c column) ToKeySchemaElement() *dynamodb.KeySchemaElement {
	key := new(dynamodb.KeySchemaElement)
	key.SetAttributeName(c.Name)
	if c.Hash {
		key.SetKeyType(dynamodb.KeyTypeHash)
	} else if c.Range {
		key.SetKeyType(dynamodb.KeyTypeRange)
	}
	return key
}

type projection struct {
	Type    string    `json:"type"`
	NonKeys []*string `json:"nonKeys"`
}

type provisionedThroughput struct {
	ReadCapacityUnits  int64 `json:"readCapacityUnits"`
	WriteCapacityUnits int64 `json:"writeCapacityUnits"`
}
