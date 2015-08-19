module.exports = function(el, predicate) {
  while (el) {
    if (predicate(el)) {
      return el
    }
    el = el.parentNode
  }
}
