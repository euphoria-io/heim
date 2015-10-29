export default function walkForward(startEl, endEl, callback) {
  let el = startEl
  while (el) {
    callback(el)

    if (el.childNodes.length) {
      el = el.childNodes[0]
    } else {
      while (!el.nextSibling) {
        el = el.parentNode
        if (!el || el === endEl) {
          return
        }
      }

      el = el.nextSibling
    }
  }
}
