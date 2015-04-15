var Immutable = require('immutable')


module.exports.stripSpaces = function(text) {
  return text.replace(/[^\S]/g, '')
}

module.exports.normalize = function(text) {
  return text.replace(/[^\w_-]/g, '').toLowerCase()
}

var cache = {data: Immutable.Map()}

module.exports.hue = function(text) {
  var cached = cache.data.get(text)
  if (cached) {
    return cached
  }

  var normalized = module.exports.normalize(text)
  if (!normalized.length) {
    normalized = text
  }

  // DJBX33A
  var val = 0
  for (var i = 0; i < normalized.length; i++) {
    if (/\s/.test(normalized[i])) {
      continue
    }
    val = val * 33 + normalized.charCodeAt(i)
  }
  val = (val + 155) % 360
  cache.data = cache.data.set(text, val)

  return val
}
