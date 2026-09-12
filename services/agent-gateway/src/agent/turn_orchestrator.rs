use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

use crate::AppState;
use crate::agent::prompt::{ensure_operator_system_prompt, strip_system_messages};
use crate::state::redis_store;
use crate::streaming::events::{AgentEvent, AgentRequest};

pub async fn orchestrate_turn(
    state: AppState,
    request: AgentRequest,
    tx: mpsc::Sender<AgentEvent>,
) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
    let mut turn_number: u32 = 0;
    let max_turns: u32 = 10;

    let mut messages = redis_store::load_context(&state, &request.session_id).await?;
    messages.push(serde_json::json!({
        "role": "user",
        "content": request.message
    }));
    ensure_operator_system_prompt(&mut messages);

    loop {
        turn_number += 1;
        if turn_number > max_turns {
            tx.send(AgentEvent::Error {
                message: "Max turns exceeded".into(),
            })
            .await?;
            break;
        }

        let turn_id = Uuid::new_v4().to_string();
        tx.send(AgentEvent::TurnStart {
            turn_id: turn_id.clone(),
            turn_number,
        })
        .await?;

        let (tool_calls, finish_reason) =
            crate::streaming::llm_stream::stream_llm_response(&state, &messages, &tx).await?;

        if tool_calls.is_empty() {
            tx.send(AgentEvent::TurnEnd {
                turn_id,
                finish_reason: finish_reason.unwrap_or_else(|| "stop".into()),
            })
            .await?;
            break;
        }

        // Dispatch tool calls concurrently with cancellation support
        let mut tool_handles: Vec<(
            String,
            CancellationToken,
            tokio::task::JoinHandle<serde_json::Value>,
        )> = Vec::new();

        for tc in &tool_calls {
            let tool_name = tc.tool_name.clone();
            let arguments = tc.arguments.clone();
            let tx_clone = tx.clone();
            let state_clone = state.clone();
            let token = CancellationToken::new();
            let token_child = token.clone();

            let handle = tokio::spawn(async move {
                tokio::select! {
                    result = crate::agent::tools::execute_tool(&state_clone, &tool_name, &arguments) => {
                        let _ = tx_clone.send(AgentEvent::ToolResult {
                            tool_name: tool_name.clone(),
                            result: result.clone(),
                        }).await;
                        result
                    }
                    _ = token_child.cancelled() => {
                        tracing::warn!(tool = %tool_name, "Tool task cancelled");
                        serde_json::json!({ "status": "cancelled" })
                    }
                }
            });
            tool_handles.push((tc.tool_name.clone(), token, handle));
        }

        for (name, cancel_token, handle) in tool_handles {
            match tokio::time::timeout(std::time::Duration::from_secs(30), handle).await {
                Ok(Ok(result)) => {
                    let content = match &result {
                        serde_json::Value::String(s) => s.clone(),
                        other => other.to_string(),
                    };
                    messages.push(serde_json::json!({
                        "role": "tool",
                        "name": name,
                        "content": content
                    }));
                }
                Ok(Err(join_err)) => {
                    tracing::error!("Tool task panicked: {join_err}");
                    tx.send(AgentEvent::Error {
                        message: "Tool call failed (task panic)".into(),
                    })
                    .await?;
                }
                Err(_elapsed) => {
                    cancel_token.cancel(); // Actually stop the running task.
                    tx.send(AgentEvent::Error {
                        message: format!("Tool call '{name}' timed out"),
                    })
                    .await?;
                }
            }
        }

        tx.send(AgentEvent::TurnEnd {
            turn_id,
            finish_reason: "tool_calls".into(),
        })
        .await?;
    }

    let to_persist = strip_system_messages(&messages);
    redis_store::save_context(&state, &request.session_id, &to_persist).await?;
    Ok(())
}
