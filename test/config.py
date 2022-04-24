import os

_app_host = os.environ.get("APP_HOST", "localhost")
_app_port = os.environ.get("APP_PORT", None)

if _app_port is None:
	host = _app_host
else:
	host = f"{_app_host}:{_app_port}"

ssl = os.environ.get("SSL", "false").lower() in {"true", "yes", "1"}
scheme = "https" if ssl else "http"
