FROM migrate/migrate

COPY ./backend/migrations /migrations

COPY run-migrations.sh /run-migrations.sh

RUN chmod +x /migrate.sh

WORKDIR /migrations

ENTRYPOINT ["/migrate.sh"]