import _ from 'lodash'
require('string.fromcodepoint')
import unicodeIndex from 'emoji-annotation-to-unicode'
import twemoji from 'twemoji'


const index = _.extend({}, unicodeIndex, {
  '+1': 'plusone',
  'bronze': 'bronze',
  'bronze!?': 'bronze2',
  'bronze?!': 'bronze2',
  'euphoria': 'euphoria',
  'euphoria!': 'euphoric',
  'chromakode': 'chromakode',
  'pewpewpew': 'pewpewpew',
  'leck': 'leck',
  'dealwithit': 'dealwithit',
  'spider': 'spider',
  'indigo_heart': 'indigo_heart',
  'orange_heart': 'orange_heart',
  'bot': 'bot',
  'greenduck': 'greenduck',
  'mobile': unicodeIndex.iphone,
})

delete index.iphone

const names = _.invert(index)

const codes = _.uniq(_.values(index))

const emojiNames = _.filter(_.map(index, (code, name) => code && _.escapeRegExp(name)))
const namesRe = new RegExp(':(' + emojiNames.join('|') + '):', 'g')

function nameToUnicode(name) {
  const code = unicodeIndex[name]
  if (!code) {
    return null
  }
  return String.fromCodePoint(Number.parseInt(code, 16))
}

export function lookupEmojiCharacter(icon) {
  const codePoint = twemoji.convert.toCodePoint(icon)
  if (!names[codePoint]) {
    return null
  }
  // Don't display ™ as an emoji.
  if (codePoint === '2122') {
    return null
  }
  return codePoint
}

export default { index, names, codes, namesRe, nameToUnicode, lookupEmojiCharacter }

