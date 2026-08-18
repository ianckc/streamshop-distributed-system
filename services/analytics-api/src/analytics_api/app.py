from contextlib import asynccontextmanager
from uuid import UUID

from fastapi import FastAPI, HTTPException, Request
from pydantic import BaseModel

from analytics_api.config import Config
from analytics_api.store.clickhouse import ClickHouseStore, OrderSummary
from analytics_api.store.postgres import OrderDetail, PostgresStore

try:
    from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor

    _HAS_OTEL = True
except ImportError:
    _HAS_OTEL = False


class HealthResponse(BaseModel):
    status: str
    service: str


class ErrorResponse(BaseModel):
    error: str


class OrderItemResponse(BaseModel):
    product_id: str
    qty: int
    price_pence: int


class OrderDetailResponse(BaseModel):
    id: str
    user_id: str
    status: str
    total_pence: int
    items: list[OrderItemResponse]
    created_at: str


class OrderSummaryResponse(BaseModel):
    order_count: int
    total_revenue_pence: int
    avg_order_value_pence: float
    total_items: int


def create_app(
    config: Config | None = None,
    *,
    postgres: PostgresStore | None = None,
    clickhouse: ClickHouseStore | None = None,
) -> FastAPI:
    cfg = config or Config.from_env()
    use_lifespan = postgres is None and clickhouse is None

    @asynccontextmanager
    async def lifespan(app: FastAPI):
        pg = PostgresStore(cfg.database_url)
        ch = ClickHouseStore(
            f"http://{cfg.clickhouse_host}:{cfg.clickhouse_port}",
            cfg.clickhouse_user,
            cfg.clickhouse_password,
            cfg.clickhouse_database,
        )
        app.state.config = cfg
        app.state.postgres = pg
        app.state.clickhouse = ch
        yield
        pg.close()
        ch.close()

    app = (
        FastAPI(title="analytics-api", lifespan=lifespan)
        if use_lifespan
        else FastAPI(title="analytics-api")
    )

    if not use_lifespan:
        app.state.config = cfg
        app.state.postgres = postgres
        app.state.clickhouse = clickhouse

    if _HAS_OTEL:
        FastAPIInstrumentor.instrument_app(app)

    @app.get("/health", response_model=HealthResponse)
    def health(request: Request) -> HealthResponse:
        service_name: str = request.app.state.config.service_name
        return HealthResponse(status="ok", service=service_name)

    @app.get("/ready", response_model=HealthResponse, responses={503: {"model": ErrorResponse}})
    def ready(request: Request) -> HealthResponse:
        service_name: str = request.app.state.config.service_name
        try:
            request.app.state.clickhouse.ping()
            request.app.state.postgres.ping()
        except Exception:
            raise HTTPException(status_code=503, detail={"error": "not ready"}) from None
        return HealthResponse(status="ok", service=service_name)

    @app.get(
        "/api/analytics/orders/summary",
        response_model=OrderSummaryResponse,
        responses={503: {"model": ErrorResponse}},
    )
    def orders_summary(request: Request) -> OrderSummaryResponse:
        try:
            summary: OrderSummary = request.app.state.clickhouse.order_summary()
        except Exception:
            raise HTTPException(
                status_code=503,
                detail={"error": "failed to fetch order summary"},
            ) from None
        return _to_summary_response(summary)

    @app.get(
        "/api/analytics/orders/{order_id}",
        response_model=OrderDetailResponse,
        responses={404: {"model": ErrorResponse}, 503: {"model": ErrorResponse}},
    )
    def order_detail(request: Request, order_id: UUID) -> OrderDetailResponse:
        try:
            order: OrderDetail | None = request.app.state.postgres.get_order(order_id)
        except Exception:
            raise HTTPException(
                status_code=503,
                detail={"error": "failed to fetch order"},
            ) from None
        if order is None:
            raise HTTPException(status_code=404, detail={"error": "order not found"})
        return _to_detail_response(order)

    return app


def _to_summary_response(summary: OrderSummary) -> OrderSummaryResponse:
    return OrderSummaryResponse(
        order_count=summary.order_count,
        total_revenue_pence=summary.total_revenue_pence,
        avg_order_value_pence=summary.avg_order_value_pence,
        total_items=summary.total_items,
    )


def _to_detail_response(order: OrderDetail) -> OrderDetailResponse:
    return OrderDetailResponse(
        id=str(order.id),
        user_id=str(order.user_id),
        status=order.status,
        total_pence=order.total_pence,
        items=[
            OrderItemResponse(
                product_id=item.product_id,
                qty=item.qty,
                price_pence=item.price_pence,
            )
            for item in order.items
        ],
        created_at=order.created_at.isoformat(),
    )
