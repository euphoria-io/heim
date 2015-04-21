var _ = require('lodash')


module.exports = function EventListeners() {
  this._listeners = []
}

_.extend(module.exports.prototype, {
  addEventListener: function(target, type, listener, useCapture) {
    target.addEventListener(type, listener, useCapture)
    this._listeners.push(_.toArray(arguments))
  },

  removeEventListener: function(target, type, listener, useCapture) {
    target.removeEventListener(type, listener, useCapture)
    var toRemove = _.toArray(arguments)
    _.remove(this._listeners, l => _.isEqual(l, toRemove))
  },

  removeAllEventListeners: function() {
    // iterate in reverse order so removals don't affect iteration
    _.eachRight(this._listeners, l => this.removeEventListener.apply(this, l))
  },
})
