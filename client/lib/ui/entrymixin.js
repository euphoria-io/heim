module.exports = {
  focus: function(withChar) {
    var node = this.refs.input.getDOMNode()
    if (withChar) {
      node.value += withChar
    }
    node.focus()
  },

  proxyKeyDown: function(ev) {
    if (ev.key == 'Backspace') {
      var node = this.refs.input.getDOMNode()
      node.value = node.value.substr(0, node.value.length - 1)
      node.focus()
      return true
    }
  },

  componentWillUnmount: function() {
    // FIXME: hack to work around Reflux #156.
    this.replaceState = function() {}
  },
}
