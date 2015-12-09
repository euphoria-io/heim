export default function(el, predicate) {
  let curEl = el
  while (curEl) {
    if (predicate(curEl)) {
      return curEl
    }
    curEl = curEl.parentNode
  }
}
