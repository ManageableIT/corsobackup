package selectors_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/alcionai/corso/src/internal/common"
	"github.com/alcionai/corso/src/pkg/backup/details"
	"github.com/alcionai/corso/src/pkg/selectors"
	"github.com/alcionai/corso/src/pkg/selectors/testdata"
)

type SelectorReduceSuite struct {
	suite.Suite
}

func TestSelectorReduceSuite(t *testing.T) {
	suite.Run(t, new(SelectorReduceSuite))
}

func (suite *SelectorReduceSuite) TestReduce() {
	ctx := context.Background()
	allDetails := testdata.GetDetailsSet()
	table := []struct {
		name     string
		selFunc  func() selectors.Reducer
		expected []details.DetailsEntry
	}{
		{
			name: "ExchangeAllMail",
			selFunc: func() selectors.Reducer {
				sel := selectors.NewExchangeRestore()
				sel.Include(sel.Mails(
					selectors.Any(),
					selectors.Any(),
					selectors.Any(),
				))

				return sel
			},
			expected: testdata.ExchangeEmailItems,
		},
		{
			name: "ExchangeMailSubject",
			selFunc: func() selectors.Reducer {
				sel := selectors.NewExchangeRestore()
				sel.Filter(sel.MailSubject("foo"))

				return sel
			},
			expected: []details.DetailsEntry{testdata.ExchangeEmailItems[0]},
		},
		{
			name: "ExchangeMailSubjectExcludeItem",
			selFunc: func() selectors.Reducer {
				sel := selectors.NewExchangeRestore()
				sel.Filter(sel.MailSender("a-person"))
				sel.Exclude(sel.Mails(
					selectors.Any(),
					selectors.Any(),
					[]string{testdata.ExchangeEmailItemPath2.ShortRef()},
				))

				return sel
			},
			expected: []details.DetailsEntry{testdata.ExchangeEmailItems[0]},
		},
		{
			name: "ExchangeMailSender",
			selFunc: func() selectors.Reducer {
				sel := selectors.NewExchangeRestore()
				sel.Filter(sel.MailSender("a-person"))

				return sel
			},
			expected: testdata.ExchangeEmailItems,
		},
		{
			name: "ExchangeMailReceivedTime",
			selFunc: func() selectors.Reducer {
				sel := selectors.NewExchangeRestore()
				sel.Filter(sel.MailReceivedBefore(
					common.FormatTime(testdata.Time1.Add(time.Second)),
				))

				return sel
			},
			expected: []details.DetailsEntry{testdata.ExchangeEmailItems[0]},
		},
		{
			name: "ExchangeMailID",
			selFunc: func() selectors.Reducer {
				sel := selectors.NewExchangeRestore()
				sel.Include(sel.Mails(
					selectors.Any(),
					selectors.Any(),
					[]string{testdata.ExchangeEmailItemPath1.Item()},
				))

				return sel
			},
			expected: []details.DetailsEntry{testdata.ExchangeEmailItems[0]},
		},
		{
			name: "ExchangeMailShortRef",
			selFunc: func() selectors.Reducer {
				sel := selectors.NewExchangeRestore()
				sel.Include(sel.Mails(
					selectors.Any(),
					selectors.Any(),
					[]string{testdata.ExchangeEmailItemPath1.ShortRef()},
				))

				return sel
			},
			expected: []details.DetailsEntry{testdata.ExchangeEmailItems[0]},
		},
		{
			name: "ExchangeAllEventsAndMailWithSubject",
			selFunc: func() selectors.Reducer {
				sel := selectors.NewExchangeRestore()
				sel.Include(sel.Events(
					selectors.Any(),
					selectors.Any(),
					selectors.Any(),
				))
				sel.Filter(sel.MailSubject("foo"))

				return sel
			},
			expected: []details.DetailsEntry{testdata.ExchangeEmailItems[0]},
		},
		{
			name: "ExchangeEventsAndMailWithSubject",
			selFunc: func() selectors.Reducer {
				sel := selectors.NewExchangeRestore()
				sel.Filter(sel.EventSubject("foo"))
				sel.Filter(sel.MailSubject("foo"))

				return sel
			},
			expected: []details.DetailsEntry{},
		},
		{
			name: "ExchangeAll",
			selFunc: func() selectors.Reducer {
				sel := selectors.NewExchangeRestore()
				sel.Include(sel.Users(
					selectors.Any(),
				))

				return sel
			},
			expected: append(
				append(
					append(
						[]details.DetailsEntry{},
						testdata.ExchangeEmailItems...),
					testdata.ExchangeContactsItems...),
				testdata.ExchangeEventsItems...,
			),
		},
	}

	for _, test := range table {
		suite.T().Run(test.name, func(t *testing.T) {
			test := test

			t.Parallel()

			output := test.selFunc().Reduce(ctx, allDetails)
			assert.ElementsMatch(t, test.expected, output.Entries)
		})
	}
}