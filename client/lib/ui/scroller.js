var _ = require('lodash')
var React = require('react')


// http://stackoverflow.com/a/16459606
var isWebkit = 'WebkitAppearance' in document.documentElement.style

function clamp(min, v, max) {
  return Math.min(Math.max(min, v), max)
}

module.exports = React.createClass({
  displayName: 'Scroller',

  componentWillMount: function() {
    window.addEventListener('resize', this.onResize)
    this._checkScroll = _.throttle(this.checkScroll, 150)
    this._finishResize = _.debounce(this.finishResize, 150)
    this._resizing = false
    this._targetInView = false
    this._anchor = null
    this._anchorPos = null
  },

  componentDidMount: function() {
    this.updateAnchorPos()
  },

  componentWillUnmount: function() {
    window.removeEventListener('resize', this.onResize)
  },

  onResize: function() {
    // When resizing, the goal is to keep the entry onscreen in the same
    // position, if possible. This is accomplished by scrolling relative to the
    // previous display height factored into the pos recorded by updateAnchorPos.
    // However, those scrolls will trigger scroll events (and updateAnchorPos in
    // return), which leads to measurement bugs when the window is resizing
    // constantly (I think it's a timing issue). It's much simpler if we
    // disable updateAnchorPos via flag while drag resizing is occurring, so that
    // everything is in reference to the original display height.
    this._resizing = true
    this.scroll()
    this._finishResize()
  },

  finishResize: function() {
    this._resizing = false
    this.updateAnchorPos()
  },

  onScroll: function() {
    this._checkScroll()

    if (!this._resizing) {
      // While resizing the window, we trigger our own scroll events. If we
      // re-measure anchor/target position at this point it may be slightly out
      // of date by the time we scroll again.
      this.updateAnchorPos()
    }
  },

  componentDidUpdate: function() {
    if (!this.scroll()) {
      // If we scrolled, updateAnchorPos will get called after the scroll.
      // If we didn't scroll, we need to updateAnchorPos here.
      this.updateAnchorPos()
    }
    this._checkScroll()
  },

  updateAnchorPos: function() {
    // Record the position of our point of reference. Either the target (if
    // it's in view), or the centermost child element.
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

  checkScroll: function() {
    // Checks based on content / scroll position, to be updated when either
    // changes. Ratelimited to not burden browser while scrolling.
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

  scroll: function(forceTargetInView) {
    // Scroll so our point of interest (target or anchor) is in the right place.
    var node = this.refs.scroller.getDOMNode()
    var displayHeight = node.offsetHeight
    var target = node.querySelector(this.props.target)

    var newScrollTop = null
    if (forceTargetInView || (this._targetInView && this._anchor != target)) {
      // If the target is onscreen, make sure it's within this.props.edgeSpace
      // from the top or bottom.
      var targetPos = node.scrollTop + displayHeight - target.offsetTop
      var clampedPos = clamp(this.props.edgeSpace, targetPos, displayHeight - this.props.edgeSpace)
      newScrollTop = clampedPos - displayHeight + target.offsetTop
    } else if (this._anchor) {
      // Otherwise, try to keep the anchor element in the same place it was when
      // we last saw it via updateAnchorPos.
      newScrollTop = this._anchorPos - displayHeight + this._anchor.offsetTop
    }

    if (newScrollTop != node.scrollTop && displayHeight != node.scrollHeight) {
      if (isWebkit) {
        // Note: mobile Webkit does this funny thing where getting/setting
        // scrollTop doesn't happen promptly during inertial scrolling. It turns
        // out that setting scrollTop inside a requestAnimationFrame callback
        // circumvents this issue.
        window.requestAnimationFrame(function() {
          node.scrollTop = newScrollTop
        })
      } else {
        node.scrollTop = newScrollTop
      }
      return true
    }
  },

  scrollToTarget: function() {
    this.scroll(true)
  },

  render: function() {
    return (
      <div ref="scroller" onScroll={this.onScroll} className={this.props.className}>
        {this.props.children}
      </div>
    )
  },
})
