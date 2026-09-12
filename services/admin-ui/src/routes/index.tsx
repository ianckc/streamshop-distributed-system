import { createFileRoute } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
import { streamAgent, type AgentEvent } from '#/lib/agentStream'

export const Route = createFileRoute('/')({ component: ChatPage })

type ChatRole = 'user' | 'assistant' | 'tool'

type ChatMessage = {
  id: string
  role: ChatRole
  content: string
  streaming?: boolean
}

function newSessionId(): string {
  return crypto.randomUUID()
}

function formatJson(value: unknown): string {
  try {
    return JSON.stringify(value, null, 0)
  } catch {
    return String(value)
  }
}

function ChatPage() {
  const [sessionId, setSessionId] = useState<string>(() => newSessionId())
  const [input, setInput] = useState('')
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [error, setError] = useState<string | null>(null)
  const [streaming, setStreaming] = useState(false)
  const abortRef = useRef<AbortController | null>(null)
  const bottomRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, streaming])

  useEffect(() => {
    return () => {
      abortRef.current?.abort()
    }
  }, [])

  const appendToken = (assistantId: string, chunk: string) => {
    setMessages((prev) =>
      prev.map((m) =>
        m.id === assistantId ? { ...m, content: m.content + chunk } : m,
      ),
    )
  }

  const finishAssistant = (assistantId: string) => {
    setMessages((prev) =>
      prev.map((m) =>
        m.id === assistantId ? { ...m, streaming: false } : m,
      ),
    )
  }

  const pushToolLine = (content: string) => {
    setMessages((prev) => [
      ...prev,
      { id: crypto.randomUUID(), role: 'tool', content },
    ])
  }

  const handleEvent = (assistantId: string, event: AgentEvent) => {
    switch (event.type) {
      case 'TokenChunk':
        appendToken(assistantId, event.payload.content)
        break
      case 'ToolCall':
        pushToolLine(
          `→ ${event.payload.tool_name}(${formatJson(event.payload.arguments)})`,
        )
        break
      case 'ToolResult':
        pushToolLine(
          `← ${event.payload.tool_name}: ${formatJson(event.payload.result)}`,
        )
        break
      case 'Error':
        setError(event.payload.message)
        break
      case 'TurnEnd':
        finishAssistant(assistantId)
        break
      default:
        break
    }
  }

  const stop = () => {
    abortRef.current?.abort()
    abortRef.current = null
    setStreaming(false)
    setMessages((prev) =>
      prev.map((m) => (m.streaming ? { ...m, streaming: false } : m)),
    )
  }

  const send = async () => {
    const message = input.trim()
    const sid = sessionId.trim()
    if (!message || streaming) return

    if (!sid || !/^[a-zA-Z0-9-]+$/.test(sid)) {
      setError('session_id must be alphanumeric or hyphens only')
      return
    }

    setError(null)
    setInput('')

    const userId = crypto.randomUUID()
    const assistantId = crypto.randomUUID()
    setMessages((prev) => [
      ...prev,
      { id: userId, role: 'user', content: message },
      { id: assistantId, role: 'assistant', content: '', streaming: true },
    ])

    const controller = new AbortController()
    abortRef.current = controller
    setStreaming(true)

    try {
      await streamAgent({
        request: { session_id: sid, message },
        signal: controller.signal,
        onEvent: (event) => handleEvent(assistantId, event),
      })
      finishAssistant(assistantId)
    } catch (err) {
      if ((err as Error).name === 'AbortError') {
        finishAssistant(assistantId)
        return
      }
      const msg = err instanceof Error ? err.message : 'Stream failed'
      setError(msg)
      finishAssistant(assistantId)
    } finally {
      setStreaming(false)
      abortRef.current = null
    }
  }

  return (
    <main className="page-wrap flex min-h-[calc(100vh-8rem)] flex-col px-4 pb-8 pt-10">
      <section className="island-shell rise-in relative flex min-h-0 flex-1 flex-col overflow-hidden rounded-[2rem] px-5 py-6 sm:px-8 sm:py-8">
        <div className="pointer-events-none absolute -left-20 -top-24 h-56 w-56 rounded-full bg-[radial-gradient(circle,rgba(79,184,178,0.28),transparent_66%)]" />
        <div className="pointer-events-none absolute -bottom-24 -right-16 h-56 w-56 rounded-full bg-[radial-gradient(circle,rgba(47,106,74,0.16),transparent_66%)]" />

        <header className="relative mb-5 shrink-0">
          <p className="island-kicker mb-2">Ops / SRE copilot</p>
          <h1 className="display-title mb-2 text-3xl font-bold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
            StreamShop agent
          </h1>
          <p className="m-0 max-w-2xl text-sm text-[var(--sea-ink-soft)] sm:text-base">
            Streams SSE from the Rust gateway. Tool calls appear inline. Same
            session id keeps Redis conversation context across turns.
          </p>
        </header>

        <label className="relative mb-4 flex shrink-0 flex-col gap-1.5 text-sm">
          <span className="font-semibold text-[var(--sea-ink)]">Session ID</span>
          <div className="flex flex-wrap gap-2">
            <input
              value={sessionId}
              onChange={(e) => setSessionId(e.target.value)}
              disabled={streaming}
              className="min-w-0 flex-1 rounded-xl border border-[var(--line)] bg-[var(--surface-strong)] px-3 py-2 font-mono text-xs text-[var(--sea-ink)] outline-none focus:border-[var(--lagoon)] sm:text-sm"
              spellCheck={false}
            />
            <button
              type="button"
              disabled={streaming}
              onClick={() => {
                setSessionId(newSessionId())
                setMessages([])
                setError(null)
              }}
              className="rounded-xl border border-[var(--chip-line)] bg-[var(--chip-bg)] px-3 py-2 text-xs font-semibold text-[var(--sea-ink)] transition hover:bg-[var(--link-bg-hover)] disabled:opacity-50"
            >
              New session
            </button>
          </div>
        </label>

        <div className="relative mb-4 min-h-[14rem] flex-1 space-y-3 overflow-y-auto rounded-2xl border border-[var(--line)] bg-[var(--foam)]/50 p-4">
          {messages.length === 0 && (
            <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
              Try: “How are orders looking?” or “Is analytics healthy?”
            </p>
          )}
          {messages.map((m) => (
            <div
              key={m.id}
              className={`flex ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}
            >
              <div
                className={`max-w-[min(100%,36rem)] rounded-2xl px-3.5 py-2.5 text-sm leading-relaxed whitespace-pre-wrap ${
                  m.role === 'user'
                    ? 'bg-[rgba(79,184,178,0.22)] text-[var(--sea-ink)]'
                    : m.role === 'tool'
                      ? 'border border-dashed border-[var(--line)] bg-transparent font-mono text-xs text-[var(--sea-ink-soft)]'
                      : 'border border-[var(--line)] bg-[var(--surface-strong)] text-[var(--sea-ink)]'
                }`}
              >
                {m.content || (m.streaming ? '…' : '')}
                {m.streaming && m.content ? (
                  <span className="ml-0.5 inline-block animate-pulse">▍</span>
                ) : null}
              </div>
            </div>
          ))}
          <div ref={bottomRef} />
        </div>

        {error ? (
          <p
            className="relative mb-3 shrink-0 rounded-xl border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-800 dark:text-red-200"
            role="alert"
          >
            {error}
          </p>
        ) : null}

        <form
          className="relative flex shrink-0 flex-col gap-3 sm:flex-row sm:items-end"
          onSubmit={(e) => {
            e.preventDefault()
            void send()
          }}
        >
          <label className="flex min-w-0 flex-1 flex-col gap-1.5 text-sm">
            <span className="sr-only">Message</span>
            <textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  void send()
                }
              }}
              rows={3}
              disabled={streaming}
              placeholder="Ask about order status, summary, or service health…"
              className="w-full resize-y rounded-2xl border border-[var(--line)] bg-[var(--surface-strong)] px-3.5 py-3 text-sm text-[var(--sea-ink)] outline-none focus:border-[var(--lagoon)] disabled:opacity-60"
            />
          </label>
          <div className="flex gap-2 sm:flex-col">
            {streaming ? (
              <button
                type="button"
                onClick={stop}
                className="rounded-full border border-[rgba(23,58,64,0.2)] bg-white/50 px-5 py-2.5 text-sm font-semibold text-[var(--sea-ink)] transition hover:-translate-y-0.5"
              >
                Stop
              </button>
            ) : (
              <button
                type="submit"
                disabled={!input.trim()}
                className="rounded-full border border-[rgba(50,143,151,0.3)] bg-[rgba(79,184,178,0.14)] px-5 py-2.5 text-sm font-semibold text-[var(--lagoon-deep)] transition hover:-translate-y-0.5 hover:bg-[rgba(79,184,178,0.24)] disabled:opacity-40 disabled:hover:translate-y-0"
              >
                Send
              </button>
            )}
          </div>
        </form>
      </section>
    </main>
  )
}
