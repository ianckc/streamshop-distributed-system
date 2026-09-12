import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/about')({
  component: About,
})

function About() {
  return (
    <main className="page-wrap px-4 py-12">
      <section className="island-shell rounded-2xl p-6 sm:p-8">
        <p className="island-kicker mb-2">StreamShop</p>
        <h1 className="display-title mb-3 text-4xl font-bold text-[var(--sea-ink)] sm:text-5xl">
          Ops / SRE copilot
        </h1>
        <p className="m-0 max-w-3xl text-base leading-8 text-[var(--sea-ink-soft)]">
          Chat UI for the Rust agent gateway. Phase A tools query analytics
          order status/summary and service <code>/ready</code> probes. Open this
          page at <code>/admin</code> via Traefik; SSE calls go to{' '}
          <code>/v1/agent/stream</code>.
        </p>
      </section>
    </main>
  )
}
