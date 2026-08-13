# Skins

CCSM ships four skins (Light, Dark, Ocean, Contrast), user-selectable from the header's
🌙/☀️/🌊/⚡ dropdown, persisted in `localStorage` (`ccsm_skin`) and applied with no login required
— it's a client-side preference, not a server setting. This is a guide to adding your own.

## How it works

A skin is a set of CSS custom properties (RGB triplets, no `rgb()` wrapper) scoped under
`[data-skin="<name>"]` on `<html>`. `tailwind.config.js`'s color tokens (`bg`, `fg`, `accent`,
`danger`, `success`, `warn`) all resolve to `rgb(var(--color-x) / <alpha-value>)`, so every
Tailwind utility already in use — `bg-accent`, `text-fg-muted`, `border-accent/50`, etc. —
repaints automatically; you never touch `index.html` or `app.js` class lists.

`app.css` is a **pre-built, purged Tailwind bundle**, committed to the repo (no Tailwind build
step at runtime or in Docker). Adding a skin means editing the CSS variable source
(`static/css/input.css`) and regenerating `app.css` from it — see "Building" below.

## Adding a skin

1. **Pick a name** (lowercase, one word — it's the `data-skin` attribute value and the
   `localStorage` value). This guide uses `myskin`.

2. **Add the variable block** in `static/css/input.css`, alongside the existing ones:

   ```css
   [data-skin="myskin"] {
     --color-bg: 20 20 24;          /* page background */
     --color-bg-card: 30 30 36;     /* cards, header, dropdowns */
     --color-bg-hover: 40 40 48;    /* hover state for bg-hover */
     --color-fg: 230 230 230;       /* body text */
     --color-fg-muted: 150 150 158; /* secondary text */
     --color-accent: 90 140 255;    /* primary actions, links, focus ring */
     --color-accent-hover: 110 160 255;
     --color-accent-muted: 60 100 200;
     --color-danger: 230 70 60;     /* errors, destructive actions */
     --color-danger-hover: 240 100 90;
     --color-success: 60 200 100;   /* confirmations, "active" dots */
     --color-warn: 240 170 20;      /* warnings, pending states */
   }
   ```

   All twelve are required — Tailwind's `<alpha-value>` colors fall through to invalid CSS
   if a variable a utility needs is missing. `:root` (the `dark` skin) has the reference
   values and doubles as the fallback if `data-skin` is ever unset or unrecognized.

3. **Override the neutral grays**, if your background isn't close to the dark skin's. A
   handful of borders/dividers use Tailwind's literal `gray-600/700/800` (not the color-token
   system above, since they're neutral chrome rather than skin identity) — tuned for a dark
   background. On a light or strongly-tinted skin they'll look wrong. Add the same override
   block the `light` and `solarized` skins use:

   ```css
   [data-skin="myskin"] .border-gray-600,
   [data-skin="myskin"] .border-gray-700,
   [data-skin="myskin"] .border-gray-800,
   [data-skin="myskin"] .divide-gray-800 > :not([hidden]) ~ :not([hidden]),
   [data-skin="myskin"] .hover\:border-gray-600:hover,
   [data-skin="myskin"] .hover\:border-gray-700:hover {
     border-color: rgb(60 60 68);
   }
   [data-skin="myskin"] .bg-gray-700 { background-color: rgb(50 50 58); }
   [data-skin="myskin"] .text-gray-400 { color: rgb(150 150 158); }
   ```

   Skip this step if your skin's background is dark and neutral enough that the default
   grays already read fine (e.g. `contrast` reuses them almost as-is).

4. **Terminal pane colors** (optional). The terminal grid (`static/css/terminal-grid.css`) has
   its own variable set — `--term-bg`/`--term-fg`/`--term-select-*` and 32 `--ansi-fg-N`/
   `--ansi-bg-N` variables (the 16-color ANSI palette, foreground and background) — because a
   raw terminal pane needs its own contrast-tuned palette, not just the four semantic status
   colors. `:root` has the dark set; `light` is the only skin that currently overrides it (see
   that block for the full 32-variable shape to copy). Skins built on a dark background
   (`contrast`, `solarized`) inherit the dark terminal look, which reads fine — only add your
   own block if your skin's page background is light/mid-toned enough that the dark pane would
   look like a hole in the UI.

5. **Register the name** in three JS-side places (skin buttons are hand-written in
   `index.html`, not generated from a list, so a name has to be added wherever the current four
   are checked):
   - `static/js/skin-init.js` — the allow-list in the `localStorage` check (this file runs
     before Alpine, applying the saved skin before first paint to avoid a flash of the wrong
     one — see the file's own comment for why it can't be inline `<script>`).
   - `static/js/app.js` — `initSkin()`'s allow-list (same check, re-applied after Alpine
     loads) and an `I18N` label (`skin_myskin`) in both `es` and `en`.
   - `static/index.html` — a new `<button @click="setSkin('myskin')">` in the skin dropdown
     (copy an existing one) and an icon in the header button's `x-text` ternary.

## Building

After editing `input.css` (and `terminal-grid.css` if you touched it), regenerate `app.css`:

```bash
npx tailwindcss@3 -i static/css/input.css -o static/css/app.css --minify
```

`terminal-grid.css` is hand-written CSS (not Tailwind output — see the comment at its top) and
needs no build step; edits to it take effect immediately.

Verify the rebuild didn't drop anything used elsewhere: diff the class selectors between the
old and new `app.css` (`grep -oE '\.[a-zA-Z0-9_\\:/.\[\]%-]+\s*\{' file.css | sort -u`) — the
new file should be a superset. `tailwind.config.js`'s `content` glob already covers
`static/**/*.{html,js}`, so nothing else needs configuring.

## Testing

`e2e/` has a full Playwright harness against stubbed `tmux`/`claude` (see the repo README's
Testing section). For a skin specifically, the fastest check is visual: run the harness
(`cd e2e && bash run-e2e.sh`, or `npx playwright test` for the full suite), open the app, and
in the browser console or a quick Playwright script:

```js
document.documentElement.setAttribute('data-skin', 'myskin');
```

Check the header, the action-bar buttons, a modal (Settings), and — if you touched the
terminal variables — the terminal grid (needs at least one active session) for anything still
using a hardcoded color instead of a token.
