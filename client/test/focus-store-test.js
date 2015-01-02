var support = require('./support/setup')
var assert = require('assert')


describe('focus store', function() {
  var focus = require('../lib/stores/focus')

  beforeEach(function() {
    support.resetStore(focus.store)
  })

  function windowEvent(name) {
    var ev = document.createEvent('Event')
    ev.initEvent(name, false, false)
    window.dispatchEvent(ev)
  }

  it('should initialize with window focused', function() {
    assert.equal(focus.store.getInitialState().windowFocused, true)
  })

  describe('when window focused', function() {
    it('should set window state focused', function(done) {
      support.listenOnce(focus.store, function(state) {
        assert.equal(state.windowFocused, true)
        assert.equal(state.focusChangedAt, Date.now())
        done()
      })

      windowEvent('focus')
    })
  })

  describe('when window blurred', function() {
    it('should set window state not focused', function(done) {
      support.listenOnce(focus.store, function(state) {
        assert.equal(state.windowFocused, false)
        assert.equal(state.focusChangedAt, Date.now())
        done()
      })

      windowEvent('blur')
    })
  })
})
