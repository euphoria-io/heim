var _ = require('lodash')
var React = require('react')


module.exports = {}

module.exports = React.createClass({
  componentDidMount: function() {
    window.addEventListener('resize', this.onResize)
    this._checkScroll = _.debounce(this.checkScroll, 150, {leading: false})
  },

  componentWillUnmount: function() {
    window.removeEventListener('resize', this.onResize)
  },

  onResize: function() {
    // delay scroll check via debounce
    this._checkScroll()
    this.scroll()
  },

  componentWillUpdate: function() {
    this.checkScroll()
  },

  componentDidUpdate: function() {
    this.scroll()
  },

  checkScroll: function() {
    // via http://blog.vjeux.com/2013/javascript/scroll-position-with-react.html
    var node = this.refs.scroller.getDOMNode()
    this._atBottom = node.scrollTop + node.offsetHeight >= node.scrollHeight
  },

  scroll: function() {
    if (this._atBottom) {
      var node = this.refs.scroller.getDOMNode()
      node.scrollTop = node.scrollHeight
    }
  },

  render: function() {
    return (
      <div ref="scroller" onScroll={this._checkScroll} {...this.props} />
    )
  },
})
