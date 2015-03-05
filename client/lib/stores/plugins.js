var _ = require('lodash')
var Reflux = require('reflux')


var storeActions = Reflux.createActions([
  'load',
])
_.extend(module.exports, storeActions)

var hooks = module.exports.hooks = Reflux.createActions({
  'pageBottom': {sync: true},
  'sidebar': {sync: true},
})

module.exports.store = Reflux.createStore({
  listenables: storeActions,

  load: function() {
    require('../fauxplugins')
  },


  triggerHook: function(hookName, props, state) {
    var results = []
    hooks[hookName](results, props, state)
    return results
  },
})

module.exports.triggerHook = module.exports.store.triggerHook
