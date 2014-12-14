var React = require('react')


module.exports = {}

module.exports = React.createClass({
  componentDidMount: function() {
    window.addEventListener('resize', this.onResize)
  },

  componentWillUnmount: function() {
    window.removeEventListener('resize', this.onResize)
  },

  onResize: function() {
    this.componentDidUpdate()
  },

  componentWillUpdate: function() {
    // via http://blog.vjeux.com/2013/javascript/scroll-position-with-react.html
    var node = this.refs.scroller.getDOMNode()
    this._atBottom = node.scrollTop + node.offsetHeight >= node.scrollHeight
  },

  componentDidUpdate: function() {
    if (this._atBottom) {
      var node = this.refs.scroller.getDOMNode()
      node.scrollTop = node.scrollHeight
    }
  },

  render: function() {
    return (
      <div ref="scroller" {...this.props} />
    )
  },
})
