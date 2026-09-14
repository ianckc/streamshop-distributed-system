import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/')({
  component: HomePage,
})

function HomePage() {
  return (
    <main className="page-wrap px-4 py-12">
      <section className="island-shell rise-in rounded-2xl p-6 sm:p-10">
        <p className="island-kicker mb-3">StreamShop</p>
        <h1 className="display-title mb-4 text-4xl font-bold text-[var(--sea-ink)] sm:text-5xl">
          StreamShop
        </h1>
        <p className="m-0 max-w-2xl text-base leading-8 text-[var(--sea-ink-soft)]">
          Public storefront scaffold. Shopping assistant chat, catalog browse,
          checkout, and accounts will land here. Ops chat remains at{' '}
          <code>/admin</code>.
        </p>
      </section>
    </main>
  )
}
