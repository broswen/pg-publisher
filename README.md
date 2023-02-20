# pg-publisher

pg-publisher is a service to watch for changes in a postgres table and publish the rows to kafka.

See [transactional outbox](https://microservices.io/patterns/data/transactional-outbox.html) and [polling publisher](https://microservices.io/patterns/data/polling-publisher.html) for more details.

### Usage
1. the table schema must include a version number with each row that is increased during every change
1. set the `BROKERS` environment variable to your kafka brokers
1. set the `PUBLISHER_TOPIC` environment variable for your kafka topic
1. set the `DSN` environment variable to your postgres DSN
1. set the `TABLE_NAME` environment variable for the name of the table the publisher will watch
1. set the `VERSION_COLUMN` environment variable for the name of the version column in the table schema
1. set the `ID`  environment variable for the id of the publisher
1. set the `LOCK_NAME` and `NAMESPACE` environment variables if using HA mode with kubernetes leader election
1. start!

### What it does
pg-publisher will get the last published version for `ID` in the `pg_publisher` table. It will then find the maximum version
in `TABLE_NAME` and submit all rows greater than the last published version into `PUBLISHER_TOPIC`.
Then it will store the last published version and repeat.

If `LOCK_NAME` and `NAMESPACE` are defined, pg-publisher will attempt to run in HA mode with kubernetes leader election.
You can run multiple replicas and only one instance will be publishing. If the leading instance fails, one of the standy instances
will become the leader and start publishing.

### Example
See [001-initial-schema](sql/migrations/001-initial-schema.sql) for an example table schema and [docker-compose.yaml](docker-compose.yaml) for a configuration setup.

### Issues
- because all rows are serialized as `map[string]interface{}`, and pg-publisher doesn't know the table schema, things are not always serialized the best way.

### TODO
- [ ] pass in table schema as config
- [ ] add tests
