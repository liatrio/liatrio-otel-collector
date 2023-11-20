include ../../Makefile.Common

.PHONY: compgen
compgen:
	go install .
	rm -fr ../../pkg/receiver/MyAwesomeReceiver/
	compgen receiver github.com/liatrio/liatrio-otel-collector/pkg/receiver/MyAwesomeReceiver
