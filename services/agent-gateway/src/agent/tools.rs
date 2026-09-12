use crate::AppState;

/// OpenAI-compatible tool definitions advertised to the LLM (Phase A).
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

const TOOL_HTTP_TIMEOUT: std::time::Duration = std::time::Duration::from_secs(5);

/// Dispatch a Phase A tool call to StreamShop HTTP APIs.
pub async fn execute_tool(
    state: &AppState,
    name: &str,
    args: &serde_json::Value,
) -> serde_json::Value {
    match name {
        "get_order" => get_order(state, args).await,
        "get_orders_summary" => get_orders_summary(state).await,
        "check_service_health" => check_service_health(state, args).await,
        other => serde_json::json!({
            "ok": false,
            "error": format!("unknown tool: {other}")
        }),
    }
}

async fn get_order(state: &AppState, args: &serde_json::Value) -> serde_json::Value {
    let Some(order_id) = args.get("order_id").and_then(|v| v.as_str()).filter(|s| !s.is_empty())
    else {
        return serde_json::json!({
            "ok": false,
            "error": "missing required argument: order_id"
        });
    };

    let url = format!(
        "{}/api/analytics/orders/{}",
        state.streamshop.analytics_url.trim_end_matches('/'),
        order_id
    );
    http_get(state, &url).await
}

async fn get_orders_summary(state: &AppState) -> serde_json::Value {
    let url = format!(
        "{}/api/analytics/orders/summary",
        state.streamshop.analytics_url.trim_end_matches('/')
    );
    http_get(state, &url).await
}

async fn check_service_health(state: &AppState, args: &serde_json::Value) -> serde_json::Value {
    let Some(service) = args.get("service").and_then(|v| v.as_str()) else {
        return serde_json::json!({
            "ok": false,
            "error": "missing required argument: service"
        });
    };

    let base = match service {
        "catalog" => state.streamshop.catalog_url.as_str(),
        "order" => state.streamshop.order_url.as_str(),
        "analytics" => state.streamshop.analytics_url.as_str(),
        "event-processor" => state.streamshop.event_processor_url.as_str(),
        other => {
            return serde_json::json!({
                "ok": false,
                "error": format!("unknown service: {other}; expected catalog|order|analytics|event-processor")
            });
        }
    };

    let url = format!("{}/ready", base.trim_end_matches('/'));
    let mut result = http_get(state, &url).await;
    if let Some(obj) = result.as_object_mut() {
        obj.insert("service".into(), serde_json::json!(service));
        obj.insert("probe".into(), serde_json::json!("/ready"));
    }
    result
}

async fn http_get(state: &AppState, url: &str) -> serde_json::Value {
    tracing::debug!(%url, "tool HTTP GET");
    match state
        .http_client
        .get(url)
        .timeout(TOOL_HTTP_TIMEOUT)
        .send()
        .await
    {
        Ok(resp) => {
            let status = resp.status().as_u16();
            let ok = resp.status().is_success();
            let text = resp.text().await.unwrap_or_default();
            let body = serde_json::from_str::<serde_json::Value>(&text)
                .unwrap_or_else(|_| serde_json::json!(text));
            serde_json::json!({
                "ok": ok,
                "status": status,
                "body": body
            })
        }
        Err(err) => serde_json::json!({
            "ok": false,
            "error": format!("request failed: {err}"),
            "url": url
        }),
    }
}
