var _ = require('lodash')


module.exports = function Hooks(...names) {
  this._hooks = {}
  _.each(names, n => this.create(n))
}

_.extend(module.exports.prototype, {
  create: function(name) {
    this._hooks[name] = []
  },

  register: function(name, callback) {
    this._hooks[name].push(callback)
  },

  run: function(name, context, ...args) {
    return _.map(this._hooks[name], h => h.apply(context, args))
  },
})
