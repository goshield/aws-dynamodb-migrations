package aws_dynamodb_migrations

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"

	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// Importer is an interface for database importer
type Importer interface {
	// Migrate migrates table from file
	Import(path string) error
}

// NewImporter returns a default Importer instance
func NewImporter(svc *dynamodb.DynamoDB) Importer {
	return factoryImporter{svc: svc}
}

type factoryImporter struct {
	svc *dynamodb.DynamoDB
}

func (m factoryImporter) Import(path string) (err error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	var schema jsonSchema
	err = json.Unmarshal(data, &schema)
	if err != nil {
		return err
	}
	if schema.Table == "" {
		return errors.New("table must be specified")
	}

	actions := make([]func(*jsonSchema) error, 0)
	actions = append(actions, m.deleteTableIfExists)
	actions = append(actions, m.createTable)
	actions = append(actions, m.seedTable)

	for _, action := range actions {
		err = action(&schema)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m factoryImporter) deleteTableIfExists(schema *jsonSchema) (err error) {
	if !schema.DropIfExists {
		return nil
	}
	_, err = m.describeTable(schema.Table)
	if err != nil {
		return nil
	}
	_, err = m.svc.DeleteTable(&dynamodb.DeleteTableInput{TableName: &schema.Table})
	return err
}

func (m factoryImporter) describeTable(table string) (*dynamodb.DescribeTableOutput, error) {
	return m.svc.DescribeTable(&dynamodb.DescribeTableInput{TableName: &table})
}

func (m factoryImporter) createTable(schema *jsonSchema) (err error) {
	in := new(dynamodb.CreateTableInput)

	// ------------ GENERAL ---------------------- //
	in.SetTableName(schema.Table)
	in.SetProvisionedThroughput(&dynamodb.ProvisionedThroughput{
		ReadCapacityUnits:  &schema.ProvisionedThroughput.ReadCapacityUnits,
		WriteCapacityUnits: &schema.ProvisionedThroughput.WriteCapacityUnits,
	})
	// ------------ END GENERAL ---------------------- //

	// ------------ SCHEMA ---------------------- //
	schema.ColumnTypes = make(map[string]string)
	attributes := make([]*dynamodb.AttributeDefinition, 0)
	keys := make([]*dynamodb.KeySchemaElement, 0)
	for _, c := range schema.Columns {
		schema.ColumnTypes[c.Name] = c.Type
		if c.Index {
			attributes = append(attributes, c.ToAttributeDefinition())
		}
		if c.Hash || c.Range {
			keys = append(keys, c.ToKeySchemaElement())
		}
	}
	in.SetAttributeDefinitions(attributes)
	in.SetKeySchema(keys)
	// ------------ SCHEMA ---------------------- //

	// ------------ GLOBAL INDEXES ---------------------- //
	if len(schema.GlobalIndexes) > 0 {
		gbIndexes := make([]*dynamodb.GlobalSecondaryIndex, 0)
		for _, gi := range schema.GlobalIndexes {
			if gi.Name == "" {
				return errors.New("name of GSI must be specified")
			}
			if gi.Projection.Type == "" {
				return errors.New("projection's type of GSI must be specified")
			}
			idx := new(dynamodb.GlobalSecondaryIndex)
			idx.SetIndexName(gi.Name)
			idx.SetProvisionedThroughput(&dynamodb.ProvisionedThroughput{
				ReadCapacityUnits:  &gi.ProvisionedThroughput.ReadCapacityUnits,
				WriteCapacityUnits: &gi.ProvisionedThroughput.WriteCapacityUnits,
			})
			idx.Projection = new(dynamodb.Projection)
			idx.Projection.SetProjectionType(gi.Projection.Type)
			if gi.Projection.NonKeys != nil {
				idx.Projection.SetNonKeyAttributes(gi.Projection.NonKeys)
			}
			keys := make([]*dynamodb.KeySchemaElement, 0)
			for _, c := range gi.Keys {
				keys = append(keys, c.ToKeySchemaElement())
			}
			idx.SetKeySchema(keys)
			gbIndexes = append(gbIndexes, idx)
		}
		in.SetGlobalSecondaryIndexes(gbIndexes)
	}
	// ------------ END GLOBAL INDEXES ---------------------- //

	// ------------ LOCAL INDEXES ---------------------- //
	if len(schema.LocalIndexes) > 0 {
		lcIndexes := make([]*dynamodb.LocalSecondaryIndex, 0)
		for _, li := range schema.GlobalIndexes {
			if li.Name == "" {
				return errors.New("name of LSI must be specified")
			}
			if li.Projection.Type == "" {
				return errors.New("projection's type of LSI must be specified")
			}
			idx := new(dynamodb.LocalSecondaryIndex)
			idx.SetIndexName(li.Name)
			idx.Projection = new(dynamodb.Projection)
			idx.Projection.SetProjectionType(li.Projection.Type)
			if li.Projection.NonKeys != nil {
				idx.Projection.SetNonKeyAttributes(li.Projection.NonKeys)
			}
			keys := make([]*dynamodb.KeySchemaElement, 0)
			for _, c := range li.Keys {
				keys = append(keys, c.ToKeySchemaElement())
			}
			idx.SetKeySchema(keys)
			lcIndexes = append(lcIndexes, idx)
		}
		in.SetLocalSecondaryIndexes(lcIndexes)
	}
	// ------------ END LOCAL INDEXES ---------------------- //

	_, err = m.svc.CreateTable(in)
	return err
}

func (m factoryImporter) seedTable(schema *jsonSchema) (err error) {
	if len(schema.Items) == 0 {
		return nil
	}

	for _, item := range schema.Items {
		iv, err := m.convertMapOfDynamoDBAttributeValue(item)
		if err != nil {
			return err
		}
		pi := new(dynamodb.PutItemInput)
		pi.SetTableName(schema.Table)
		pi.SetItem(iv)
		_, err = m.svc.PutItem(pi)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m factoryImporter) convertMapOfDynamoDBAttributeValue(item map[string]interface{}) (map[string]*dynamodb.AttributeValue, error) {
	iv := make(map[string]*dynamodb.AttributeValue)
	for k, v := range item {
		vv, err := m.convertDynamoDBAttributeValue(v)
		if err != nil {
			return nil, err
		}
		iv[k] = vv
	}
	return iv, nil
}

func (m factoryImporter) convertListOfDynamoDBAttributeValue(items []interface{}) ([]*dynamodb.AttributeValue, error) {
	list := make([]*dynamodb.AttributeValue, len(items))
	for i, v := range items {
		vv, err := m.convertDynamoDBAttributeValue(v)
		if err != nil {
			return nil, err
		}
		list[i] = vv
	}

	return list, nil
}

func (m factoryImporter) convertDynamoDBAttributeValue(v interface{}) (*dynamodb.AttributeValue, error) {
	attr := new(dynamodb.AttributeValue)
	if v == nil || v == "" {
		attr.SetNULL(true)
		return attr, nil
	}
	switch reflect.TypeOf(v).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
	case reflect.Float32, reflect.Float64:
		attr.SetN(fmt.Sprintf("%v", v))
		break
	case reflect.String:
		attr.SetS(fmt.Sprintf("%v", v))
		break
	case reflect.Bool:
		attr.SetBOOL(v.(bool))
		break
	case reflect.Slice:
		lt, ok := v.([]interface{})
		if !ok {
			return nil, fmt.Errorf("Expect value is []interface{}")
		}
		lv, err := m.convertListOfDynamoDBAttributeValue(lt)
		if err != nil {
			return nil, err
		}
		attr.SetL(lv)
		break
	case reflect.Map:
		mp, ok := v.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("Expect value is map[string]interface{}")
		}
		mv, err := m.convertMapOfDynamoDBAttributeValue(mp)
		if err != nil {
			return nil, err
		}
		attr.SetM(mv)
		break
	default:
		return nil, fmt.Errorf("(%s) is not a supporting type", reflect.TypeOf(v).Kind())
	}
	return attr, nil
}
