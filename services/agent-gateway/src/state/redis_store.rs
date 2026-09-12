use crate::AppState;
use redis::AsyncCommands;

pub async fn save_context(
    state: &AppState,
    session_id: &str,
    messages: &[serde_json::Value],
) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
    // Validate session_id to prevent Redis key injection
    if session_id.is_empty() || !session_id.chars().all(|c| c.is_alphanumeric() || c == '-') {
        return Err("invalid session_id: must be alphanumeric or hyphens only".into());
    }
    let mut conn = state.redis_conn.clone();
    let serialized = serde_json::to_string(messages)?;
    conn.set_ex::<_, _, ()>(
        format!("session:{session_id}:context"),
        serialized,
        3600, // 1-hour TTL
    )
    .await?;
    Ok(())
}

pub async fn load_context(
    state: &AppState,
    session_id: &str,
) -> Result<Vec<serde_json::Value>, Box<dyn std::error::Error + Send + Sync>> {
    if session_id.is_empty() || !session_id.chars().all(|c| c.is_alphanumeric() || c == '-') {
        return Err("invalid session_id: must be alphanumeric or hyphens only".into());
    }
    let mut conn = state.redis_conn.clone();
    let result: Option<String> = conn.get(format!("session:{session_id}:context")).await?;
    match result {
        Some(data) => Ok(serde_json::from_str(&data)?),
        None => Ok(Vec::new()),
    }
}
