module.exports = {
  focus: function(withChar) {
    var node = this.refs.input.getDOMNode()
    if (withChar) {
      node.value += withChar
    }
    node.focus()
  },

  componentWillUnmount: function() {
    // FIXME: hack to work around Reflux #156.
    this.replaceState = function() {}
  },
}
