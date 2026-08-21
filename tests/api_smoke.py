import os
import concurrent.futures

import pytest
import requests

VISITOR = os.environ.get("VISITOR_URL", "http://server:8080")
STATUS = os.environ.get("STATUS_URL", "http://server:9090")
DEMO = os.environ.get("DEMO_URL", "http://demo:8080")
CLIENT = os.environ.get("CLIENT_HEALTH_URL", "http://client:9091")
NODE_ID = os.environ.get("EXPECTED_NODE_ID", "intranet-alpha-7")


def test_server_health():
    r = requests.get(f"{STATUS}/api/v1/health", timeout=5)
    assert r.status_code == 200
    body = r.json()["data"]
    assert body["role"] == "server"
    assert body["status"] == "ok"
    assert "time" in body


def test_client_connected():
    r = requests.get(f"{CLIENT}/api/v1/health", timeout=5)
    assert r.status_code == 200
    assert r.json()["data"]["connected"] is True


def test_tunnel_identity_matches_direct():
    via_tunnel = requests.get(f"{VISITOR}/api/v1/identity", timeout=5)
    direct = requests.get(f"{DEMO}/api/v1/identity", timeout=5)
    assert via_tunnel.status_code == 200
    assert direct.status_code == 200
    t = via_tunnel.json()["data"]
    d = direct.json()["data"]
    assert t["node_id"] == NODE_ID
    assert d["node_id"] == NODE_ID
    assert t["hostname"] == d["hostname"]


def test_demo_page_through_tunnel():
    r = requests.get(f"{VISITOR}/", timeout=5)
    assert r.status_code == 200
    assert "SimpleFrp" in r.text
    assert 'id="root"' in r.text
    assert ".js" in r.text


def test_concurrent_visitors():
    def hit(_):
        r = requests.get(f"{VISITOR}/api/v1/identity", timeout=10)
        r.raise_for_status()
        return r.json()["data"]["node_id"]

    with concurrent.futures.ThreadPoolExecutor(max_workers=100) as ex:
        ids = list(ex.map(hit, range(100)))
    assert ids.count(NODE_ID) == 100


def test_status_reports_client():
    r = requests.get(f"{STATUS}/api/v1/status", timeout=5)
    assert r.status_code == 200
    data = r.json()["data"]
    assert data["client_connected"] is True
    assert data["client_id"] == "edge-01"


def test_unknown_endpoint_error_shape():
    r = requests.get(f"{STATUS}/api/v1/nope", timeout=5)
    assert r.status_code == 404
    err = r.json()["error"]
    assert err["code"] == "not_found"
    assert "message" in err
