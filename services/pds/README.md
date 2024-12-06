# BlueSky Personal Data Server (PDS)

## Environment Variables

| name                | type | env                        |
| port                | int  | PDS_PORT                   |
| hostname            | str  | PDS_HOSTNAME               |
| serviceDid          | str  | PDS_SERVICE_DID            |
| serviceName         | str  | PDS_SERVICE_NAME           |
| version             | str  | PDS_VERSION                |
| homeUrl             | str  | PDS_HOME_URL               |
| logoUrl             | str  | PDS_LOGO_URL               |
| privacyPolicyUrl    | str  | PDS_PRIVACY_POLICY_URL     |
| supportUrl          | str  | PDS_SUPPORT_URL            |
| termsOfServiceUrl   | str  | PDS_TERMS_OF_SERVICE_URL   |
| contactEmailAddress | str  | PDS_CONTACT_EMAIL_ADDRESS  |
| acceptingImports    | bool | PDS_ACCEPTING_REPO_IMPORTS |
| blobUploadLimit     | int  | PDS_BLOB_UPLOAD_LIMIT      |
| devMode             | bool | PDS_DEV_MODE               |
| log enabled         | bool | LOG_ENABLED                |
| log level           | str  | LOG_LEVEL                  |

### branding
| name         | type | env               |
| brandColor   | str  | PDS_PRIMARY_COLOR |
| errorColor   | str  | PDS_ERROR_COLOR   |
| warningColor | str  | PDS_WARNING_COLOR |

### database
| name                     | type | env |
| dataDirectory            | str  | PDS_DATA_DIRECTORY                     |
| disableWalAutoCheckpoint | bool | PDS_SQLITE_DISABLE_WAL_AUTO_CHECKPOINT |
| accountDbLocation        | str  | PDS_ACCOUNT_DB_LOCATION                |
| sequencerDbLocation      | str  | PDS_SEQUENCER_DB_LOCATION              |
| didCacheDbLocation       | str  | PDS_DID_CACHE_DB_LOCATION              |

### actor store
| name                | type | env                        |
| actorStoreDirectory | str  | PDS_ACTOR_STORE_DIRECTORY  |
| actorStoreCacheSize | int  | PDS_ACTOR_STORE_CACHE_SIZE |

### blobstore
Either S3 or disk is required.

| name                       | type | env                                |
| blobstoreS3Bucket          | str  | PDS_BLOBSTORE_S3_BUCKET            |
| blobstoreS3Region          | str  | PDS_BLOBSTORE_S3_REGION            |
| blobstoreS3Endpoint        | str  | PDS_BLOBSTORE_S3_ENDPOINT          |
| blobstoreS3ForcePathStyle  | bool | PDS_BLOBSTORE_S3_FORCE_PATH_STYLE  |
| blobstoreS3AccessKeyId     | str  | PDS_BLOBSTORE_S3_ACCESS_KEY_ID     |
| blobstoreS3SecretAccessKey | str  | PDS_BLOBSTORE_S3_SECRET_ACCESS_KEY |
| blobstoreS3UploadTimeoutMs | int  | PDS_BLOBSTORE_S3_UPLOAD_TIMEOUT_MS |
| blobstoreDiskLocation      | str  | PDS_BLOBSTORE_DISK_LOCATION        |
| blobstoreDiskTmpLocation   | str  | PDS_BLOBSTORE_DISK_TMP_LOCATION    |

### identity
| name                    | type | env                             |
| didPlcUrl               | str  | PDS_DID_PLC_URL                 |
| didCacheStaleTTL        | int  | PDS_DID_CACHE_STALE_TTL         |
| didCacheMaxTTL          | int  | PDS_DID_CACHE_MAX_TTL           |
| resolverTimeout         | int  | PDS_ID_RESOLVER_TIMEOUT         |
| recoveryDidKey          | str  | PDS_RECOVERY_DID_KEY            |
| serviceHandleDomains    | list | PDS_SERVICE_HANDLE_DOMAINS      |
| handleBackupNameservers | list | PDS_HANDLE_BACKUP_NAMESERVERS   |
| enableDidDocWithSession | bool | PDS_ENABLE_DID_DOC_WITH_SESSION |

