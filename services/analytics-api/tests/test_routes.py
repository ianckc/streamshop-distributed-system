from datetime import UTC, datetime
from unittest.mock import MagicMock
from uuid import UUID

import pytest
from fastapi.testclient import TestClient

from analytics_api.app import create_app
from analytics_api.config import Config
from analytics_api.store.clickhouse import OrderSummary
from analytics_api.store.postgres import OrderDetail, OrderItem


@pytest.fixture
def mock_postgres() -> MagicMock:
    return MagicMock()


@pytest.fixture
def mock_clickhouse() -> MagicMock:
    return MagicMock()


@pytest.fixture
def client(mock_postgres: MagicMock, mock_clickhouse: MagicMock) -> TestClient:
    config = Config(
        port=3003,
        service_name="analytics-api-test",
        database_url="postgres://test:test@localhost:5432/test",
        clickhouse_host="localhost",
        clickhouse_port=8123,
        clickhouse_user="test",
        clickhouse_password="test",
        clickhouse_database="streamshop",
    )
    app = create_app(config, postgres=mock_postgres, clickhouse=mock_clickhouse)
    return TestClient(app)


def test_health(client: TestClient) -> None:
    response = client.get("/health")
    assert response.status_code == 200
    assert response.json() == {"status": "ok", "service": "analytics-api-test"}


def test_ready_ok(client: TestClient, mock_postgres: MagicMock, mock_clickhouse: MagicMock) -> None:
    response = client.get("/ready")
    assert response.status_code == 200
    mock_postgres.ping.assert_called_once()
    mock_clickhouse.ping.assert_called_once()


def test_ready_503_when_clickhouse_down(
    client: TestClient, mock_clickhouse: MagicMock
) -> None:
    mock_clickhouse.ping.side_effect = RuntimeError("down")
    response = client.get("/ready")
    assert response.status_code == 503
    assert response.json() == {"detail": {"error": "not ready"}}


def test_orders_summary(client: TestClient, mock_clickhouse: MagicMock) -> None:
    mock_clickhouse.order_summary.return_value = OrderSummary(
        order_count=3,
        total_revenue_pence=12000,
        avg_order_value_pence=4000.0,
        total_items=7,
    )
    response = client.get("/api/analytics/orders/summary")
    assert response.status_code == 200
    assert response.json() == {
        "order_count": 3,
        "total_revenue_pence": 12000,
        "avg_order_value_pence": 4000.0,
        "total_items": 7,
    }


def test_order_detail(client: TestClient, mock_postgres: MagicMock) -> None:
    order_id = UUID("550e8400-e29b-41d4-a716-446655440000")
    mock_postgres.get_order.return_value = OrderDetail(
        id=order_id,
        user_id=UUID("660e8400-e29b-41d4-a716-446655440001"),
        status="processed",
        total_pence=3998,
        created_at=datetime(2026, 1, 15, 12, 0, 0, tzinfo=UTC),
        items=[OrderItem(product_id="prod-001", qty=2, price_pence=1999)],
    )
    response = client.get(f"/api/analytics/orders/{order_id}")
    assert response.status_code == 200
    assert response.json() == {
        "id": str(order_id),
        "user_id": "660e8400-e29b-41d4-a716-446655440001",
        "status": "processed",
        "total_pence": 3998,
        "items": [{"product_id": "prod-001", "qty": 2, "price_pence": 1999}],
        "created_at": "2026-01-15T12:00:00+00:00",
    }


def test_order_detail_not_found(client: TestClient, mock_postgres: MagicMock) -> None:
    order_id = UUID("550e8400-e29b-41d4-a716-446655440000")
    mock_postgres.get_order.return_value = None
    response = client.get(f"/api/analytics/orders/{order_id}")
    assert response.status_code == 404
    assert response.json() == {"detail": {"error": "order not found"}}


def test_order_detail_invalid_uuid(client: TestClient) -> None:
    response = client.get("/api/analytics/orders/not-a-uuid")
    assert response.status_code == 422
