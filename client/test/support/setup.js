var sinon = require('sinon')

var Immutable = require('immutable')
Immutable.Iterable.noLengthWarning = true

var support = {}

support.setupClock = function() {
  var clock = sinon.useFakeTimers()

  // manually fix Sinon #624 until it updates Lolex to 1.2.0
  Date.now = function() { return Date().getTime() }

  // set up fake clock to work with lodash
  var _ = require('lodash')
  var origDebounce = _.debounce
  _.debounce = _.runInContext(window).debounce

  var origRestore = clock.restore.bind(clock)
  clock.restore = function() {
    _.debounce = origDebounce
    origRestore()
  }

  // remove erroneous entry from coverage listing
  Date.now()

  return clock
}

support.listenOnce = function(listenable, callback) {
  var remove = listenable.listen(function() {
    remove()
    callback.apply(this, arguments)
  })
}

support.resetStore = function(store) {
  store.init()
  store.emitter.removeAllListeners()
}

window.Heim = {
  setFavicon: function() {},
}

module.exports = support