### entryway
| name                                 | type | env                                             |
| entrywayUrl                          | str  | PDS_ENTRYWAY_URL                                |
| entrywayDid                          | str  | PDS_ENTRYWAY_DID                                |
| entrywayJwtVerifyKeyK256PublicKeyHex | str  | PDS_ENTRYWAY_JWT_VERIFY_KEY_K256_PUBLIC_KEY_HEX |
| entrywayPlcRotationKey               | str  | PDS_ENTRYWAY_PLC_ROTATION_KEY                   |

### invites
| name | type | env |
| inviteRequired | bool | PDS_INVITE_REQUIRED |
| inviteInterval | int  | PDS_INVITE_INTERVAL |
| inviteEpoch    | int  | PDS_INVITE_EPOCH    |

### email
| name | type | env |
| emailSmtpUrl           | str  | PDS_EMAIL_SMTP_URL            |
| emailFromAddress       | str  | PDS_EMAIL_FROM_ADDRESS        |
| moderationEmailSmtpUrl | str  | PDS_MODERATION_EMAIL_SMTP_URL |
| moderationEmailAddress | str  | PDS_MODERATION_EMAIL_ADDRESS  |

### subscription
| name | type | env |
| maxSubscriptionBuffer | int | PDS_MAX_SUBSCRIPTION_BUFFER |
| repoBackfillLimitMs   | int | PDS_REPO_BACKFILL_LIMIT_MS  |

### appview
| name                     | type | env                               |
| bskyAppViewUrl           | str  | PDS_BSKY_APP_VIEW_URL             |
| bskyAppViewDid           | str  | PDS_BSKY_APP_VIEW_DID             |
| bskyAppViewCdnUrlPattern | str  | PDS_BSKY_APP_VIEW_CDN_URL_PATTERN |

### mod service
| name | type | env |
| modServiceUrl | str | PDS_MOD_SERVICE_URL |
| modServiceDid | str | PDS_MOD_SERVICE_DID |

### report service
| name | type | env |
| reportServiceUrl | str | PDS_REPORT_SERVICE_URL |
| reportServiceDid | str | PDS_REPORT_SERVICE_DID |

### rate limits
| name               | type | env                       |
| rateLimitsEnabled  | bool | PDS_RATE_LIMITS_ENABLED   |
| rateLimitBypassKey | str  | PDS_RATE_LIMIT_BYPASS_KEY |
| rateLimitBypassIps | list | PDS_RATE_LIMIT_BYPASS_IPS |

### redis
| name | type | env |
| redisScratchAddress  | str | PDS_REDIS_SCRATCH_ADDRESS  |
| redisScratchPassword | str | PDS_REDIS_SCRATCH_PASSWORD |

### crawlers
| name     | type | env |
| crawlers | list | PDS_CRAWLERS |

### secrets
| name          | type | env                | generate                |
| dpopSecret    | str  | PDS_DPOP_SECRET    |                         |
| jwtSecret     | str  | PDS_JWT_SECRET     | `openssl rand --hex 16` |
| adminPassword | str  | PDS_ADMIN_PASSWORD |                         |

### Encryption
| name                            | type | env                                       | generate |
| plcRotationKeyKmsKeyId          | str  | PDS_PLC_ROTATION_KEY_KMS_KEY_ID           | |
| plcRotationKeyK256PrivateKeyHex | str  | PDS_PLC_ROTATION_KEY_K256_PRIVATE_KEY_HEX | `openssl ecparam --name secp256k1 --genkey --noout --outform DER | tail --bytes=+8 | head --bytes=32 | xxd --plain --cols 32` |

### user provided url http requests
| name                  | type | env                         |
| disableSsrfProtection | bool | PDS_DISABLE_SSRF_PROTECTION |

### fetch
| name                 | type | env                         |
| fetchMaxResponseSize | int  | PDS_FETCH_MAX_RESPONSE_SIZE |

### proxy
| name                  | type | env                         |
| proxyAllowHTTP2       | bool | PDS_PROXY_ALLOW_HTTP2       |
| proxyHeadersTimeout   | int  | PDS_PROXY_HEADERS_TIMEOUT   |
| proxyBodyTimeout      | int  | PDS_PROXY_BODY_TIMEOUT      |
| proxyMaxResponseSize  | int  | PDS_PROXY_MAX_RESPONSE_SIZE |
| proxyMaxRetries       | int  | PDS_PROXY_MAX_RETRIES       |
| proxyPreferCompressed | bool | PDS_PROXY_PREFER_COMPRESSED |
