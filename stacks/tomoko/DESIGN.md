# Tomoko — Design & Architecture Spec

## Core concept & UX philosophy

- **The "Invisible AI" engine**: the interface is a semantic router, not a
  chatbot. Commands translate intent directly into rendered UI actions
  (auto-populated task cards, navigation to pages).
- **No generated text**: the AI never responds with chat bubbles. It only
  renders, creates, or navigates to functional components.
- **Input mechanic**: a Command Palette (Omnibar) is the primary interaction.
  - Desktop: hotkeys (`Cmd+K`), heavily keyboard-navigable.
  - Mobile (PWA): input bar permanently anchored to the bottom; "responses"
    (task cards) materialize directly above the input — no separate form pages.

## Visual identity ("Sophon Unfolding")

- Dark, deep-space ethereal. Highly advanced, soft but slightly ominous,
  incredibly efficient.
- **Logo**: the "Katana Strike" variant — classical profile face cut by the
  Dimension Blade. Wordmark: `public/tomoko_logo.svg`; square icon:
  `public/icon.svg`.
- **Palette**: backgrounds `#0A0E17` (deep space); glassmorphism surfaces;
  **ember orange** accents (`#EB550E`, sampled from the mark) — no neon AI
  blue/green. Things glow.
- **Typography**: `FantasqueSansM Nerd Font` across the board (bundled web
  fallback: TypoPRO Fantasque Sans Mono).

## Theming rules (enforced pattern)

**All tokens live in `app/src/app.css` under `@theme`** — colors, fonts, glow
shadows — plus the composable `glass` / `glow-drop` / `glow-text` utilities.
Components reference tokens exclusively through Tailwind utilities
(`bg-space`, `text-ember`, `shadow-glow`, `font-sans`, `glass`).
**Never hardcode hex codes, font names, or one-off shadows in components.**
Change the theme in one place.

## Tech stack

- Svelte 5 (runes) + Vite, built with bun
- Tailwind CSS v4 (`@tailwindcss/vite`, CSS-first config via `@theme`)
- Bits UI for headless component logic; shadcn-svelte's Command component as
  the Omnibar base (lands with the command routing phase)
- Static dist served by unprivileged nginx; sophon API is same-origin `/api`
