package ldapreceiver

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/go-ldap/ldap/v3"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.uber.org/zap"
)

type ldapReceiver struct {
	host         component.Host
	cancel       context.CancelFunc
	logger       *zap.Logger
	nextConsumer consumer.Metrics
	config       *Config
}

// Insantiate the client connection to LDAP
func ldapClient(ldapRcvr *ldapReceiver) *ldap.Conn {
	// TODO: replace with basic auth through OTel configuration
	user := ldapRcvr.config.User
	pw := ldapRcvr.config.Pw

	//#nosec G402 (CWE-295)  ignore InsecureSkipVerify TLS setting due to self-signed certificates and network isolation
	endpoint := fmt.Sprint("ldaps://", ldapRcvr.config.Endpoint, ":636")
	connection, err := ldap.DialURL(endpoint, ldap.DialWithTLSConfig(&tls.Config{InsecureSkipVerify: ldapRcvr.config.InsecureSkipVerify}))
	if err != nil {
		ldapRcvr.logger.Sugar().Fatalf("Error dialing ldap server %v", err)
	}
	ldapRcvr.logger.Sugar().Debugf("ldaps client succesfully dialed")

	err = connection.Bind(user, pw)
	if err != nil {
		ldapRcvr.logger.Sugar().Fatalf("Error binding user: %v", err)
	}
	ldapRcvr.logger.Sugar().Debugf("ldaps client succesfully bound")

	return connection
}

// Get the results from an ldapsearch by making a connection to LDAP and returning the search
func getResults(conn *ldap.Conn, ldapRcvr *ldapReceiver) (search *ldap.SearchResult) {
	ldapRcvr.logger.Sugar().Debugf("search filter is: %v", ldapRcvr.config.SearchFilter)

	search = performSearch(conn, fmt.Sprint(ldapRcvr.config.SearchFilter), ldapRcvr)

	ldapRcvr.logger.Sugar().Debugf("Number of returned Entries: %d", len(search.Entries))

	return
}

// Perform a search query in LDAP and return the result
func performSearch(conn *ldap.Conn, query string, ldapRcvr *ldapReceiver) (result *ldap.SearchResult) {
	sr := ldap.NewSearchRequest(
		ldapRcvr.config.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0,
		false,
		query,
		[]string{"member"},
		nil,
	)

	result, err := conn.Search(sr)
	if err != nil {
		log.Fatalf("error with searching request: %v", err)
	}

	return result
}

func (ldapRcvr *ldapReceiver) Start(ctx context.Context, host component.Host) error {
	ldapRcvr.host = host
	ctx = context.Background()
	ctx, ldapRcvr.cancel = context.WithCancel(ctx)

	interval, _ := time.ParseDuration(ldapRcvr.config.Interval)

	go func() {
		ldapConn := ldapClient(ldapRcvr)
		defer ldapConn.Close()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ldapRcvr.logger.Info("Processing metrics..")
				getResults(ldapConn, ldapRcvr)
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (ldapRcvr *ldapReceiver) Shutdown(ctx context.Context) error {
	ldapRcvr.cancel()
	return nil
}
