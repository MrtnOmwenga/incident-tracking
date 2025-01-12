#!/bin/sh
migrate -verbose -source file:///migrations -database "postgres://$DB_USER:$DB_PASSWORD@postgres:5432/$DB_NAME?sslmode=disable" up