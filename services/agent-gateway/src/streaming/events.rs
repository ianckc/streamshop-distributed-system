use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AgentRequest {
    pub session_id: String,
    pub message: String,
    #[serde(default)]
    pub model: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type", content = "payload")]
pub enum AgentEvent {
    TurnStart {
        turn_id: String,
        turn_number: u32,
    },
    TokenChunk {
        content: String,
    },
    ToolCall {
        tool_name: String,
        arguments: serde_json::Value,
    },
    ToolResult {
        tool_name: String,
        result: serde_json::Value,
    },
    TurnEnd {
        turn_id: String,
        finish_reason: String,
    },
    Error {
        message: String,
    },
}

impl AgentEvent {
    pub fn event_type(&self) -> &'static str {
        match self {
            AgentEvent::TurnStart { .. } => "turn_start",
            AgentEvent::TokenChunk { .. } => "token_chunk",
            AgentEvent::ToolCall { .. } => "tool_call",
            AgentEvent::ToolResult { .. } => "tool_result",
            AgentEvent::TurnEnd { .. } => "turn_end",
            AgentEvent::Error { .. } => "error",
        }
    }
}
