module.exports = {
  templateHook: function(name) {
    return Heim.plugins.hooks.run(name, this)
  }
}
