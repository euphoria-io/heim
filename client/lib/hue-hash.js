var Immutable = require('immutable')
var emoji = require('./emoji')


module.exports.stripSpaces = function(text) {
  return text.replace(/[^\S]/g, '')
}

module.exports.normalize = function(text) {
  text = text.replace(emoji.namesRe, '')
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

    // multiply val by 33 while constraining within signed 32 bit int range.
    // this keeps the value within Number.MAX_SAFE_INTEGER without throwing out
    // information.
    var origVal = val
    val = val << 5
    val += origVal

    // add the character information to the hash.
    val += charVal
  }

  // cast the result of the final character addition to a 32 bit int.
  val = val << 0

  // add the minimum possible value, to ensure that val is positive (without
  // throwing out information).
  val += Math.pow(2, 31)

  // add the calibration offset and scale within 0-254 (an arbitrary range kept
  // for consistency with prior behavior).
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
