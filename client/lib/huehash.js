var Immutable = require('immutable')


module.exports.stripSpaces = function(text) {
  return text.replace(/[^\S]/g, '')
}

module.exports.normalize = function(text) {
  return text.replace(/[^\w_-]/g, '').toLowerCase()
}

function hueHash(text, offset) {
  offset = offset || 0

  // DJBX33A-ish
  var val = 0
  for (var i = 0; i < text.length; i++) {
    // scramble char codes across [0-255]
    // prime multiple chosen so @greenie can green, and @redtaboo red.
    var charVal = (text.charCodeAt(i) * 439) % 256
    var origVal = val
    val = val << 5
    val += origVal
    val += charVal
  }
  val = val << 0
  val += Math.pow(2, 31)

  return (val + offset) % 255
}

var cache = {data: Immutable.Map()}

var greenieOffset = 148 - hueHash('greenie')

module.exports.hue = function(text) {
  var cached = cache.data.get(text)
  if (cached) {
    return cached
  }

  var normalized = module.exports.normalize(text)
  if (!normalized.length) {
    normalized = text
  }

  var val = hueHash(normalized, greenieOffset)
  cache.data = cache.data.set(text, val)
  return val
}
