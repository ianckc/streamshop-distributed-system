use axum::{
    Json,
    extract::State,
    response::sse::{Event, KeepAlive, Sse},
};
use futures::stream::Stream;
use std::convert::Infallible;
use tokio::sync::mpsc;
use tokio_stream::StreamExt;
use tokio_stream::wrappers::ReceiverStream;

use crate::AppState;
use crate::agent::turn_orchestrator::orchestrate_turn;
use crate::streaming::events::{AgentEvent, AgentRequest};

pub async fn handle_stream(
    State(state): State<AppState>,
    Json(request): Json<AgentRequest>,
) -> Result<Sse<impl Stream<Item = Result<Event, Infallible>>>, (axum::http::StatusCode, String)> {
    // Validate session_id before spawning any work.
    if request.session_id.is_empty()
        || !request
            .session_id
            .chars()
            .all(|c| c.is_alphanumeric() || c == '-')
    {
        return Err((
            axum::http::StatusCode::BAD_REQUEST,
            "invalid session_id: must be alphanumeric or hyphens only".into(),
        ));
    }

    let (tx, rx) = mpsc::channel::<AgentEvent>(64);

    tokio::spawn(async move {
        if let Err(e) = orchestrate_turn(state, request, tx.clone()).await {
            let _ = tx
                .send(AgentEvent::Error {
                    message: e.to_string(),
                })
                .await;
        }
    });

    let stream = ReceiverStream::new(rx).map(|event| {
        let data = serde_json::to_string(&event).unwrap_or_else(|e| {
            tracing::error!("Event serialization failed: {e}");
            "{\"type\":\"error\",\"payload\":{\"message\":\"serialization failed\"}}".into()
        });
        let event_type = event.event_type();
        Ok(Event::default().event(event_type).data(data))
    });

    Ok(Sse::new(stream).keep_alive(
        KeepAlive::new()
            .interval(std::time::Duration::from_secs(15))
            .text("ping"),
    ))
}
