package connector

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ExchangeDataCollectionSuite struct {
	suite.Suite
}

func TestExchangeDataCollectionSuite(t *testing.T) {
	suite.Run(t, new(ExchangeDataCollectionSuite))
}

func (suite *ExchangeDataCollectionSuite) TestExchangeDataReader_Valid() {
	m := []byte("test message")
	description := "aFile"
	ed := &ExchangeData{id: description, message: m}

	// Read the message using the `ExchangeData` reader and validate it matches what we set
	buf := &bytes.Buffer{}
	_, err := buf.ReadFrom(ed.ToReader())
	assert.Nil(suite.T(), err, "received a buf.Read error")
	assert.Equal(suite.T(), buf.Bytes(), m)
	assert.Equal(suite.T(), description, ed.UUID())
}

func (suite *ExchangeDataCollectionSuite) TestExchangeDataReader_Empty() {
	var empty []byte
	expected := int64(0)
	ed := &ExchangeData{message: empty}
	buf := &bytes.Buffer{}
	received, err := buf.ReadFrom(ed.ToReader())
	suite.Equal(expected, received)
	assert.Nil(suite.T(), err, "received buf.Readfrom error ")
}
func (suite *ExchangeDataCollectionSuite) TestExchangeData_FullPath() {
	user := "a-user"
	fullPath := []string{"a-tenant", user, "emails"}
	edc := NewExchangeDataCollection(user, fullPath)
	assert.Equal(suite.T(), edc.FullPath(), fullPath)
}

func (suite *ExchangeDataCollectionSuite) TestExchangeDataCollection_NewExchangeDataCollection() {
	name := "User"
	edc := NewExchangeDataCollection(name, []string{"Directory", "File", "task"})
	suite.Equal(name, edc.user)
	suite.True(Contains(edc.FullPath(), "Directory"))
	suite.True(Contains(edc.FullPath(), "File"))
	suite.True(Contains(edc.FullPath(), "task"))
	suite.Zero(edc.Length())
}

func (suite *ExchangeDataCollectionSuite) TestExchangeDataCollection_PopulateCollection() {
	inputStrings := []string{"Jack", "and", "Jill", "went", "up", "the", "hill to",
		"fetch", "a", "pale", "of", "water"}
	expected := len(inputStrings) / 2 // We are using pairs
	edc := NewExchangeDataCollection("Fletcher", []string{"sugar", "horses", "painted red"})
	for i := 0; i < expected; i++ {
		edc.PopulateCollection(ExchangeData{id: inputStrings[i*2], message: []byte(inputStrings[i*2+1])})
	}
	suite.Equal(expected, edc.Length())
}
func (suite *ExchangeDataCollectionSuite) TestExchangeDataCollection_NextItem() {
	inputStrings := []string{"Jack", "and", "Jill", "went", "up", "the", "hill to",
		"fetch", "a", "pale", "of", "water"}
	expected := len(inputStrings) / 2 // We are using pairs
	edc := NewExchangeDataCollection("Fletcher", []string{"sugar", "horses", "painted red"})
	for i := 0; i < expected; i++ {
		edc.PopulateCollection(ExchangeData{id: inputStrings[i*2], message: []byte(inputStrings[i*2+1])})
	}
	edc.FinishPopulation() // finished writing
	for i := 0; i < 6; i++ {
		data, err := edc.NextItem()
		assert.Nil(suite.T(), err)
		assert.NotNil(suite.T(), data)
	}
	// Need that EOF
	data, err := edc.NextItem()
	assert.Nil(suite.T(), data)
	assert.NotNil(suite.T(), err)

}