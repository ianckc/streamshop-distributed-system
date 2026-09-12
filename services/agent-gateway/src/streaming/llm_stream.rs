use bytes::BytesMut;
use futures::StreamExt;
use std::collections::HashMap;
use tokio::sync::mpsc;

use crate::AppState;
use crate::streaming::events::AgentEvent;

pub(crate) struct ToolCallDirective {
    pub tool_name: String,
    pub arguments: serde_json::Value,
}

pub(crate) async fn stream_llm_response(
    state: &AppState,
    messages: &[serde_json::Value],
    tx: &mpsc::Sender<AgentEvent>,
) -> Result<(Vec<ToolCallDirective>, Option<String>), Box<dyn std::error::Error + Send + Sync>> {
    let response = state
        .llm_client
        .post(format!("{}/v1/chat/completions", state.llm_base_url))
        .bearer_auth(state.llm_api_key.api_key())
        .timeout(std::time::Duration::from_secs(90))
        .json(&serde_json::json!({
            "model": "qwen3:8b",
            "messages": messages,
            "stream": true
        }))
        .send()
        .await?
        .error_for_status()?;

    let mut byte_stream = response.bytes_stream();
    let mut buf = BytesMut::with_capacity(4096);
    let mut finish_reason: Option<String> = None;

    // Accumulator for streaming tool-call deltas.
    // OpenAI sends tool-call `function.arguments` as incremental string
    // fragments across multiple chunks. We must buffer per tool-call index
    // and parse only after the stream completes or finish_reason is received.
    let mut tool_call_buffers: HashMap<usize, (String, String)> = HashMap::new();

    'outer: while let Some(chunk) = byte_stream.next().await {
        buf.extend_from_slice(&chunk?);

        loop {
            // Find newline within the buffered bytes
            let Some(pos) = buf.iter().position(|&b| b == b'\n') else {
                break; // wait for more data
            };
            let line_bytes = buf.split_to(pos + 1);
            // Safely decode; we split on a single-byte newline so partial
            // multi-byte sequences cannot span the split boundary.
            let line = match std::str::from_utf8(&line_bytes) {
                Ok(s) => s.trim().to_owned(),
                Err(_) => {
                    tracing::warn!("Invalid UTF-8 in LLM stream line; skipping");
                    continue;
                }
            };

            if line == "data: [DONE]" {
                break 'outer;
            }

            if let Some(json_str) = line.strip_prefix("data: ") {
                if let Ok(parsed) = serde_json::from_str::<serde_json::Value>(json_str) {
                    if let Some(delta) = parsed["choices"][0]["delta"]["content"].as_str() {
                        if !delta.is_empty() {
                            if tx
                                .send(AgentEvent::TokenChunk {
                                    content: delta.to_string(),
                                })
                                .await
                                .is_err()
                            {
                                // Receiver dropped (client disconnected);
                                // stop consuming upstream.
                                break 'outer;
                            }
                        }
                    }

                    if let Some(reason) = parsed["choices"][0]["finish_reason"].as_str() {
                        finish_reason = Some(reason.to_string());
                    }

                    // Accumulate tool-call deltas by index
                    if let Some(tc) = parsed["choices"][0]["delta"]["tool_calls"].as_array() {
                        for call in tc {
                            let index = call["index"].as_u64().unwrap_or(0) as usize;
                            let entry = tool_call_buffers
                                .entry(index)
                                .or_insert_with(|| (String::new(), String::new()));
                            if let Some(name) = call["function"]["name"].as_str() {
                                entry.0.push_str(name);
                            }
                            if let Some(args_fragment) = call["function"]["arguments"].as_str() {
                                entry.1.push_str(args_fragment);
                            }
                        }
                    }
                }
            }
        }
    }

    let tool_calls = finalize_tool_calls(&tool_call_buffers, tx).await?;
    Ok((tool_calls, finish_reason))
}

async fn finalize_tool_calls(
    buffers: &HashMap<usize, (String, String)>,
    tx: &mpsc::Sender<AgentEvent>,
) -> Result<Vec<ToolCallDirective>, Box<dyn std::error::Error + Send + Sync>> {
    // Sort by index to guarantee deterministic message ordering.
    let mut indices: Vec<usize> = buffers.keys().copied().collect();
    indices.sort_unstable();

    let mut tool_calls = Vec::new();
    for index in indices {
        let (name, args_str) = &buffers[&index];
        if name.is_empty() {
            continue;
        }
        let arguments = match serde_json::from_str::<serde_json::Value>(args_str) {
            Ok(v) => v,
            Err(e) => {
                tracing::error!(
                    tool_name = %name,
                    raw_args = %args_str,
                    error = %e,
                    "Failed to parse tool-call arguments JSON"
                );
                let _ = tx
                    .send(AgentEvent::Error {
                        message: format!("Malformed arguments for tool '{name}': {e}"),
                    })
                    .await;
                // Skip this tool call; do not push corrupt data
                // into messages.
                continue;
            }
        };
        let _ = tx
            .send(AgentEvent::ToolCall {
                tool_name: name.clone(),
                arguments: arguments.clone(),
            })
            .await;
        tool_calls.push(ToolCallDirective {
            tool_name: name.clone(),
            arguments,
        });
    }
    Ok(tool_calls)
}
