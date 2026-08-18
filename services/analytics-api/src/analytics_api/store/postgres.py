from dataclasses import dataclass
from datetime import datetime
from uuid import UUID

from psycopg.rows import dict_row
from psycopg_pool import ConnectionPool


@dataclass(frozen=True)
class OrderItem:
    product_id: str
    qty: int
    price_pence: int


@dataclass(frozen=True)
class OrderDetail:
    id: UUID
    user_id: UUID
    status: str
    total_pence: int
    items: list[OrderItem]
    created_at: datetime


class PostgresStore:
    def __init__(self, database_url: str, *, pool: ConnectionPool | None = None) -> None:
        self._pool = pool or ConnectionPool(
            conninfo=database_url,
            min_size=1,
            max_size=5,
            kwargs={"row_factory": dict_row},
        )
        self._owns_pool = pool is None

    def ping(self) -> None:
        with self._pool.connection() as conn:
            conn.execute("SELECT 1")

    def get_order(self, order_id: UUID) -> OrderDetail | None:
        with self._pool.connection() as conn:
            order = conn.execute(
                """
                SELECT id, user_id, status, total_pence, created_at
                FROM orders
                WHERE id = %s
                """,
                (order_id,),
            ).fetchone()
            if order is None:
                return None

            items = conn.execute(
                """
                SELECT product_id, qty, price_pence
                FROM order_items
                WHERE order_id = %s
                ORDER BY id
                """,
                (order_id,),
            ).fetchall()

        return OrderDetail(
            id=order["id"],
            user_id=order["user_id"],
            status=order["status"],
            total_pence=order["total_pence"],
            created_at=order["created_at"],
            items=[
                OrderItem(
                    product_id=row["product_id"],
                    qty=row["qty"],
                    price_pence=row["price_pence"],
                )
                for row in items
            ],
        )

    def close(self) -> None:
        if self._owns_pool:
            self._pool.close()
