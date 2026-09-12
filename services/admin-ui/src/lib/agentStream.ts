export type AgentRequest = {
  session_id: string
  message: string
  model?: string
}

export type AgentEvent =
  | { type: 'TurnStart'; payload: { turn_id: string; turn_number: number } }
  | { type: 'TokenChunk'; payload: { content: string } }
  | {
      type: 'ToolCall'
      payload: { tool_name: string; arguments: unknown }
    }
  | {
      type: 'ToolResult'
      payload: { tool_name: string; result: unknown }
    }
  | { type: 'TurnEnd'; payload: { turn_id: string; finish_reason: string } }
  | { type: 'Error'; payload: { message: string } }

export type StreamAgentOptions = {
  request: AgentRequest
  signal?: AbortSignal
  onEvent: (event: AgentEvent) => void
}

function parseAgentEvent(data: string): AgentEvent | null {
  if (!data || data === 'ping') return null
  try {
    const parsed = JSON.parse(data) as AgentEvent
    if (!parsed || typeof parsed !== 'object' || !('type' in parsed)) {
      return null
    }
    return parsed
  } catch {
    return null
  }
}

/**
 * POST to the gateway SSE endpoint and invoke onEvent for each parsed frame.
 * Uses fetch + ReadableStream because EventSource only supports GET.
 */
export async function streamAgent(options: StreamAgentOptions): Promise<void> {
  const { request, signal, onEvent } = options

  const response = await fetch('/v1/agent/stream', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Accept: 'text/event-stream' },
    body: JSON.stringify(request),
    signal,
  })

  if (!response.ok) {
    const detail = await response.text().catch(() => '')
    throw new Error(
      detail.trim() || `Gateway error ${response.status} ${response.statusText}`,
    )
  }

  if (!response.body) {
    throw new Error('No response body from gateway')
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let eventName = ''
  let dataLines: string[] = []

  const flushFrame = () => {
    if (dataLines.length === 0) {
      eventName = ''
      return
    }
    const data = dataLines.join('\n')
    dataLines = []
    eventName = ''

    const event = parseAgentEvent(data)
    if (event) onEvent(event)
  }

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split(/\r?\n/)
    buffer = lines.pop() ?? ''

    for (const line of lines) {
      if (line === '') {
        flushFrame()
        continue
      }
      if (line.startsWith(':')) continue
      if (line.startsWith('event:')) {
        eventName = line.slice(6).trim()
        continue
      }
      if (line.startsWith('data:')) {
        dataLines.push(line.slice(5).trimStart())
        continue
      }
    }
  }

  // Trailing frame without final blank line
  flushFrame()
  void eventName
}
