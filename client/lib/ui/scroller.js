var _ = require('lodash')
var React = require('react')


function clamp(min, v, max) {
  return Math.min(Math.max(min, v), max)
}

module.exports = React.createClass({
  displayName: 'Scroller',

  componentWillMount: function() {
    window.addEventListener('resize', this.onResize)
    this._checkPos = _.throttle(this.checkPos, 150)
    this._finishResize = _.debounce(this.finishResize, 150)
    this._resizing = false
    this._targetInView = false
    this._anchor = null
    this._anchorPos = null
  },

  componentDidMount: function() {
    this.checkScroll()
  },

  componentWillUnmount: function() {
    window.removeEventListener('resize', this.onResize)
  },

  onResize: function() {
    this._resizing = true
    this.scroll()
    this._finishResize()
  },

  finishResize: function() {
    this._resizing = false
    this.checkScroll()
  },

  onScroll: function() {
    this._checkPos()

    if (!this._resizing) {
      // while resizing the window, we trigger our own scroll events. if we
      // re-measure anchor/target position at this point it may be slightly out
      // of date by the time we scroll again.
      this.checkScroll()
    }
  },

  componentDidUpdate: function() {
    if (!this.scroll()) {
      // if we scrolled checkScroll will get called after the scroll.
      this.checkScroll()
    }
    this._checkPos()
  },

  checkScroll: function() {
    var node = this.refs.scroller.getDOMNode()
    var displayHeight = node.offsetHeight

    var target = node.querySelector(this.props.target)
    var targetPos = node.scrollTop + displayHeight - target.offsetTop
    this._targetInView = targetPos >= target.offsetHeight && targetPos < displayHeight

    var anchor
    if (this._targetInView) {
      this._anchor = target
      this._anchorPos = targetPos
    } else {
      var box = this.getDOMNode().getBoundingClientRect()
      anchor = document.elementFromPoint(box.left + box.width / 2, box.top + box.height / 2)
      if (!anchor) {
        console.warn('scroller: unable to find anchor')  // jshint ignore:line
      }
      this._anchor = anchor
      this._anchorPos = anchor && node.scrollTop + displayHeight - anchor.offsetTop
    }
  },

  checkPos: function() {
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
    var displayHeight = node.offsetHeight
    var target = node.querySelector(this.props.target)

    var newScrollTop = null
    if (this._targetInView && this._anchor != target) {
      var targetPos = node.scrollTop + displayHeight - target.offsetTop
      var clampedPos = clamp(this.props.edgeSpace, targetPos, displayHeight - this.props.edgeSpace)
      newScrollTop = clampedPos - displayHeight + target.offsetTop
    } else if (this._anchor) {
      newScrollTop = this._anchorPos - displayHeight + this._anchor.offsetTop
    }

    if (newScrollTop != node.scrollTop) {
      window.requestAnimationFrame(function() {
        node.scrollTop = newScrollTop
      })
      return true
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
