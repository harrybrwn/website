import requests
from config import host


def test_homepage():
    for url in [
        f"http://{host}/",
        f"http://{host}/~harry"
    ]:
        res = requests.get(url)
        assert res.ok
        assert res.status_code == 200
        assert res.headers["Content-Type"].startswith("text/html")
        assert "Last-Modified" in res.headers


def test_robots_txt():
    res = requests.get(f"http://{host}/robots.txt")
    assert res.ok
    assert res.status_code == 200
    assert res.headers["Content-Type"].startswith("text/plain")
    assert "Content-Length" in res.headers
    assert "Last-Modified" in res.headers


def test_manifest_json():
    res = requests.get(f"http://{host}/manifest.json")
    assert res.ok
    assert res.headers.get("Content-Type") == "application/json"
    assert "Content-Length" in res.headers


def test_favicon():
    res = requests.get(f"http://{host}/favicon.ico")
    assert res.ok
    assert res.headers.get("Content-Type") == "image/x-icon"
    assert "Content-Length" in res.headers
    assert "Last-Modified" in res.headers


def test_sitmap():
    res = requests.get(f"http://{host}/sitemap.xml")
    assert "Content-Length" in res.headers
    assert res.headers["Content-Type"].startswith("text/xml")
    assert "Content-Encoding" not in res.headers
    assert "Last-Modified" in res.headers
    res = requests.get(f"http://{host}/sitemap.xml.gz")
    assert res.headers["Content-Type"].startswith("text/xml")
    assert res.headers["Content-Encoding"] == "gzip"
    assert "Content-Length" in res.headers
    assert "Last-Modified" in res.headers


def test_static_images():
    res = requests.get(f"http://{host}/static/img/goofy.jpg")
    assert res.ok
    assert "Content-Length" in res.headers
    assert res.headers["Content-Type"] == "image/jpeg"
    res = requests.get(f"http://{host}/static/img/github.svg")
    assert res.ok
    assert "Content-Length" in res.headers
    assert "Last-Modified" in res.headers
    assert res.headers["Content-Type"] == "image/svg+xml"


def test_resume():
    res = requests.get(f"http://{host}/static/files/HarrisonBrown.pdf")
    assert res.ok
    assert "Content-Length" in res.headers
    assert "Last-Modified" in res.headers
    assert res.headers.get("Content-Type") == "application/pdf"


def test_bootstrap_css():
    res = requests.get(f"http://{host}/static/css/bootstrap.min.css")
    assert res.ok
    assert "Content-Length" in res.headers
    assert "Last-Modified" in res.headers
    assert res.headers.get("Content-Type").startswith("text/css")
