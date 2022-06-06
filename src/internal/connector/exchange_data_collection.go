package connector

import "io"

// A DataCollection represents a collection of data of the
// same type (e.g. mail)
type DataCollection interface {
	// Returns either the next item in the collection or an error if one occurred.
	// If not more items are available in the collection, returns (nil, nil).
	NextItem() (DataStream, error)
}

// DataStream represents a single item within a DataCollection
// that can be consumed as a stream (it embeds io.Reader)
type DataStream interface {
	io.Reader
	// Provides a unique identifier for this data
	UUID() string
}

// ExchangeDataCollection represents exchange mailbox
// data for a single user.
//
// It implements the DataCollection interface
type ExchangeDataCollection struct {
	user string
	// TODO: We would want to replace this with a channel so that we
	// don't need to wait for all data to be retrieved before reading it out
	data []ExchangeData
}

// NextItem returns either the next item in the collection or an error if one occurred.
// If not more items are available in the collection, returns (nil, nil).
func (*ExchangeDataCollection) NextItem() (DataStream, error) {
	// TODO: Return the next "to be read" item in the collection as a
	// DataStream
	return nil, nil
}

// Internal Helper that is invoked when the data collection is created to populate it
func (ed *ExchangeDataCollection) populateCollection() error {
	// TODO: Read data for `ed.user` and add to collection
	return nil
}

// ExchangeData represents a single item retrieved from exchange
type ExchangeData struct {
	id string
	// TODO: We may need this to be a "oneOf" of `message`, `contact`, etc.
	// going forward. Using []byte for now but I assume we'll have
	// some structured type in here (serialization to []byte can be done in `Read`)
	message []byte
}

func (ed *ExchangeData) UUID() string {
	return ed.id
}

func (ed *ExchangeData) Read(bytes []byte) (int, error) {
	// TODO: Copy ed.message into []bytes. Will need to take care of partial reads
	return 0, nil
}