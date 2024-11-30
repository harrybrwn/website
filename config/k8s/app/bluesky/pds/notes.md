# PDS Configuration
See the [env config file](https://github.com/bluesky-social/atproto/blob/main/packages/pds/src/config/env.ts).

# Basics

```sh
PDS_HOSTNAME=
PDS_JWT_SECRET=
PDS_ADMIN_PASSWORD=
PDS_PLC_ROTATION_KEY_K256_PRIVATE_KEY_HEX=
PDS_DATA_DIRECTORY=/pds
```

# Constants
These configuration variables should not change.
```sh
PDS_BLOB_UPLOAD_LIMIT=52428800
PDS_DID_PLC_URL=https://plc.directory
PDS_BSKY_APP_VIEW_URL=https://api.bsky.app
PDS_BSKY_APP_VIEW_DID=did:web:api.bsky.app
PDS_REPORT_SERVICE_URL=https://mod.bsky.app
PDS_REPORT_SERVICE_DID=did:plc:ar7c4by46qjdydhdevvrndac
PDS_CRAWLERS=https://bsky.network
```

# Misc Service Level Config
```sh
LOG_ENABLED=true
# off-config but still from env:
# logging: LOG_LEVEL, LOG_SYSTEMS, LOG_ENABLED, LOG_DESTINATION
```

# Storage

## SQL
Defaults:
```sh
PDS_ACCOUNT_DB_LOCATION=$PDS_DATA_DIRECTORY/account.sqlite
PDS_SEQUENCER_DB_LOCATION=$PDS_DATA_DIRECTORY/sequencer.sqlite
PDS_DID_CACHE_DB_LOCATION=$PDS_DATA_DIRECTORY/did_cache.sqlite
```

## S3
```sh
PDS_BLOBSTORE_S3_BUCKET=
PDS_BLOBSTORE_S3_REGION=
PDS_BLOBSTORE_S3_ENDPOINT=
PDS_BLOBSTORE_S3_FORCE_PATH_STYLE=
PDS_BLOBSTORE_S3_ACCESS_KEY_ID=
PDS_BLOBSTORE_S3_SECRET_ACCESS_KEY=
PDS_BLOBSTORE_S3_UPLOAD_TIMEOUT_MS=
```

## Disk
```sh
PDS_BLOBSTORE_DISK_LOCATION=/pds/blocks
PDS_BLOBSTORE_DISK_TMP_LOCATION
```
