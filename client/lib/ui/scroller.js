var _ = require('lodash')
var React = require('react')


module.exports = React.createClass({
  displayName: 'Scroller',

  componentWillMount: function() {
    window.addEventListener('resize', this.onResize)
    this._checkScroll = _.debounce(this.checkScroll, 150, {leading: false})
    this._checkPos = _.throttle(this.checkPos, 150)
    this._targetLocked = false
    this._targetPos = 0
    this._lastHeight = 0
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
    this._checkPos()
  },

  componentDidUpdate: function() {
    this.scroll()
    this._checkPos()
  },

  checkScroll: function() {
    // via http://blog.vjeux.com/2013/javascript/scroll-position-with-react.html
    var node = this.refs.scroller.getDOMNode()
    var target = node.querySelector(this.props.target)
    var displayHeight = node.offsetHeight
    this._targetPos = node.scrollTop + displayHeight - target.offsetTop
    this._targetLocked = this._targetPos >= target.offsetHeight && this._targetPos < displayHeight
  },

  checkPos: function() {
    this.checkScroll()

    var node = this.refs.scroller.getDOMNode()

    var displayHeight = node.offsetHeight
    if (this.props.onNearTop && node.scrollTop < displayHeight * 2) {
      this.props.onNearTop()
    }

    if (this.props.onScrollbarSize) {
      var scrollbarWidth = node.offsetWidth - node.clientWidth
      if (scrollbarWidth != this.scrollbarWidth) {
        this.scrollbarWidth = scrollbarWidth
        this.props.onScrollbarSize(scrollbarWidth)
      }
    }
  },

  scroll: function() {
    var node = this.refs.scroller.getDOMNode()
    var height = node.scrollHeight
    if (this._targetLocked) {
      var target = node.querySelector(this.props.target)
      node.scrollTop = Math.max(this.props.bottomSpace, this._targetPos) - node.offsetHeight + target.offsetTop
    } else {
      if (height > this._lastHeight) {
        var delta = height - this._lastHeight
        window.requestAnimationFrame(function() {
          node.scrollTop += delta
        })
      }
    }
    this._lastHeight = height
  },

  render: function() {
    return (
      <div ref="scroller" onScroll={this.onScroll} className={this.props.className}>
        {this.props.children}
      </div>
    )
  },
})
