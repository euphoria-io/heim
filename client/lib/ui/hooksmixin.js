var plugins = require('../stores/plugins')


module.exports = {
  templateHook: function(name) {
    return plugins.hooks.run(name, this)
  }
}
