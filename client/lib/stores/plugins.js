var _ = require('lodash')
var Reflux = require('reflux')

var Hooks = require('../hooks')


var storeActions = Reflux.createActions([
  'load',
])
_.extend(module.exports, storeActions)

var hooks = module.exports.hooks = new Hooks(
  'page-bottom',
  'sidebar-top',
  'sidebar-bottom',
  'incoming-messages'
)

module.exports.hook = hooks.register.bind(hooks)

module.exports.store = Reflux.createStore({
  listenables: storeActions,

  load: function(roomName) {
    require('../faux-plugins')(roomName)
  },
})
