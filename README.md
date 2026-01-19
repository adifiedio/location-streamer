# location streamer

this is a system built to handle high-frequency location data from devices and stream it out to third-party webhooks. it uses a "buffer" architecture so that even if the external servers are slow or down, no data is lost.

## how it works
1. **ingestion service**: this is the entry point. it receives http posts from devices. it's designed to be fast, so it just validates the request and pushes the data to kafka before returning immediately.
2. **kafka**: acts as our safety buffer. if the database or the external webhook is lagging, the location data just piles up here until we're ready to process it.
3. **streamer service**: this guy does the heavy lifting. it reads from kafka, saves the data to postgres (for our own records), and then tries to post it to the tenant's webhook. it has built-in retries with backoff in case the webhook fails.

## getting started
you'll need docker and the `migrate` tool.

```bash
# this builds everything and starts postgres, kafka, and the 3 services.
# it also runs the database migrations automatically.
make docker-up-all
```

## testing the flow

1. **setup a listener**: go to [webhook.site](https://webhook.site) and grab a unique url. this represents the tenant's actual server.
2. **register a tenant**: create a tenant account and tell the system where to send the data.
   ```bash
   curl -X POST http://localhost:8081/api/v1/register \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer dev-token-123" \
     -d '{
       "tenant_name": "my corp",
       "webhook_url": "https://webhook.site/your-id",
       "api_key": "secret-key-1"
     }'
   ```
3. **send location updates**: simulate a device sending coordinates. 
   ```bash
   curl -X POST http://localhost:8082/api/v1/locations \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer dev-token-123" \
     -d '{
       "latitude": 34.05,
       "longitude": -118.24,
       "timestamp": 1737274800
     }'
   ```
4. **check results**: 
   - your webhook.site tab should light up with the data.
   - check the db logs: `docker logs -f streamer_service`.

## internal architecture & security

### multi-tenancy
we use a shared schema where every row has a `tenant_id`. right now, the app logic handles the isolation by filtering queries based on the user's token. for a more "hardened" prod setup, we can enable postgres row level security (rls) to make sure tenants can never see each other's data even if there's a bug in the code.

### auth & api gateway
- **local development**: the `pkg/auth` middleware handles jwt validation. in `dev` mode, it's pretty relaxed so you can test with dummy tokens.
- **production (aws)**: the idea is to use aws api gateway with a cognito authorizer. the gateway handles the heavy lifting of verifying tokens at the edge, and our services just trust the claims passed down to them.

## what's next?
- scale it: move from docker compose to a kubernetes cluster.
- cloud native: swap the local containers for rds, msk (kafka), and eks using terraform.
- speed: stick redis in front of the tenant lookups.
- visibility: add some prometheus metrics and grafana dashboards to see the "flow" in real-time.
- ci/cd: add github actions for continuous integration and deployment.
- logging: add centralized logging using ELK stack or logrus.