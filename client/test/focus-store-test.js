var support = require('./support/setup')
var assert = require('assert')


describe('focus store', function() {
  var focus = require('../lib/stores/focus')

  beforeEach(function() {
    support.resetStore(focus.store)
  })

  it('should initialize with window unfocused', function() {
    var initialState = focus.store.getInitialState()
    assert.equal(initialState.windowFocused, false)
    assert.equal(initialState.focusChangedAt, null)
  })

  describe('when window focused', function() {
    it('should set window state focused', function(done) {
      support.listenOnce(focus.store, function(state) {
        assert.equal(state.windowFocused, true)
        assert.equal(state.focusChangedAt, Date.now())
        done()
      })

      focus.store.windowFocused()
    })
  })

  describe('when window blurred', function() {
    it('should set window state not focused', function(done) {
      support.listenOnce(focus.store, function(state) {
        assert.equal(state.windowFocused, false)
        assert.equal(state.focusChangedAt, Date.now())
        done()
      })

      focus.store.windowBlurred()
    })
  })
})
