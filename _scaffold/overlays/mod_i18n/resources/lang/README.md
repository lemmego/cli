# The catalogue

`resources/lang/{locale}/{group}.json`, so `en/messages.json` holds
`messages.greeting`. The file layout *is* the key space — there is no
registry to keep in step with it.

## Plurals are CLDR form names

A message is a plural set when every key in its object is one of the six CLDR
form names — `zero`, `one`, `two`, `few`, `many`, `other` — or an exact match
like `=0`. Anything else is a nesting level:

```json
"items": { "one": ":count item", "other": ":count items" },
"nav":   { "home": "Home", "about": "About" }
```

An object that **mixes** the two is a load error naming the file and the key,
rather than a guess. Guessing is how one message quietly disappears in one
language.

This is also the one place the module deliberately diverges from Laravel.
Laravel's `"apple|apples"` selector is a pre-CLDR switch on the locale
string, and its `choose` returns the **singular** when a language has more
forms than the translator wrote — silently, forever. Named forms cannot do
that: a form a language needs is named, and a missing one falls back to
`other`.

`:count` is bound for you, so `":count items"` needs nothing passed.

## Placeholders

Laravel's `:name`. Values are substituted longest-key-first, which matters
when one placeholder is a prefix of another — `:value` and `:values` both
exist in the framework's own validation messages — and the substitution never
rescans its own output, so a value containing a colon stays literal.

## French is deliberately incomplete

`fr/messages.json` omits `nav`, so the fallback to `en` is exercised by a
real page rather than by a fixture. A missing key returns the key itself and
records one miss per unique key per process: a page that will not render
because a string is missing is worse than a page showing `messages.nav.home`.

## Directory names are canonical tags

`pt-BR`, never `pt-br`. A non-canonical name fails the boot — which is the
bug that passes on every macOS laptop and fails only in the Linux image.
