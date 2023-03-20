package ldapreceiver

import (
	"context"
	"crypto/tls"
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
	//user := strings.Join([]string{"LUV", creds.Username}, "\\")
	//pw := creds.Password
	// TODO: replace with what is above coming from OTEL config
	user := "cn=admin,dc=example,dc=org"
	pw := "admin"

	//#nosec G402 (CWE-295)  ignore InsecureSkipVerify TLS setting due to self-signed certificates and network isolation
	// TODO: LDAP config should come from OTEL config.yaml
	//connection, err := ldap.DialURL("ldaps://<configured>:636", ldap.DialWithTLSConfig(&tls.Config{InsecureSkipVerify: SkipTlsVerification}))
	connection, err := ldap.DialURL("ldaps://localhost:636", ldap.DialWithTLSConfig(&tls.Config{InsecureSkipVerify: true}))
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
