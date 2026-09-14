import { Link } from '@tanstack/react-router'
import ThemeToggle from './ThemeToggle'

export default function Header() {
  return (
    <header className="sticky top-0 z-50 border-b border-[var(--line)] bg-[var(--header-bg)] px-4 backdrop-blur-lg">
      <nav className="page-wrap flex flex-wrap items-center gap-x-4 gap-y-2 py-3 sm:py-4">
        <Link
          to="/"
          className="display-title text-xl font-bold tracking-tight text-[var(--sea-ink)] no-underline sm:text-2xl"
        >
          StreamShop
        </Link>

        <div className="order-3 flex w-full flex-wrap items-center gap-x-4 gap-y-1 pb-1 text-sm font-semibold sm:order-none sm:w-auto sm:flex-nowrap sm:pb-0">
          <Link
            to="/"
            className="nav-link"
            activeProps={{ className: 'nav-link is-active' }}
          >
            Ask
          </Link>
          <Link
            to="/about"
            className="nav-link"
            activeProps={{ className: 'nav-link is-active' }}
          >
            About
          </Link>
        </div>

        <div className="ml-auto flex items-center gap-1.5 sm:gap-2">
          <ThemeToggle />
        </div>
      </nav>
    </header>
  )
}
