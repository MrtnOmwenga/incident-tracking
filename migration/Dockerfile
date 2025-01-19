FROM migrate/migrate

COPY ../backend/migrations /migrations

COPY migrate.sh /migrate.sh

RUN chmod +x /migrate.sh

WORKDIR /migrations

ENTRYPOINT /migrate.sh