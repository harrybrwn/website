import pytest
import requests


@pytest.mark.parametrize("ip", [
	"198.12.65.246",
	"134.195.121.38",
	"135.181.162.99",
	"99.83.231.61",
	"95.216.235.9",
	"20.244.22.67",
	"45.32.111.71",
])
def test_ip(ip: str):
	res = requests.get(f"https://ip.hrry.dev-local/{ip}")
	assert res.ok
