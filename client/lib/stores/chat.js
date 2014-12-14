var Reflux = require('reflux')


module.exports = Reflux.createStore({
  listenables: require('../actions'),

  init: function() {
    this.state = {}
  },

  getDefaultData: function() {
    return this.state
  },
})
