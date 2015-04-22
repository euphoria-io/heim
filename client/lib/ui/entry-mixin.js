module.exports = {
  focus: function(withChar) {
    var node
    if (this.getDOMNode().contains(uidocument.activeElement)) {
      node = uidocument.activeElement
    } else {
      node = this.refs.input.getDOMNode()
    }
    if (withChar) {
      node.value += withChar
      if (this.onChange) {
        this.onChange()
      }
    }
    node.focus()
  },

  blur: function() {
    this.refs.input.getDOMNode().blur()
  },

  proxyKeyDown: function(ev) {
    var node = this.refs.input.getDOMNode()
    if (ev.key == 'Backspace' && ev.target != node) {
      node.value = node.value.substr(0, node.value.length - 1)
      node.focus()
      return true
    } else {
      return false
    }
  },

  componentWillUnmount: function() {
    // FIXME: hack to work around Reflux #156.
    this.replaceState = function() {}
  },
}
