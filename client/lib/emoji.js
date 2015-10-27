var _ = require('lodash')
require('string.fromcodepoint')
var twemoji = require('twemoji')
var unicodeIndex = require('emoji-annotation-to-unicode')


// (don't nag about object notation)
// jshint -W069

unicodeIndex['mobile'] = unicodeIndex['iphone']
delete unicodeIndex['iphone']

module.exports = {}
var index = module.exports.index = _.clone(unicodeIndex)

index['+1'] = 'plusone'
index['bronze'] = 'bronze'
index['bronze!?'] = 'bronze2'
index['bronze?!'] = 'bronze2'
index['euphoria'] = 'euphoria'
index['euphoria!'] = 'euphoric'
index['chromakode'] = 'chromakode'
index['pewpewpew'] = 'pewpewpew'
index['leck'] = 'leck'
index['dealwithit'] = 'dealwithit'
index['spider'] = 'spider'
index['indigo_heart'] = 'indigo_heart'
index['orange_heart'] = 'orange_heart'
index['bot'] = 'bot'
index['greenduck'] = 'greenduck'

var names = module.exports.names = _.invert(index)

module.exports.codes = _.uniq(_.values(index))

var emojiNames = _.filter(_.map(index, function(code, name) {
  return code && _.escapeRegExp(name)
}))
module.exports.namesRe = new RegExp(':(' + emojiNames.join('|') + '):', 'g')

module.exports.nameToUnicode = function(name) {
  var code = unicodeIndex[name]
  if (!code) {
    return
  }
  return String.fromCodePoint(Number.parseInt(code, 16))
}

module.exports.lookupEmojiCharacter = function(icon) {
  var codePoint = twemoji.convert.toCodePoint(icon)
  if (!names[codePoint]) {
    return null
  }
  // Don't display ™ as an emoji.
  if (codePoint == '2122') {
    return null
  }
  return codePoint
}
