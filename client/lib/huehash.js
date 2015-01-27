var Immutable = require('immutable')


var cache = {data: Immutable.Map()}

module.exports = function(text) {
  var cached = cache.data.get(text)
  if (cached) {
    return cached
  }

  // DJBX33A
  var val = 0
  for (var i = 0; i < text.length; i++) {
    if (/\s/.test(text[i])) {
      continue
    }
    val = val * 33 + text.charCodeAt(i)
  }
  val = (val + 155) % 255
  cache.data = cache.data.set(text, val)

  return val
}
