require('./support/setup')
var assert = require('assert')


describe('emoji', function() {
  var emoji = require('../lib/emoji.js')

  describe('toCodePoint', function() {
    it('translates regular emoji', function() {
      assert.equal(emoji.lookupEmojiCharacter('\u25b6'), '25b6')
    })

    it('translates a non-BMP emoji', function() {
      assert.equal(emoji.lookupEmojiCharacter('\ud83d\udd14'), '1f514')
    })

    it('does not translate the (tm) character', function() {
      assert.equal(emoji.lookupEmojiCharacter('\u2122'), null)
    })

    it('does not translate a character without a name', function() {
      assert.equal(emoji.lookupEmojiCharacter('\u00a9'), null)
    })
  })
})
