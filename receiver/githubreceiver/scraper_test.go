package githubreceiver // import "github.com/liatrio/otel-liatrio-contrib/receiver/githubreceiver"
//import (
//	"testing"
//
//	"github.com/go-ldap/ldap/v3"
//	"github.com/stretchr/testify/mock"
//
//	//"go.opentelemetry.io/collector/component/componenttest"
//	//"go.opentelemetry.io/collector/consumer/consumertest"
//	"go.uber.org/zap"
//)
//
//type MockLDAPConn struct {
//	mock.Mock
//}
//
//func (m *MockLDAPConn) Search(searchRequest *ldap.SearchRequest) (*ldap.SearchResult, error) {
//	args := m.Called(searchRequest)
//	return args.Get(0).(*ldap.SearchResult), args.Error(1)
//}
//
//func TestPerformSearch(t *testing.T) {
//	// Prepare a mocked LDAP connection
//	conn := new(MockLDAPConn)
//	entries := []*ldap.Entry{
//		{DN: "cn=test1,ou=users,dc=example,dc=com"},
//		{DN: "cn=test2,ou=users,dc=example,dc=com"},
//	}
//
//	searchResult := &ldap.SearchResult{
//		Entries: entries,
//	}
//
//	conn.On("Search", mock.Anything).Return(searchResult, nil)
//
//	// Prepare the ldapReceiver
//	rcvr := &ldapReceiver{
//		logger: zap.NewNop(),
//		config: &Config{
//			Interval:     "30s",
//			SearchFilter: "(cn=test*)",
//			Endpoint:     "localhost",
//			BaseDN:       "dc=example,dc=com",
//			User:         "cn=admin,dc=example,dc=com",
//			Pw:           "password",
//		},
//	}
//
//	// Test performSearch
//	result := performSearch(conn, rcvr.config.SearchFilter, rcvr)
//
//	// Check if the search result is as expected
//	if len(result.Entries) != len(entries) {
//		t.Errorf("Expected %d entries, got %d", len(entries), len(result.Entries))
//	}
//
//	for i, entry := range result.Entries {
//		if entry.DN != entries[i].DN {
//			t.Errorf("Expected DN: %s, got: %s", entries[i].DN, entry.DN)
//		}
//	}
//
//	conn.AssertExpectations(t)
//}
