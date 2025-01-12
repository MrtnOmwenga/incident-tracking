FROM golang-migrate/migrate:v4.18.1
COPY ./backend/migrations /migrations
WORKDIR /migrations
ENTRYPOINT ["migrate", "-verbose", "-source", "file:///migrations", "-database"]
CMD ["postgres://${DB_USER}:${DB_PASSWORD}@postgres:5432/${DB_NAME}?sslmode=disable", "up"]