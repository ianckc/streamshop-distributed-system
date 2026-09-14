import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/about')({
  component: About,
})

function About() {
  return (
    <main className="page-wrap px-4 py-12">
      <section className="island-shell rounded-2xl p-6 sm:p-8">
        <h1 className="display-title mb-4 text-4xl font-bold text-[var(--sea-ink)] sm:text-5xl">
          StreamShop
        </h1>
        <p className="m-0 max-w-2xl text-base leading-8 text-[var(--sea-ink-soft)]">
          A small commerce demo where you can ask about what&apos;s in stock —
          names, prices, and details — before browse, checkout, and accounts
          arrive on this storefront.
        </p>
      </section>
    </main>
  )
}
