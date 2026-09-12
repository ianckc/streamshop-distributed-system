/// OpenAI-compatible tool definitions advertised to the LLM (Phase A).
/// Handlers are wired in A4; schemas alone let the model emit tool calls.
pub fn phase_a_tool_schemas() -> Vec<serde_json::Value> {
    vec![
        serde_json::json!({
            "type": "function",
            "function": {
                "name": "get_order",
                "description": "Fetch a StreamShop order by UUID from analytics-api (Postgres). Returns status, totals, and line items.",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "order_id": {
                            "type": "string",
                            "description": "Order UUID (e.g. from a prior create-order response)."
                        }
                    },
                    "required": ["order_id"],
                    "additionalProperties": false
                }
            }
        }),
        serde_json::json!({
            "type": "function",
            "function": {
                "name": "get_orders_summary",
                "description": "Fetch aggregate order metrics from analytics-api (ClickHouse): counts, revenue, etc.",
                "parameters": {
                    "type": "object",
                    "properties": {},
                    "additionalProperties": false
                }
            }
        }),
        serde_json::json!({
            "type": "function",
            "function": {
                "name": "check_service_health",
                "description": "Call /ready (or /health) on a StreamShop service and return status + body.",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "service": {
                            "type": "string",
                            "enum": ["catalog", "order", "analytics", "event-processor"],
                            "description": "Which service to probe."
                        }
                    },
                    "required": ["service"],
                    "additionalProperties": false
                }
            }
        }),
    ]
}
