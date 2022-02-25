import os

api_host = os.environ.get("API_HOST", "localhost")
api_port = os.environ.get("API_PORT", "8080")
host = f"{api_host}:{api_port}"
