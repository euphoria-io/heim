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

// Clocks
index['clock115'] = 'clock115'
index['clock145'] = 'clock145'
index['clock215'] = 'clock215'
index['clock245'] = 'clock245'
index['clock315'] = 'clock315'
index['clock345'] = 'clock345'
index['clock415'] = 'clock415'
index['clock445'] = 'clock445'
index['clock515'] = 'clock515'
index['clock545'] = 'clock545'
index['clock615'] = 'clock615'
index['clock645'] = 'clock645'
index['clock715'] = 'clock715'
index['clock745'] = 'clock745'
index['clock815'] = 'clock815'
index['clock845'] = 'clock845'
index['clock915'] = 'clock915'
index['clock945'] = 'clock945'
index['clock1015'] = 'clock1015'
index['clock1045'] = 'clock1045'
index['clock1115'] = 'clock1115'
index['clock1145'] = 'clock1145'
index['clock1215'] = 'clock1215'
index['clock1245'] = 'clock1245'

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
