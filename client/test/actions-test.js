var assert = require('assert')


describe('global actions', function() {
  var actions = require('../lib/actions')

  describe('connect action', function() {
    it('should be synchronous', function() {
      assert.equal(actions.connect.sync, true)
    })
  })
})
