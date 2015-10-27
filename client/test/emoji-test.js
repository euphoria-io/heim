require('./support/setup')
var assert = require('assert')


describe('emoji', function() {
  var emoji = require('../lib/emoji.js')

  describe('toCodePoint', function() {
    it('translates regular emoji', function() {
      // U+25B6 BLACK RIGHT-POINTING TRIANGLE (:arrow_forward:)
      assert.equal(emoji.lookupEmojiCharacter('\u25b6'), '25b6')
    })

    it('translates a non-BMP emoji', function() {
      // U+1F514 BELL (:bell:)
      assert.equal(emoji.lookupEmojiCharacter('\ud83d\udd14'), '1f514')
    })

    it('does not translate the (tm) character', function() {
      // U+2122 TRADE MARK SIGN (:tm:)
      assert.equal(emoji.lookupEmojiCharacter('\u2122'), null)
    })

    it('does not translate a character without a name', function() {
      // U+00A9 COPYRIGHT SIGN (no emoji)
      assert.equal(emoji.lookupEmojiCharacter('\u00a9'), null)
    })
  })
})
