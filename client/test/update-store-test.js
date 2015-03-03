var support = require('./support/setup')
var assert = require('assert')


describe('update store', function() {
  var update = require('../lib/stores/update')

  beforeEach(function() {
    support.resetStore(update.store)
  })

  it('should initialize with default state', function() {
    var initialState = update.store.getInitialState()
    assert.equal(initialState.get('ready'), false)
    assert.equal(initialState.get('currentVersion'), null)
    assert.equal(initialState.get('newVersion'), null)
  })

  describe('on chat state change', function() {
    it('should store the current version if none set')
    describe('if the server version changes', function() {
      it('should store the new version, if not seen before')
      it('should prepare to update if the window is focused')
      it('should not prepare to update if the window is not focused')
    })
  })

  describe('on focus state change', function() {
    it('should prepare to update if the window is focused')
  })

  describe('prepare action', function() {
    it('should prepare an update')
    it('should skip preparing the same update twice')
  })

  describe('setReady action', function() {
    it('should update ready state')
    it('should store update finalize callback')
  })

  describe('perform action', function() {
    it('should call stored update finalize callback')
  })
})
