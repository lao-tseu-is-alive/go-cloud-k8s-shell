#!/bin/bash
rm test-report.json
echo -n > test-report.json
go test -coverprofile coverage.out -json . >> test-report.json