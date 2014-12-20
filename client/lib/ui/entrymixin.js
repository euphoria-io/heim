module.exports = {
  isFocused: function() {
    return document.activeElement == this.refs.input.getDOMNode()
  },

  focusInput: function() {
    var input = this.refs.input || this.refs.nick
    input.getDOMNode().focus()
  },
}
