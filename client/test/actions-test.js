var assert = require('assert')


describe('global actions', function() {
  var actions = require('../lib/actions')

  describe('setup action', function() {
    it('should be synchronous', function() {
      assert.equal(actions.setup.sync, true)
    })
  })
})
