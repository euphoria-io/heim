var _ = require('lodash')
var React = require('react')


module.exports = React.createClass({
  displayName: 'Scroller',

  componentWillMount: function() {
    window.addEventListener('resize', this.onResize)
    this._checkScroll = _.debounce(this.checkScroll, 150, {leading: false})
    this._targetLocked = false
    this._targetPos = 0
  },

  componentDidMount: function() {
    this.checkScroll()
  },

  componentWillUnmount: function() {
    window.removeEventListener('resize', this.onResize)
  },

  onResize: function() {
    // delay scroll check via debounce
    this._checkScroll()
    this.scroll()
  },

  onScroll: function() {
    this._checkScroll()
    this.checkScrollbar()
  },

  componentDidUpdate: function() {
    this._checkScroll()
    this.scroll()
  },

  checkScroll: function() {
    // via http://blog.vjeux.com/2013/javascript/scroll-position-with-react.html
    var node = this.refs.scroller.getDOMNode()
    var target = node.querySelector(this.props.target)
    this._targetPos = node.scrollTop + node.offsetHeight - target.offsetTop
    this._targetLocked = this._targetPos >= target.offsetHeight && this._targetPos < node.offsetHeight
  },

  checkScrollbar: function() {
    if (this.props.onScrollbarSize) {
      var node = this.refs.scroller.getDOMNode()
      var scrollbarWidth = node.offsetWidth - node.clientWidth
      if (scrollbarWidth != this.scrollbarWidth) {
        this.scrollbarWidth = scrollbarWidth
        this.props.onScrollbarSize(scrollbarWidth)
      }
    }
  },

  scroll: function() {
    if (this._targetLocked) {
      var node = this.refs.scroller.getDOMNode()
      var target = node.querySelector(this.props.target)
      node.scrollTop = Math.max(this.props.bottomSpace, this._targetPos) - node.offsetHeight + target.offsetTop
    }
  },

  render: function() {
    return (
      <div ref="scroller" onScroll={this.onScroll} className={this.props.className}>
        {this.props.children}
      </div>
    )
  },
})
