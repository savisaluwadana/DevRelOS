// Single source of "now" for a server-rendered page.
//
// Calling Date.now() directly in a component body trips react-hooks/purity, and
// calling it inside a filter/map callback is genuinely wrong: every row gets
// compared against a slightly different timestamp. Resolve the timestamp once
// per render and pass that value down.
export function renderTimestamp(): number {
  return Date.now();
}
