var sinon = require('sinon')

var Immutable = require('immutable')
Immutable.Iterable.noLengthWarning = true

var support = {}

// set up fake clock to work with lodash
var _ = require('lodash')
support.clock = sinon.useFakeTimers()
// manually fix Sinon #624 until it updates Lolex to 1.2.0
Date.now = function() { return Date().getTime() }
_.debounce = _.runInContext(window).debounce

// remove erroneous entry from --cover listing
Date.now()

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

module.exports = support
window.uiwindow = window
window.uidocument = document
