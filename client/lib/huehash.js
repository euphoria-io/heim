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
    var oval = val
    val = val << 5
    val += oval
    val += normalized.charCodeAt(i)
  }
  val = val << 0
  val += Math.pow(2, 31)
  val = (val + 29) % 360
  cache.data = cache.data.set(text, val)

  return val
}
