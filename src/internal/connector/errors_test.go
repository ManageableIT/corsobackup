package connector

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type GraphConnectorErrorSuite struct {
	suite.Suite
}

func TestGraphConnectorErrorSuite(t *testing.T) {
	suite.Run(t, new(GraphConnectorErrorSuite))
}

func (suite *GraphConnectorErrorSuite) TestWrapAndAppend() {
	err1 := fmt.Errorf("New Error")
	err2 := errors.New("I have two")
	returnErr := WrapAndAppend("arc376", err2, err1)
	suite.True(strings.Contains(returnErr.Error(), "arc376"))
	suite.Error(returnErr)
	multi := &multierror.Error{Errors: []error{err1, err2}}
	suite.True(strings.Contains(ListErrors(*multi), "two")) // Does not contain the wrapped information
	suite.T().Log(ListErrors(*multi))
}

func (suite *GraphConnectorErrorSuite) TestWrapAndAppend_Add3() {
	errOneTwo := WrapAndAppend("user1", assert.AnError, assert.AnError)
	combined := WrapAndAppend("unix36", assert.AnError, errOneTwo)
	allErrors := WrapAndAppend("fxi92874", assert.AnError, combined)
	suite.True(strings.Contains(combined.Error(), "unix36"))
	suite.True(strings.Contains(combined.Error(), "user1"))
	suite.True(strings.Contains(allErrors.Error(), "fxi92874"))

}

func (suite *GraphConnectorErrorSuite) TestWrapAndAppendf() {
	err1 := assert.AnError
	err2 := assert.AnError
	combined := WrapAndAppendf(134323, err2, err1)
	suite.True(strings.Contains(combined.Error(), "134323"))
}

func (suite *GraphConnectorErrorSuite) TestConcatenateStringFromPointers() {
	var s1, s2, s3 *string
	var outString string
	v1 := "Corso"
	v3 := "remains"
	s1 = &v1
	s3 = &v3
	outString = concatenateStringFromPointers(outString, []*string{s1, s2, s3})
	suite.True(strings.Contains(outString, v1))
	suite.True(strings.Contains(outString, v3))
}